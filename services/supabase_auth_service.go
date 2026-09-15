package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"refer_api/models"
	"refer_api/repository"
	"refer_api/supabase"
	"refer_api/utils"

	"golang.org/x/crypto/bcrypt"
)

// AuthSession is the token pair returned to Flutter.
// Field names stay {accessToken, refreshToken} at the HTTP layer.
type AuthSession struct {
	AccessToken  string
	RefreshToken string
}

// SessionAuthService is implemented by auth services that can return real
// Supabase sessions. Handlers type-assert to it and fall back to the legacy
// mock-refresh behavior when it is not available.
type SessionAuthService interface {
	AuthService
	RegisterWithSession(name, email, password string, keepUpdated bool) (*models.User, AuthSession, error)
	LoginWithSession(email, password string) (*models.User, AuthSession, error)
	VerifyEmailWithSession(email, code string) (*models.User, AuthSession, error)
	ResendVerification(email string) error
	RefreshSession(refreshToken string) (AuthSession, error)
	Logout(accessToken string) error
}

// ErrEmailNotConfirmed is returned when the account exists but its email OTP
// has not been verified yet. Handlers map it to 403 {code: EMAIL_NOT_VERIFIED}
// on login, and to 201 {needsVerification:true} on register.
var ErrEmailNotConfirmed = errors.New("email not verified")

// supabaseAuthService adapts Supabase Auth behind the existing AuthService
// contract so Flutter sees zero changes.
type supabaseAuthService struct {
	userRepo   repository.UserRepository
	supabase   *supabase.Client
	cfg        supabase.Config
	fallback   AuthService
	configRepo repository.ConfigRepository
}

// NewSupabaseAuthService builds the Supabase-backed AuthService.
// fallback is used for transitional dual-write (legacy bcrypt + reset codes)
// so existing users keep working before the data migration (Fase 2).
// configRepo is optional (variadic for backwards-compat in tests): when
// provided, los toggles smtp_enabled/signup_email_enabled/recovery_email_enabled
// de app_settings se respetan (si el operador los deshabilita en refer_web,
// no se envían correos de registro/recuperación).
func NewSupabaseAuthService(userRepo repository.UserRepository, client *supabase.Client, cfg supabase.Config, configRepo ...repository.ConfigRepository) SessionAuthService {
	svc := &supabaseAuthService{
		userRepo: userRepo,
		supabase: client,
		cfg:      cfg,
		fallback: NewAuthService(userRepo),
	}
	if len(configRepo) > 0 {
		svc.configRepo = configRepo[0]
	}
	return svc
}

// authSettings lee app_settings (best-effort): ante cualquier fallo devuelve
// nil para que los helpers apliquen los defaults (true = comportamiento actual).
// Fase 0: reusa el cache de 30s de ConfigService para no golpear DB en cada
// register/login/forgot/resend.
func (s *supabaseAuthService) authSettings() map[string]string {
	if cached, ok := getCachedSettings(); ok {
		return cached
	}
	if s.configRepo == nil {
		return nil
	}
	rows, err := s.configRepo.FindAll()
	if err != nil {
		return nil
	}
	settings := map[string]string{}
	for _, row := range rows {
		settings[row.Key] = row.Value
	}
	if len(settings) > 0 {
		setCachedSettings(settings)
	}
	return settings
}

func (s *supabaseAuthService) signupEmailEnabled() bool {
	return SignupEmailEnabled(s.authSettings())
}

func (s *supabaseAuthService) recoveryEmailEnabled() bool {
	return RecoveryEmailEnabled(s.authSettings())
}

func roleNames(roles []models.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.Name)
	}
	return out
}

func (s *supabaseAuthService) ensureUserRole() (*models.Role, error) {
	role, err := s.userRepo.FindRoleByName("USER")
	if err == nil {
		return role, nil
	}
	role = &models.Role{Name: "USER"}
	if createErr := s.userRepo.CreateRole(role); createErr != nil {
		// Re-read in case of race; fall back to in-memory role.
		if found, findErr := s.userRepo.FindRoleByName("USER"); findErr == nil {
			return found, nil
		}
		return role, nil
	}
	return role, nil
}

