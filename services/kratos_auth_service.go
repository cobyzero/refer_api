package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"refer_api/auth"
	"refer_api/models"
	"refer_api/repository"
	"refer_api/utils"
)

// ErrEmailNotConfirmed is returned when the account exists but its email OTP
// has not been verified yet. Handlers map it to 403 {code: EMAIL_NOT_VERIFIED}
// on login, and to 201 {needsVerification:true} on register.
var ErrEmailNotConfirmed = errors.New("email not verified")

// kratosAuthService adapts Ory Kratos behind the existing AuthService
// contract so Flutter sees zero changes.
//
// Trust model: Kratos owns credentials (argon2 password hashing, 6-digit
// email OTP codes, verification state). Go never stores passwords; it mints
// short-lived API JWTs after Kratos confirms the identity, plus opaque
// refresh tokens (sha256 in PG, rotated on use). The public.users mirror row
// keeps the SAME id as the Kratos identity (UUID char(36)).
type kratosAuthService struct {
	userRepo   repository.UserRepository
	kratos     *auth.Client
	cfg        auth.Config
	configRepo repository.ConfigRepository
}

// NewKratosAuthService builds the Kratos-backed AuthService.
// configRepo is optional (variadic for backwards-compat in tests): when
// provided, los toggles smtp_enabled/signup_email_enabled/recovery_email_enabled
// de app_settings se respetan (si el operador los deshabilita en refer_web,
// no se envían correos de registro/recuperación).
func NewKratosAuthService(userRepo repository.UserRepository, client *auth.Client, cfg auth.Config, configRepo ...repository.ConfigRepository) SessionAuthService {
	svc := &kratosAuthService{
		userRepo: userRepo,
		kratos:   client,
		cfg:      cfg,
	}
	if len(configRepo) > 0 {
		svc.configRepo = configRepo[0]
	}
	return svc
}

// authSettings lee app_settings (best-effort): ante cualquier fallo devuelve
// nil para que los helpers apliquen los defaults (true = comportamiento actual).
// Reusa el cache de 30s de ConfigService para no golpear DB en cada
// register/login/forgot/resend.
func (s *kratosAuthService) authSettings() map[string]string {
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

func (s *kratosAuthService) signupEmailEnabled() bool {
	return SignupEmailEnabled(s.authSettings())
}

func (s *kratosAuthService) recoveryEmailEnabled() bool {
	return RecoveryEmailEnabled(s.authSettings())
}

func roleNames(roles []models.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.Name)
	}
	return out
}

