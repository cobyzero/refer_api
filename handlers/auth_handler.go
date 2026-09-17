package handlers

import (
	"errors"
	"strings"

	"refer_api/auth"
	"refer_api/middleware"
	"refer_api/models"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

type RegisterRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	KeepUpdated bool   `json:"keepUpdated"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// AuthHandler coordinates HTTP logic with the AuthService (Ory Kratos).
type AuthHandler struct {
	authService services.SessionAuthService
}

// NewAuthHandler constructs a new AuthHandler.
func NewAuthHandler(authService services.SessionAuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func roleNames(user *models.User) []string {
	if user == nil {
		return []string{}
	}
	out := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		out = append(out, r.Name)
	}
	return out
}

// Register maps HTTP registration to the AuthService.
// Contract preserved: {name,email,password,keepUpdated} ->
// 201 {user:{id,name,email,keepUpdated,roles}, accessToken, refreshToken}.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Name, email, and password are required"})
	}

	// Kratos returns a real session pair {accessToken, refreshToken}.
	user, session, err := h.authService.RegisterWithSession(req.Name, req.Email, req.Password, req.KeepUpdated)
	if err != nil {
		// AUTOCONFIRM=false: account created, email OTP pending.
		// 201 (not an error) so Flutter shows the verification screen.
		if errors.Is(err, services.ErrEmailNotConfirmed) && user != nil {
			middleware.AttachAudit(c, "auth.register.pending", "user", user.ID, map[string]interface{}{"email": user.Email})
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{
				"user": fiber.Map{
					"id":          user.ID,
					"name":        user.Name,
					"email":       user.Email,
					"keepUpdated": user.KeepUpdated,
					"roles":       roleNames(user),
				},
				"needsVerification": true,
				"message":           "Te enviamos un código de verificación. Revisa tu correo.",
			})
		}
		if strings.Contains(err.Error(), "already exists") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		// Mailer outage (SMTP sin configurar, credenciales mal, caída del
		// proveedor): 503 para que Flutter muestre "reintenta más tarde"
		// en vez de un 500 genérico. El detalle queda en el log del server.
		if errors.Is(err, auth.ErrEmailServiceUnavailable) || auth.IsMailerErr(err) {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"message": "No pudimos enviar el correo de verificación. Inténtalo de nuevo en unos minutos.",
				"code":    "EMAIL_SERVICE_UNAVAILABLE",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "auth.register", "user", user.ID, map[string]interface{}{"email": user.Email})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"keepUpdated": user.KeepUpdated,
			"roles":       roleNames(user),
		},
		"accessToken":  session.AccessToken,
		"refreshToken": session.RefreshToken,
	})
}

// Login validates user details and returns access tokens.
// Contract preserved: {email,password} -> 200 {user, accessToken, refreshToken}.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	user, session, err := h.authService.LoginWithSession(req.Email, req.Password)
	if err != nil {
		// Unverified account: 403 (not 401) so Flutter routes to the
		// OTP screen instead of showing "wrong password".
		if errors.Is(err, services.ErrEmailNotConfirmed) {
			middleware.AttachAudit(c, "auth.login.unverified", "user", "", map[string]interface{}{"email": req.Email})
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Tu correo aún no está verificado. Revisa tu bandeja e ingresa el código.",
				"code":    "EMAIL_NOT_VERIFIED",
			})
		}
		middleware.AttachAudit(c, "auth.login.failed", "user", "", map[string]interface{}{"email": req.Email})
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "auth.login", "user", user.ID, map[string]interface{}{"email": user.Email})
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"keepUpdated": user.KeepUpdated,
			"photoUrl":    user.PhotoURL,
			"roles":       roleNames(user),
		},
		"accessToken":  session.AccessToken,
		"refreshToken": session.RefreshToken,
	})
}

// ForgotPassword handles sending password recovery code.
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email is required"})
	}

	if err := h.authService.ForgotPassword(req.Email); err != nil {
		if errors.Is(err, services.ErrAuthEmailDisabled) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "La recuperación por correo está deshabilitada por configuración.",
				"code":    "EMAIL_DISABLED",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "auth.forgot", "user", "", map[string]interface{}{"email": req.Email})

	return c.JSON(fiber.Map{
		"message": "Recovery code generated and sent.",
	})
}

// ResetPassword verifies the code and resets the password.
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email, code, and newPassword are required"})
	}

	if err := h.authService.ResetPassword(req.Email, req.Code, req.NewPassword); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "auth.reset", "user", "", map[string]interface{}{"email": req.Email})

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// VerifyEmail confirms the account with the 6-digit OTP from the signup email.
// Contract: {email, code} -> 200 {user, accessToken, refreshToken}.
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Email == "" || req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email and code are required"})
	}

	user, session, err := h.authService.VerifyEmailWithSession(req.Email, req.Code)
	if err != nil {
		middleware.AttachAudit(c, "auth.verify.failed", "user", "", map[string]interface{}{"email": req.Email})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "auth.verify", "user", user.ID, map[string]interface{}{"email": user.Email})
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"keepUpdated": user.KeepUpdated,
			"photoUrl":    user.PhotoURL,
			"roles":       roleNames(user),
		},
		"accessToken":  session.AccessToken,
		"refreshToken": session.RefreshToken,
	})
}

// ResendVerification re-sends the signup OTP email.
// Contract: {email} -> 200 {message}.
func (h *AuthHandler) ResendVerification(c *fiber.Ctx) error {
	var req ResendVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email is required"})
	}

	if err := h.authService.ResendVerification(req.Email); err != nil {
		if errors.Is(err, services.ErrAuthEmailDisabled) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "El envío de correos de verificación está deshabilitado por configuración.",
				"code":    "EMAIL_DISABLED",
			})
		}
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		if errors.Is(err, auth.ErrEmailServiceUnavailable) || auth.IsMailerErr(err) {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"message": "No pudimos reenviar el correo de verificación. Inténtalo de nuevo en unos minutos.",
				"code":    "EMAIL_SERVICE_UNAVAILABLE",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "auth.verify.resend", "user", "", map[string]interface{}{"email": req.Email})

	return c.JSON(fiber.Map{
		"message": "Verification code re-sent. Revisa tu correo.",
	})
}

// Refresh exchanges an API refresh token for a new pair.
// Contract: {refreshToken} -> {accessToken, refreshToken}.
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	session, err := h.authService.RefreshSession(req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{
		"accessToken":  session.AccessToken,
		"refreshToken": session.RefreshToken,
	})
}

// Logout revokes the current session. Always 200 so Flutter logout never
// fails visibly; token is taken from Authorization when present.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	accessToken := ""
	if parts := strings.Split(auth, " "); len(parts) == 2 {
		accessToken = parts[1]
	}
	_ = h.authService.Logout(accessToken)
	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// Me returns the current profile from the JWT sub (never trust client ids).
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
	}
	email, _ := c.Locals("email").(string)
	roles, _ := c.Locals("roles").([]string)
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    userId,
			"email": email,
			"roles": roles,
		},
	})
}

// ChangePassword updates user password from secure session context.
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userId := c.Locals("userId").(string)

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if err := h.authService.ChangePassword(userId, req.CurrentPassword, req.NewPassword); err != nil {
		if strings.Contains(err.Error(), "incorrect") || strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	middleware.AttachAudit(c, "auth.password", "user", userId, nil)
	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// Validate verifies incoming token integrity and returns payload.
func (h *AuthHandler) Validate(c *fiber.Ctx) error {
	userId := c.Locals("userId")
	email := c.Locals("email")
	roles := c.Locals("roles")

	return c.JSON(fiber.Map{
		"valid": true,
		"user": fiber.Map{
			"userId": userId,
			"email":  email,
			"roles":  roles,
		},
	})
}