// Register creates auth.users via Supabase and mirrors the profile in
// public.users with the SAME id (auth.users.id). IDs are already UUID
// char(36) in this codebase, so no INT->UUID mapping is needed.
func (s *supabaseAuthService) Register(name, email, password string, keepUpdated bool) (*models.User, string, error) {
	user, session, err := s.RegisterWithSession(name, email, password, keepUpdated)
	if err != nil {
		return nil, "", err
	}
	return user, session.AccessToken, nil
}

func (s *supabaseAuthService) RegisterWithSession(name, email, password string, keepUpdated bool) (*models.User, AuthSession, error) {
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, AuthSession{}, errors.New("email already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, AuthSession{}, err
	}

	// Si el operador deshabilitó el correo de registro (smtp_enabled=false o
	// signup_email_enabled=false en app_settings), no se debe aplicar SMTP:
	// crear/confirmar vía Admin API (email_confirm=true) y devolver sesión
	// completa sin exigir OTP.
	if !s.signupEmailEnabled() && s.cfg.HasAdminKey() {
		if user, session, err := s.registerWithoutEmail(name, email, password, keepUpdated); err == nil {
			return user, session, nil
		}
		// Si el admin-flow falla (p.ej. ya existe en Auth), continuar con el
		// flujo normal + auto-confirmación abajo.
	}

	session, err := s.supabase.SignUp(email, password, name)
	if err != nil {
		// AUTOCONFIRM=false: the auth.users row exists but the email is not
		// verified yet. Still mirror the profile in public.users (same id)
		// and surface ErrEmailNotConfirmed so the handler answers
		// 201 {needsVerification:true} instead of a 500.
		// Signup only reaches this branch with a usable user id (carried in
		// session); any other failure (e.g. SMTP misconfigured) is returned
		// as-is so the real cause surfaces instead of a confusing 500.
		if errors.Is(err, supabase.ErrEmailNotConfirmed) || supabase.IsNotConfirmedErr(err) {
			// Correo de registro deshabilitado: auto-confirmar y devolver
			// sesión completa en vez de needsVerification.
			if !s.signupEmailEnabled() && s.cfg.HasAdminKey() {
				authID := session.UserID
				if authID == "" {
					if resolved, rErr := s.supabase.AdminGetUserByEmail(email); rErr == nil && resolved != "" {
						authID = resolved
					}
				}
				if authID != "" {
					if user, sess, cErr := s.confirmAndSignIn(name, email, password, keepUpdated, authID); cErr == nil {
						return user, sess, nil
					}
				}
			}
			return s.registerPendingUser(name, email, keepUpdated, session, err)
		}
		// Mailer failures (e.g. SMTP 535 "Missing password" when SMTP_PASS
		// is empty) carry ErrEmailServiceUnavailable via errors.Is so the
		// handler can answer 503. Propagate unchanged... salvo que el
		// operador haya deshabilitado el correo: en ese caso auto-confirmar.
		if errors.Is(err, supabase.ErrEmailServiceUnavailable) || supabase.IsMailerErr(err) {
			if !s.signupEmailEnabled() && s.cfg.HasAdminKey() {
				if resolved, rErr := s.supabase.AdminGetUserByEmail(email); rErr == nil && resolved != "" {
					if user, sess, cErr := s.confirmAndSignIn(name, email, password, keepUpdated, resolved); cErr == nil {
						return user, sess, nil
					}
				}
			}
			return nil, AuthSession{}, err
		}
		// Orphan recovery: a previous signup may have created the auth.users
		// row but failed afterwards (e.g. SMTP outage before the mirror
		// insert), so public.users has no row while Auth says "already
		// registered". Mirror the profile and report needs-verification
		// instead of a 500, so the user can continue via resend/verify once
		// mail is fixed.
		if isAlreadyRegisteredErr(err) && s.cfg.HasAdminKey() {
			if resolved, resolveErr := s.supabase.AdminGetUserByEmail(email); resolveErr == nil && resolved != "" {
				// Si el correo de registro está deshabilitado, auto-confirmar
				// para no dejar al usuario atascado en needsVerification.
				if !s.signupEmailEnabled() {
					if user, sess, cErr := s.confirmAndSignIn(name, email, password, keepUpdated, resolved); cErr == nil {
						return user, sess, nil
					}
				}
				return s.registerPendingUser(name, email, keepUpdated, supabase.Session{UserID: resolved, Email: email}, err)
			}
		}
		return nil, AuthSession{}, err
	}
	if session.UserID == "" {
		return nil, AuthSession{}, errors.New("supabase signup did not return a user id")
	}

	userRole, err := s.ensureUserRole()
	if err != nil {
		_ = s.supabase.AdminDeleteUser(session.UserID)
		return nil, AuthSession{}, err
	}

	user := &models.User{
		ID:          session.UserID,
		Name:        name,
		Email:       email,
		Password:    "", // Passwords live in Supabase Auth only for new users.
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if email == "" {
		user.Email = session.Email
	}

	if err := s.userRepo.Create(user); err != nil {
		// Avoid orphan auth.users when the profile insert fails.
		_ = s.supabase.AdminDeleteUser(session.UserID)
		return nil, AuthSession{}, err
	}

	return user, AuthSession{AccessToken: session.AccessToken, RefreshToken: session.RefreshToken}, nil
}

// isAlreadyRegisteredErr reports whether err means the email already has an
// auth.users row (GoTrue answers 400/422 "User already registered"). Used to
// recover orphan signups: a previous attempt created the auth row but failed
// before the public.users mirror (e.g. SMTP outage).
func isAlreadyRegisteredErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already registered") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "user already")
}

