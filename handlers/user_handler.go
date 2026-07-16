package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// UserHandler coordinates HTTP requests for user endpoints.
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler constructs a new UserHandler instance.
func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile returns the authenticated user's profile and calculated star balance.
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	user, stars, err := h.userService.GetProfile(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User not found"})
	}

	var rolesList []string
	for _, r := range user.Roles {
		rolesList = append(rolesList, r.Name)
	}

	return c.JSON(fiber.Map{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"keepUpdated": user.KeepUpdated,
		"photoUrl":    user.PhotoURL,
		"phoneNumber": user.PhoneNumber,
		"birthDate":   user.BirthDate,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
		"roles":       rolesList,
		"stars":       stars,
	})
}

// UpdateProfile updates the current user profile.
func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	user, err := h.userService.UpdateProfile(userID, updateData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	var rolesList []string
	for _, r := range user.Roles {
		rolesList = append(rolesList, r.Name)
	}

	return c.JSON(fiber.Map{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"keepUpdated": user.KeepUpdated,
		"photoUrl":    user.PhotoURL,
		"phoneNumber": user.PhoneNumber,
		"birthDate":   user.BirthDate,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
		"roles":       rolesList,
	})
}

// UploadPhoto saves profile photo uploads to disk.
func (h *UserHandler) UploadPhoto(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "No file uploaded or invalid format"})
	}

	randomBytes := make([]byte, 16)
	if _, randErr := rand.Read(randomBytes); randErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to generate unique filename"})
	}
	randomName := hex.EncodeToString(randomBytes)
	extension := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf("%s%s", randomName, extension)

	uploadDir := "./uploads"
	if _, pathErr := os.Stat(uploadDir); os.IsNotExist(pathErr) {
		if makeErr := os.Mkdir(uploadDir, 0755); makeErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create upload directory"})
		}
	}

	destPath := filepath.Join(uploadDir, newFilename)
	if saveErr := c.SaveFile(file, destPath); saveErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to save file"})
	}

	photoURL := fmt.Sprintf("%s/uploads/%s", c.BaseURL(), newFilename)

	if updateErr := h.userService.UpdatePhoto(userID, photoURL); updateErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update profile image path"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"photoUrl": photoURL,
	})
}

// FindAll returns all users (Admin only).
func (h *UserHandler) FindAll(c *fiber.Ctx) error {
	users, err := h.userService.FindAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(users)
}

// FindByID returns a single user by ID (Admin only).
func (h *UserHandler) FindByID(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := h.userService.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User not found"})
	}
	return c.JSON(user)
}

// Create inserts a user manually (Admin only).
func (h *UserHandler) Create(c *fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	var roleNames []string
	if rawRoles, ok := body["roleNames"].([]interface{}); ok {
		for _, r := range rawRoles {
			if s, isStr := r.(string); isStr {
				roleNames = append(roleNames, s)
			}
		}
	}

	user, err := h.userService.Create(body, roleNames)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// Update modifies user properties (Admin only).
func (h *UserHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	var roleNames []string
	if rawRoles, ok := body["roleNames"].([]interface{}); ok {
		for _, r := range rawRoles {
			if s, isStr := r.(string); isStr {
				roleNames = append(roleNames, s)
			}
		}
	}

	user, err := h.userService.Update(id, body, roleNames)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(user)
}

// Delete removes a user (Admin only).
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.userService.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}