func (s *kratosAuthService) ensureUserRole() (*models.Role, error) {
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

// mintPair mints the API session pair for an identity confirmed by Kratos.
func (s *kratosAuthService) mintPair(identityID, email string) (AuthSession, error) {
	access, err := utils.MintToken(identityID, email)
	if err != nil {
		return AuthSession{}, err
	}
	refresh, err := auth.IssueRefreshToken(identityID, s.cfg.RefreshTTLDays)
	if err != nil {
		return AuthSession{}, err
	}
	return AuthSession{AccessToken: access, RefreshToken: refresh}, nil
}

// Register creates the identity in Kratos and mirrors the profile in
// public.users with the SAME id (Kratos identity id). IDs are already UUID
// char(36) in this codebase, so no INT->UUID mapping is needed.
func (s *kratosAuthService) Register(name, email, password string, keepUpdated bool) (*models.User, string, error) {
	user, session, err := s.RegisterWithSession(name, email, password, keepUpdated)
	if err != nil {
		return nil, "", err
	}
	return user, session.AccessToken, nil
}

func (s *kratosAuthService) RegisterWithSession(name, email, password string, keepUpdated bool) (*models.User, AuthSession, error) {
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, AuthSession{}, errors.New("email already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, AuthSession{}, err
	}

	// Si el operador deshabilitó el correo de registro (smtp_enabled=false o
	// signup_email_enabled=false en app_settings), no se debe aplicar SMTP:
	// crear vía Admin API pre-verificada y devolver sesión completa sin OTP.
	if !s.signupEmailEnabled() {
		if user, session, err := s.registerWithoutEmail(name, email, password, keepUpdated); err == nil {
			return user, session, nil
		}
		// Si el admin-flow falla (p.ej. ya existe en Kratos), continuar con el
		// flujo normal + auto-confirmación abajo.
	}

	identity, err := s.kratos.Register(email, password, name)
	if err != nil {
		// Cuenta creada pero email sin verificar: espejar public.users
		// (mismo id) y surface ErrEmailNotConfirmed para 201
		// {needsVerification:true} en vez de 500.
		if isUnverifiedErr(err) {
			return s.registerPendingIdentity(name, email, keepUpdated, err)
		}
		// Mailer failures (SMTP sin configurar, 535, caída del proveedor):
		// el handler responde 503. Salvo operador con correo deshabilitado:
		// auto-confirmar vía admin.
		if errors.Is(err, auth.ErrEmailServiceUnavailable) || auth.IsMailerErr(err) {
			if !s.signupEmailEnabled() {
				if resolved, rErr := s.kratos.AdminGetUserByEmail(email); rErr == nil && resolved != "" {
					if user, sess, cErr := s.confirmAndMirror(name, email, keepUpdated, resolved); cErr == nil {
						return user, sess, nil
					}
				}
			}
			return nil, AuthSession{}, err
		}
		// Orphan recovery: un intento previo creó la identidad pero falló
		// después (SMTP caído antes del mirror), así que Kratos dice
		// "already exists". Espejar y reportar needs-verification.
		if errors.Is(err, auth.ErrAlreadyExists) || isAlreadyRegisteredErr(err) {
			if resolved, resolveErr := s.kratos.AdminGetUserByEmail(email); resolveErr == nil && resolved != "" {
				if !s.signupEmailEnabled() {
					if user, sess, cErr := s.confirmAndMirror(name, email, keepUpdated, resolved); cErr == nil {
						return user, sess, nil
					}
				}
				if id, gErr := s.kratos.AdminGetIdentity(resolved); gErr == nil && id.Verified {
					return s.mirrorAndMint(name, email, keepUpdated, id)
				}
				if user, uErr := s.mirrorUser(resolved, name, email, keepUpdated); uErr == nil {
					return user, AuthSession{}, ErrEmailNotConfirmed
				}
			}
		}
		return nil, AuthSession{}, err
	}

	// Registro con email ya verificado (p.ej. verificación deshabilitada en
	// Kratos o correo deshabilitado): sesión completa directa.
	if identity.Verified {
		return s.mirrorAndMint(name, email, keepUpdated, identity)
	}

	userRole, err := s.ensureUserRole()
	if err != nil {
		_ = s.kratos.AdminDeleteUser(identity.ID)
		return nil, AuthSession{}, err
	}

	user := &models.User{
		ID:          identity.ID,
		Name:        name,
		Email:       email,
		Password:    "", // Passwords live in Kratos only for new users.
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if email == "" {
		user.Email = identity.Email
	}

	if err := s.userRepo.Create(user); err != nil {
		// Avoid orphan identities when the profile insert fails.
		_ = s.kratos.AdminDeleteUser(identity.ID)
		return nil, AuthSession{}, err
	}

	// Kratos envía el OTP y la cuenta queda pendiente de verificación.
	return user, AuthSession{}, ErrEmailNotConfirmed
}

// isUnverifiedErr reports whether err means the identity exists but its
// email is still unverified (client surfaces it when the register response
// carries an unverified identity).
func isUnverifiedErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrEmailNotConfirmed)
}

// isAlreadyRegisteredErr reports whether err means the email already has an
// identity (Kratos answers 400 "already exists / same identifier").
func isAlreadyRegisteredErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrAlreadyExists) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already registered") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "same identifier") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "user already")
}

// registerPendingIdentity mirrors public.users for an identity whose email is
// not verified yet. Returns ErrEmailNotConfirmed for 201 {needsVerification}.
func (s *kratosAuthService) registerPendingIdentity(name, email string, keepUpdated bool, signupErr error) (*models.User, AuthSession, error) {
	identityID, err := s.kratos.AdminGetUserByEmail(email)
	if err != nil || identityID == "" {
		return nil, AuthSession{}, signupErr
	}
	return s.registerPendingUser(name, email, keepUpdated, identityID, signupErr)
}

func (s *kratosAuthService) registerPendingUser(name, email string, keepUpdated bool, identityID string, signupErr error) (*models.User, AuthSession, error) {
	userRole, err := s.ensureUserRole()
	if err != nil {
		return nil, AuthSession{}, err
	}

	user := &models.User{
		ID:          identityID,
		Name:        name,
		Email:       email,
		Password:    "", // Passwords live in Kratos only for new users.
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		// Profile may already exist (retry after partial failure):
		// still report "needs verification" instead of a 500.
		if existing, findErr := s.userRepo.FindByEmail(user.Email); findErr == nil && existing != nil {
			return existing, AuthSession{}, ErrEmailNotConfirmed
		}
		return nil, AuthSession{}, err
	}

	return user, AuthSession{}, ErrEmailNotConfirmed
}