// registerPendingUser mirrors the public.users profile for an auth.users row
// whose email is not verified yet (AUTOCONFIRM=false). The auth user id is
// resolved from the signup session, falling back to an admin email lookup
// when the session carries no id. When no id can be resolved (e.g. the signup
// itself failed and rolled back, like an SMTP outage), signupErr is returned
// unchanged so the real cause surfaces. Otherwise it returns
// ErrEmailNotConfirmed so the handler answers 201 {needsVerification:true}.
func (s *supabaseAuthService) registerPendingUser(name, email string, keepUpdated bool, session supabase.Session, signupErr error) (*models.User, AuthSession, error) {
	authUserID := session.UserID
	if authUserID == "" && s.cfg.HasAdminKey() {
		if resolved, resolveErr := s.supabase.AdminGetUserByEmail(email); resolveErr == nil && resolved != "" {
			authUserID = resolved
		}
	}
	if authUserID == "" {
		return nil, AuthSession{}, signupErr
	}

	userRole, err := s.ensureUserRole()
	if err != nil {
		return nil, AuthSession{}, err
	}

	user := &models.User{
		ID:          authUserID,
		Name:        name,
		Email:       email,
		Password:    "", // Passwords live in Supabase Auth only for new users.
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if email == "" {
		user.Email = session.Email
	}

	if err := s.userRepo.Create(user); err != nil {
		// Profile may already exist (e.g. retry after a partial failure):
		// still report "needs verification" instead of a 500.
		if existing, findErr := s.userRepo.FindByEmail(user.Email); findErr == nil && existing != nil {
			return existing, AuthSession{}, ErrEmailNotConfirmed
		}
		return nil, AuthSession{}, err
	}

	return user, AuthSession{}, ErrEmailNotConfirmed
}

// registerWithoutEmail crea el usuario vía Admin API con email_confirm=true
// (no pasa por SMTP) y devuelve la sesión completa. Solo se usa cuando el
// operador deshabilitó el correo de registro en el módulo de configuraciones.
func (s *supabaseAuthService) registerWithoutEmail(name, email, password string, keepUpdated bool) (*models.User, AuthSession, error) {
	authUserID, err := s.supabase.AdminCreateUser(email, password, name, "", true)
	if err != nil {
		return nil, AuthSession{}, err
	}
	sess, err := s.supabase.SignIn(email, password)
	if err != nil {
		// Confirmado pero sin sesión (raro): espejar perfil y devolver error
		// para que el login posterior funcione.
		_ = s.supabase.AdminConfirmEmail(authUserID)
		if sess2, err2 := s.supabase.SignIn(email, password); err2 == nil {
			sess = sess2
		} else {
			_, _ = s.mirrorUser(authUserID, name, email, keepUpdated)
			return nil, AuthSession{}, err
		}
	}
	if sess.UserID == "" {
		sess.UserID = authUserID
	}
	user, err := s.mirrorUser(sess.UserID, name, email, keepUpdated)
	if err != nil {
		_ = s.supabase.AdminDeleteUser(sess.UserID)
		return nil, AuthSession{}, err
	}
	return user, AuthSession{AccessToken: sess.AccessToken, RefreshToken: sess.RefreshToken}, nil
}

// confirmAndSignIn auto-confirma un auth.users existente, espeja public.users
// e inicia sesión con el password del registro para devolver sesión completa.
// Usado cuando el correo de registro está deshabilitado y el signup quedó
// pendiente o falló por SMTP.
func (s *supabaseAuthService) confirmAndSignIn(name, email, password string, keepUpdated bool, authUserID string) (*models.User, AuthSession, error) {
	if authUserID == "" || !s.cfg.HasAdminKey() {
		return nil, AuthSession{}, errors.New("cannot auto-confirm without admin key")
	}
	if err := s.supabase.AdminConfirmEmail(authUserID); err != nil {
		return nil, AuthSession{}, err
	}
	sess, err := s.supabase.SignIn(email, password)
	if err != nil {
		// Confirmado pero SignIn falló: espejar para que el login posterior
		// funcione y propagar el error original.
		_, _ = s.mirrorUser(authUserID, name, email, keepUpdated)
		return nil, AuthSession{}, err
	}
	if sess.UserID == "" {
		sess.UserID = authUserID
	}
	user, err := s.mirrorUser(sess.UserID, name, email, keepUpdated)
	if err != nil {
		return nil, AuthSession{}, err
	}
	return user, AuthSession{AccessToken: sess.AccessToken, RefreshToken: sess.RefreshToken}, nil
}

// mirrorUser crea el perfil public.users (id = auth.users.id) si no existe.
func (s *supabaseAuthService) mirrorUser(authUserID, name, email string, keepUpdated bool) (*models.User, error) {
	if existing, err := s.userRepo.FindByEmail(email); err == nil && existing != nil {
		return existing, nil
	}
	userRole, err := s.ensureUserRole()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = email
		if at := strings.Index(email, "@"); at > 0 {
			name = email[:at]
		}
	}
	user := &models.User{
		ID:          authUserID,
		Name:        name,
		Email:       email,
		Password:    "",
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.userRepo.Create(user); err != nil {
		if existing, findErr := s.userRepo.FindByEmail(email); findErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return user, nil
}

// Login validates credentials against Supabase Auth, then loads the business
// profile (roles, photoUrl, ...) from public.users.
func (s *supabaseAuthService) Login(email, password string) (*models.User, string, error) {
	user, session, err := s.LoginWithSession(email, password)
	if err != nil {
		return nil, "", err
	}
	return user, session.AccessToken, nil
}

func (s *supabaseAuthService) LoginWithSession(email, password string) (*models.User, AuthSession, error) {
	session, err := s.supabase.SignIn(email, password)
	if err != nil {
		// Unverified accounts stay blocked even when the legacy bcrypt
		// password would still match (dual-write): verification is required.
		if errors.Is(err, supabase.ErrEmailNotConfirmed) || supabase.IsNotConfirmedErr(err) {
			return nil, AuthSession{}, ErrEmailNotConfirmed
		}
		// Transitional fallback: users not yet migrated to Supabase Auth keep
		// working with the legacy bcrypt password until Fase 2 migrates them.
		// We still return a legacy-shaped session (no real refresh token).
		if legacyUser, legacyToken, legacyErr := s.fallback.Login(email, password); legacyErr == nil {
			return legacyUser, AuthSession{AccessToken: legacyToken, RefreshToken: ""}, nil
		}
		msg := err.Error()
		if strings.Contains(msg, "400") || strings.Contains(msg, "401") || strings.Contains(msg, "Invalid") || strings.Contains(msg, "invalid") {
			return nil, AuthSession{}, errors.New("invalid credentials")
		}
		return nil, AuthSession{}, err
	}

	// Canonical profile: prefer lookup by email so pre-migration users
	// (public.users.id = legacy UUID) keep their FKs working even when
	// auth.users.id differs. The id cutover happens in Fase 2.
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// Auth exists in Supabase but profile row is missing: recreate it
		// from the Supabase identity to avoid inconsistent states.
		userRole, roleErr := s.ensureUserRole()
		if roleErr != nil {
			return nil, AuthSession{}, roleErr
		}
		name := email
		if at := strings.Index(email, "@"); at > 0 {
			name = email[:at]
		}
		user = &models.User{
			ID:        session.UserID,
			Name:      name,
			Email:     email,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Roles:     []models.Role{*userRole},
		}
		if createErr := s.userRepo.Create(user); createErr != nil {
			return nil, AuthSession{}, createErr
		}
	}

	return user, AuthSession{AccessToken: session.AccessToken, RefreshToken: session.RefreshToken}, nil
}

// VerifyEmailWithSession confirms the account with the 6-digit email OTP and
// returns a full session on success. The public.users mirror row is created
// when missing (same pattern as LoginWithSession) so verification never
// leaves the account half-provisioned.
func (s *supabaseAuthService) VerifyEmailWithSession(email, code string) (*models.User, AuthSession, error) {
	if email == "" || code == "" {
		return nil, AuthSession{}, errors.New("email and code are required")
	}
	session, err := s.supabase.VerifyOtp(email, code, supabase.VerifyTypeSignup)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "400") || strings.Contains(msg, "401") || strings.Contains(msg, "422") ||
			strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(strings.ToLower(msg), "expired") {
			return nil, AuthSession{}, errors.New("invalid or expired verification code")
		}
		return nil, AuthSession{}, err
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		userRole, roleErr := s.ensureUserRole()
		if roleErr != nil {
			return nil, AuthSession{}, roleErr
		}
		name := email
		if at := strings.Index(email, "@"); at > 0 {
			name = email[:at]
		}
		user = &models.User{
			ID:        session.UserID,
			Name:      name,
			Email:     email,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Roles:     []models.Role{*userRole},
		}
		if createErr := s.userRepo.Create(user); createErr != nil {
			return nil, AuthSession{}, createErr
		}
	}

	return user, AuthSession{AccessToken: session.AccessToken, RefreshToken: session.RefreshToken}, nil
}

