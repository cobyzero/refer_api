package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

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

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// AuthHandler coordinates HTTP logic with the AuthService.
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler constructs a new AuthHandler.
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func generateMockRefreshToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "mock-refresh-token"
	}
	return "mock-refresh-token-" + hex.EncodeToString(b)
}

// Register maps HTTP registration to the AuthService.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Name, email, and password are required"})
	}

	user, token, err := h.authService.Register(req.Name, req.Email, req.Password, req.KeepUpdated)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	var rolesList []string
	for _, r := range user.Roles {
		rolesList = append(rolesList, r.Name)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"keepUpdated": user.KeepUpdated,
			"roles":       rolesList,
		},
		"accessToken":  token,
		"refreshToken": generateMockRefreshToken(),
	})
}

// Login validates user details and returns access tokens.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	user, token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err.Error()})
	}

	var rolesList []string
	for _, r := range user.Roles {
		rolesList = append(rolesList, r.Name)
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"keepUpdated": user.KeepUpdated,
			"photoUrl":    user.PhotoURL,
			"roles":       rolesList,
		},
		"accessToken":  token,
		"refreshToken": generateMockRefreshToken(),
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

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

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// Refresh handles token refresh mocks.
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	return c.JSON(fiber.Map{
		"accessToken":  "new-mock-access-token",
		"refreshToken": req.RefreshToken,
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
