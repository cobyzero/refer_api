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
// The only implementation is the Kratos-backed service: Kratos owns
// credentials (password hashing, email OTP codes); Go mints API sessions.
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

// AuthSession is the token pair returned to Flutter.
// Field names stay {accessToken, refreshToken} at the HTTP layer.
type AuthSession struct {
	AccessToken  string
	RefreshToken string
}

// SessionAuthService is implemented by the Kratos-backed auth service,
// which returns API sessions behind the Flutter contract.
type SessionAuthService interface {
	AuthService
	RegisterWithSession(name, email, password string, keepUpdated bool) (*models.User, AuthSession, error)
	LoginWithSession(email, password string) (*models.User, AuthSession, error)
	VerifyEmailWithSession(email, code string) (*models.User, AuthSession, error)
	ResendVerification(email string) error
	RefreshSession(refreshToken string) (AuthSession, error)
	Logout(accessToken string) error
}