// VerifyEmail adapts VerifyEmailWithSession to the AuthService contract.
func (s *supabaseAuthService) VerifyEmail(email, code string) (*models.User, string, error) {
	user, session, err := s.VerifyEmailWithSession(email, code)
	if err != nil {
		return nil, "", err
	}
	return user, session.AccessToken, nil
}

// ResendVerification re-sends the signup OTP email via Supabase Auth (GoTrue
// applies its own rate limit). The account must exist locally; unknown emails
// get "user not found" so we never leak which emails are registered in Auth.
// Si el operador deshabilitó el correo de registro, devuelve ErrAuthEmailDisabled
// para que no se aplique SMTP.
func (s *supabaseAuthService) ResendVerification(email string) error {
	if !s.signupEmailEnabled() {
		return ErrAuthEmailDisabled
	}
	if email == "" {
		return errors.New("email is required")
	}
	if _, err := s.userRepo.FindByEmail(email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Anti-enumeration: same success as for known users.
			return nil
		}
		return err
	}
	return s.supabase.ResendSignup(email)
}

// RefreshSession exchanges a Supabase refresh token for a new pair.
func (s *supabaseAuthService) RefreshSession(refreshToken string) (AuthSession, error) {
	if refreshToken == "" {
		return AuthSession{}, errors.New("refresh token is required")
	}
	session, err := s.supabase.Refresh(refreshToken)
	if err != nil {
		return AuthSession{}, err
	}
	return AuthSession{AccessToken: session.AccessToken, RefreshToken: session.RefreshToken}, nil
}