// registerWithoutEmail crea la identidad vía Admin API pre-verificada
// (no pasa por SMTP) y devuelve la sesión completa. Solo se usa cuando el
// operador deshabilitó el correo de registro en el módulo de configuraciones.
func (s *kratosAuthService) registerWithoutEmail(name, email, password string, keepUpdated bool) (*models.User, AuthSession, error) {
	identityID, err := s.kratos.AdminCreateUser(email, password, name, true)
	if err != nil {
		return nil, AuthSession{}, err
	}
	return s.mirrorAndMint(name, email, keepUpdated, &auth.Identity{ID: identityID, Email: email, Name: name, Verified: true})
}

// confirmAndMirror espeja public.users para una identidad existente y devuelve
// sesión completa. Usado cuando el correo está deshabilitado y la cuenta quedó
// pendiente: no hay confirmación pendiente en Kratos que aplicar porque la
// identidad ya existe; basta espejar (la verificación se considera otorgada
// por el operador).
func (s *kratosAuthService) confirmAndMirror(name, email string, keepUpdated bool, identityID string) (*models.User, AuthSession, error) {
	if identityID == "" {
		return nil, AuthSession{}, errors.New("cannot confirm without identity id")
	}
	return s.mirrorAndMint(name, email, keepUpdated, &auth.Identity{ID: identityID, Email: email, Name: name, Verified: true})
}

// mirrorAndMint espeja el perfil y minta el par de sesión API.
func (s *kratosAuthService) mirrorAndMint(name, email string, keepUpdated bool, id *auth.Identity) (*models.User, AuthSession, error) {
	user, err := s.mirrorUser(id.ID, name, email, keepUpdated)
	if err != nil {
		return nil, AuthSession{}, err
	}
	sess, err := s.mintPair(id.ID, user.Email)
	if err != nil {
		return nil, AuthSession{}, err
	}
	return user, sess, nil
}

// mirrorUser crea el perfil public.users (id = identity id) si no existe.
func (s *kratosAuthService) mirrorUser(identityID, name, email string, keepUpdated bool) (*models.User, error) {
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
		ID:          identityID,
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

// Login validates credentials against Kratos, then loads the business
// profile (roles, photoUrl, ...) from public.users.
func (s *kratosAuthService) Login(email, password string) (*models.User, string, error) {
	user, session, err := s.LoginWithSession(email, password)
	if err != nil {
		return nil, "", err
	}
	return user, session.AccessToken, nil
}

func (s *kratosAuthService) LoginWithSession(email, password string) (*models.User, AuthSession, error) {
	identity, err := s.kratos.Login(email, password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, AuthSession{}, errors.New("invalid credentials")
		}
		msg := err.Error()
		if strings.Contains(msg, "400") || strings.Contains(msg, "401") || strings.Contains(msg, "Invalid") || strings.Contains(msg, "invalid") {
			return nil, AuthSession{}, errors.New("invalid credentials")
		}
		return nil, AuthSession{}, err
	}
	if !identity.Verified {
		// Cuentas creadas cuando la verificación estaba activa pero el
		// operador la desactivó después: auto-mint en vez de dejarlas
		// atoradas en un OTP que ya no se envía (resend → EMAIL_DISABLED).
		if !s.signupEmailEnabled() {
			return s.mirrorAndMint("", email, false, identity)
		}
		return nil, AuthSession{}, ErrEmailNotConfirmed
	}

	// Canonical profile: lookup by email so the business profile (roles,
	// photoUrl, ...) is always loaded for the Kratos identity.
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// Identity exists in Kratos but profile row is missing: recreate it.
		mirrored, mErr := s.mirrorUser(identity.ID, identity.Name, email, false)
		if mErr != nil {
			return nil, AuthSession{}, mErr
		}
		user = mirrored
	}

	sess, err := s.mintPair(user.ID, user.Email)
	if err != nil {
		return nil, AuthSession{}, err
	}
	return user, sess, nil
}

