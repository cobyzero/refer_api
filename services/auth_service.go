package services

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"refer_api/models"
	"refer_api/repository"
	"refer_api/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrAuthEmailDisabled is returned when the operator disabled auth emails
// (smtp_enabled / signup_email_enabled / recovery_email_enabled in
// app_settings desde el módulo de configuraciones). Handlers lo mapean a
// 403 {code: EMAIL_DISABLED} para que refer_web oculte/deshabilite la opción.
var ErrAuthEmailDisabled = errors.New("auth emails are disabled by configuration")

// AuthService defines the business logic interface for authentication.
type AuthService interface {
	Register(name, email, password string, keepUpdated bool) (*models.User, string, error)
	Login(email, password string) (*models.User, string, error)
	ChangePassword(userID, currentPassword, newPassword string) error
	ForgotPassword(email string) error
	ResetPassword(email, code, newPassword string) error
	// VerifyEmail confirms the account with the email OTP code.
	VerifyEmail(email, code string) (*models.User, string, error)
	// ResendVerification re-sends the verification OTP email.
	ResendVerification(email string) error
}

type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService creates a new AuthService instance with its dependencies.
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Register(name, email, password string, keepUpdated bool) (*models.User, string, error) {
	// Check if already exists
	_, err := s.userRepo.FindByEmail(email)
	if err == nil {
		return nil, "", errors.New("email already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, "", err
	}

	// Get USER role
	userRole, err := s.userRepo.FindRoleByName("USER")
	if err != nil {
		// Fallback role creation
		userRole = &models.Role{
			ID:   uuid.New().String(),
			Name: "USER",
		}
		if createErr := s.userRepo.CreateRole(userRole); createErr != nil {
			return nil, "", createErr
		}
	}

	// Create user
	user := &models.User{
		ID:          uuid.New().String(),
		Name:        name,
		Email:       email,
		Password:    string(hashedPassword),
		KeepUpdated: keepUpdated,
		Roles:       []models.Role{*userRole},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", err
	}

	token, err := utils.GenerateToken(user.ID, user.Email, []string{userRole.Name})
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) Login(email, password string) (*models.User, string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("invalid credentials")
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	var rolesList []string
	for _, r := range user.Roles {
		rolesList = append(rolesList, r.Name)
	}

	token, err := utils.GenerateToken(user.ID, user.Email, rolesList)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) ChangePassword(userID, currentPassword, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password incorrect")
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, string(newHashedPassword))
}

// Recovery-code hardening (Fix #1): 6-digit codes are brute-forceable, so
// they get attempt limits, single-use semantics, uniform error messages (no
// oracle for "unknown email" vs "wrong code" vs "expired"), constant-time
// comparison, CSPRNG generation, and a resend cooldown.
const (
	resetCodeTTL         = 15 * time.Minute
	resetCodeCooldown    = 60 * time.Second
	resetCodeMaxAttempts = 5
	// minPasswordLength mirrors refer_app (passwordMinLength: 6).
	minPasswordLength = 6
)

// ErrRecoveryCodeInvalid is the ONLY error surfaced for missing, expired,
// exhausted or mismatched codes, so callers can't distinguish the cases.
var ErrRecoveryCodeInvalid = errors.New("invalid or expired recovery code")

type resetCodeInfo struct {
	code      string
	expiresAt time.Time
	attempts  int
	lastSentAt time.Time
}

var resetCodes sync.Map // maps normalized email (string) to resetCodeInfo

// normalizeRecoveryEmail trims the lookup key so "a@x.com " can't hold a
// separate code slot from "a@x.com".
func normalizeRecoveryEmail(email string) string {
	return strings.TrimSpace(email)
}

func (s *authService) ForgotPassword(email string) error {
	key := normalizeRecoveryEmail(email)
	user, err := s.userRepo.FindByEmail(key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Anti-enumeration: same success as for known users.
			return nil
		}
		return err
	}
	if user == nil {
		return nil
	}

	// Resend cooldown: don't mint a fresh code (nor extend the window) when
	// one was just sent — cheapest spam/brute-force throttle.
	if val, ok := resetCodes.Load(key); ok {
		if info, ok := val.(resetCodeInfo); ok {
			if time.Since(info.lastSentAt) < resetCodeCooldown {
				return nil
			}
		}
	}

	// CSPRNG 6-digit code (math/rand is predictable across restarts).
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	now := time.Now()

	// Store code in memory with 15 minutes expiration
	resetCodes.Store(key, resetCodeInfo{
		code:       code,
		expiresAt:  now.Add(resetCodeTTL),
		lastSentAt: now,
	})

	// Recovery codes are credential-equivalent: never print them unless the
	// operator opts in explicitly (dev). Legacy mode has no mailer, so local
	// devs set LOG_RECOVERY_CODES=true to see the code.
	if os.Getenv("LOG_RECOVERY_CODES") == "true" {
		fmt.Printf("\n[RECOVERY CODE] Generated recovery code for %s: %s (expires in 15 mins)\n\n", user.Email, code)
	}
	return nil
}

func (s *authService) ResetPassword(email, code, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	key := normalizeRecoveryEmail(email)
	val, ok := resetCodes.Load(key)
	if !ok {
		return ErrRecoveryCodeInvalid
	}

	info, ok := val.(resetCodeInfo)
	if !ok {
		resetCodes.Delete(key)
		return ErrRecoveryCodeInvalid
	}
	if time.Now().After(info.expiresAt) || info.attempts >= resetCodeMaxAttempts {
		resetCodes.Delete(key)
		return ErrRecoveryCodeInvalid
	}

	// Constant-time compare: no early-exit timing signal on prefix matches.
	if subtle.ConstantTimeCompare([]byte(info.code), []byte(code)) != 1 {
		info.attempts++
		if info.attempts >= resetCodeMaxAttempts {
			// Burn the code after too many guesses: the attacker must
			// trigger a new forgot-password (cooldown + audit trail).
			resetCodes.Delete(key)
		} else {
			resetCodes.Store(key, info)
		}
		return ErrRecoveryCodeInvalid
	}

	user, err := s.userRepo.FindByEmail(key)
	if err != nil {
		return ErrRecoveryCodeInvalid
	}
	if user == nil {
		resetCodes.Delete(key)
		return ErrRecoveryCodeInvalid
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdatePassword(user.ID, string(newHashedPassword))
	if err != nil {
		return err
	}

	// Single-use: clean up code from memory
	resetCodes.Delete(key)
	return nil
}

// VerifyEmail is not supported in legacy mode: email OTP verification lives
// in Supabase Auth (GoTrue). Kept only to satisfy the AuthService contract so
// handlers compile in both modes.
func (s *authService) VerifyEmail(email, code string) (*models.User, string, error) {
	return nil, "", errors.New("email verification requires Supabase Auth")
}

// ResendVerification is not supported in legacy mode (see VerifyEmail).
func (s *authService) ResendVerification(email string) error {
	return errors.New("email verification requires Supabase Auth")
}