// Logout revokes the Supabase session. Errors for already-expired sessions
// are swallowed so Flutter logout never fails visibly.
func (s *supabaseAuthService) Logout(accessToken string) error {
	if accessToken == "" {
		return nil
	}
	if err := s.supabase.Logout(accessToken); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "invalid") || strings.Contains(msg, "expired") {
			return nil
		}
		return err
	}
	return nil
}

// ChangePassword updates both Supabase Auth (source of truth) and the legacy
// bcrypt column (transitional dual-write) so old sessions keep working.
func (s *supabaseAuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Resolve the Supabase auth.users.id: normally == public.users.id for new
	// users, but pre-migration users may differ -> resolve by email via admin.
	authUserID := user.ID
	if s.cfg.HasAdminKey() {
		if resolved, resolveErr := s.supabase.AdminGetUserByEmail(user.Email); resolveErr == nil && resolved != "" {
			authUserID = resolved
		}
	}

	// Verify current password: try Supabase first, then legacy bcrypt.
	supabaseOK := false
	if _, signInErr := s.supabase.SignIn(user.Email, currentPassword); signInErr == nil {
		supabaseOK = true
	}
	if !supabaseOK {
		if legacyErr := s.fallback.ChangePassword(userID, currentPassword, newPassword); legacyErr != nil {
			// Preserve legacy error semantics for Flutter.
			return legacyErr
		}
		// Legacy check passed; still push the new password to Supabase.
		if s.cfg.HasAdminKey() {
			_ = s.supabase.AdminUpdatePassword(authUserID, newPassword)
		}
		return nil
	}

	if s.cfg.HasAdminKey() {
		if err := s.supabase.AdminUpdatePassword(authUserID, newPassword); err != nil {
			return err
		}
	}
	// Dual-write legacy bcrypt hash (best effort) so fallback login keeps
	// working during the transition. Never fail the request because of this.
	if hashed, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword), 10); hashErr == nil {
		_ = s.userRepo.UpdatePassword(user.ID, string(hashed))
	}
	return nil
}