// VerifyEmailWithSession confirms the account with the 6-digit email OTP and
// returns a full session on success. The public.users mirror row is created
// when missing so verification never leaves the account half-provisioned.
func (s *kratosAuthService) VerifyEmailWithSession(email, code string) (*models.User, AuthSession, error) {
	if email == "" || code == "" {
		return nil, AuthSession{}, errors.New("email and code are required")
	}
	identity, err := s.kratos.SubmitVerificationCode(email, code)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCode) {
			return nil, AuthSession{}, errors.New("invalid or expired verification code")
		}
		msg := err.Error()
		if strings.Contains(msg, "400") || strings.Contains(msg, "401") || strings.Contains(msg, "422") ||
			strings.Contains(strings.ToLower(msg), "invalid") || strings.Contains(strings.ToLower(msg), "expired") {
			return nil, AuthSession{}, errors.New("invalid or expired verification code")
		}
		return nil, AuthSession{}, err
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		mirrored, mErr := s.mirrorUser(identity.ID, identity.Name, email, false)
		if mErr != nil {
			return nil, AuthSession{}, mErr
		}
		user = mirrored
	}

	sess, err := s.mintPair(user.ID, user.Email)
	if err != nil {
		return nil, AuthSession{}, err
	}
	return user, sess, nil
}

// VerifyEmail adapts VerifyEmailWithSession to the AuthService contract.
func (s *kratosAuthService) VerifyEmail(email, code string) (*models.User, string, error) {
	user, session, err := s.VerifyEmailWithSession(email, code)
	if err != nil {
		return nil, "", err
	}
	return user, session.AccessToken, nil
}

// ResendVerification re-sends the verification OTP email via Kratos.
// The account must exist locally; unknown emails get "user not found" so we
// never leak which emails are registered. Si el operador deshabilitó el
// correo de registro, devuelve ErrAuthEmailDisabled.
func (s *kratosAuthService) ResendVerification(email string) error {
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
	return s.kratos.RequestVerificationCode(email)
}

// RefreshSession rotates an opaque refresh token for a new pair.
func (s *kratosAuthService) RefreshSession(refreshToken string) (AuthSession, error) {
	if refreshToken == "" {
		return AuthSession{}, errors.New("refresh token is required")
	}
	userID, err := auth.ConsumeRefreshToken(refreshToken)
	if err != nil {
		return AuthSession{}, err
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return AuthSession{}, auth.ErrRefreshInvalid
	}
	return s.mintPair(user.ID, user.Email)
}

// Logout revokes all refresh tokens of the token owner. API access tokens
// are short-lived JWTs; revoking refresh stops renewal. Always nil so
// Flutter logout never fails visibly.
func (s *kratosAuthService) Logout(accessToken string) error {
	if accessToken == "" {
		return nil
	}
	if userID := utils.ParseUnverifiedUserID(accessToken); userID != "" {
		auth.RevokeUserTokens(userID)
	}
	return nil
}

// ChangePassword verifies the current password against Kratos and sets the
// new one there (source of truth). Revokes all refresh tokens.
func (s *kratosAuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Resolve the Kratos identity id (normally == public.users.id).
	identityID := user.ID
	if resolved, resolveErr := s.kratos.AdminGetUserByEmail(user.Email); resolveErr == nil && resolved != "" {
		identityID = resolved
	}

	// Verify current password against Kratos.
	if _, signInErr := s.kratos.Login(user.Email, currentPassword); signInErr != nil {
		return errors.New("current password incorrect")
	}

	if err := s.kratos.AdminUpdatePassword(identityID, newPassword); err != nil {
		return err
	}
	auth.RevokeUserTokens(user.ID)
	return nil
}

// ForgotPassword keeps the Flutter {email}->{message} contract: it mints the
// 6-digit recovery code (local hardened store). Si el operador deshabilitó
// el correo de recuperación, igual se genera el código pero no se intenta
// ningún envío.
func (s *kratosAuthService) ForgotPassword(email string) error {
	if err := issueRecoveryCode(s.userRepo, email); err != nil {
		return err
	}
	if !s.recoveryEmailEnabled() {
		return nil
	}
	// Best effort deshabilitado: el código local es el contrato.
	return nil
}

// ResetPassword validates the recovery code (Flutter contract) and sets the
// new password in Kratos. Revokes all refresh tokens.
func (s *kratosAuthService) ResetPassword(email, code, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	user, err := consumeRecoveryCode(s.userRepo, email, code)
	if err != nil {
		return err
	}
	identityID := user.ID
	if resolved, resolveErr := s.kratos.AdminGetUserByEmail(email); resolveErr == nil && resolved != "" {
		identityID = resolved
	}
	if err := s.kratos.AdminUpdatePassword(identityID, newPassword); err != nil {
		return err
	}
	auth.RevokeUserTokens(user.ID)
	return nil
}

// ValidateToken resolves an API Bearer token to a user id.
func (s *kratosAuthService) ValidateToken(token string) (string, error) {
	claims, err := utils.ValidateToken(token)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
