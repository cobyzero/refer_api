package services

import (
	"errors"

	"refer_api/models"
)

// ErrAuthEmailDisabled is returned when the operator disabled auth emails
// (smtp_enabled / signup_email_enabled / recovery_email_enabled in
// app_settings desde el módulo de configuraciones). Handlers lo mapean a
// 403 {code: EMAIL_DISABLED} para que refer_web oculte/deshabilite la opción.
var ErrAuthEmailDisabled = errors.New("auth emails are disabled by configuration")

// AuthService defines the business logic interface for authentication.
// The only implementation is the Supabase-backed service: Supabase Auth is
// the single source of truth for credentials and sessions.
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