// ForgotPassword keeps the Flutter {email}->{message} contract: it preserves
// the legacy 6-digit code flow AND best-effort triggers Supabase recovery email.
// Si el operador deshabilitó el correo de recuperación (smtp_enabled=false o
// recovery_email_enabled=false), no se aplica SMTP: se genera el código legacy
// pero se omite el Recover de Supabase.
func (s *supabaseAuthService) ForgotPassword(email string) error {
	if err := s.fallback.ForgotPassword(email); err != nil {
		return err
	}
	if !s.recoveryEmailEnabled() {
		return nil
	}
	// Best effort: if the user also exists in Supabase, send its email too.
	// Never fail the request because of this (Flutter expects 200 when the
	// user exists locally).
	_ = s.supabase.Recover(email)
	return nil
}

// ResetPassword validates the legacy code (Flutter contract) and syncs the
// new password to Supabase Auth when possible.
func (s *supabaseAuthService) ResetPassword(email, code, newPassword string) error {
	if err := s.fallback.ResetPassword(email, code, newPassword); err != nil {
		return err
	}
	if s.cfg.HasAdminKey() {
		user, err := s.userRepo.FindByEmail(email)
		if err == nil && user != nil {
			authUserID := user.ID
			if resolved, resolveErr := s.supabase.AdminGetUserByEmail(email); resolveErr == nil && resolved != "" {
				authUserID = resolved
			}
			_ = s.supabase.AdminUpdatePassword(authUserID, newPassword)
		}
	}
	return nil
}

// ValidateToken resolves a Bearer token (Supabase or legacy) to a user id.
func (s *supabaseAuthService) ValidateToken(token string) (string, error) {
	claims, err := utils.ValidateAnyToken(token)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
