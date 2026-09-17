package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"

	"refer_api/middleware"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// UserHandler coordinates HTTP requests for user endpoints.
type UserHandler struct {
	userService    services.UserService
	storageService services.StorageService
}

// NewUserHandler constructs a new UserHandler instance.
func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// WithStorage injects the upload backend (R2 w/ disk fallback).
func (h *UserHandler) WithStorage(s services.StorageService) *UserHandler {
	h.storageService = s
	return h
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
	middleware.AttachAudit(c, "user.profile", "user", userID, updateData)

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

// UploadPhoto stores profile photos via StorageService (R2,
// disk fallback). Contract unchanged: multipart `file` -> 201 {photoUrl}.
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
	newFilename := fmt.Sprintf("%s%s", hex.EncodeToString(randomBytes), filepath.Ext(file.Filename))

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to read file"})
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, 50<<20))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to read file"})
	}

	photoURL := fmt.Sprintf("%s/uploads/%s", c.BaseURL(), newFilename)
	if h.storageService != nil {
		if url, saveErr := h.storageService.Save(services.BucketAvatars, newFilename, data, file.Header.Get("Content-Type"), c.BaseURL()); saveErr == nil {
			photoURL = url
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to save file"})
		}
	} else {
		if url, saveErr := services.SaveToDisk(newFilename, data, c.BaseURL()); saveErr == nil {
			photoURL = url
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to save file"})
		}
	}

	if updateErr := h.userService.UpdatePhoto(userID, photoURL); updateErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update profile image path"})
	}

	middleware.AttachAudit(c, "user.photo", "user", userID, nil)
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
// Fase 0: roles are cached 5 min in middleware; new users start uncached.
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
	middleware.AttachAudit(c, "user.create", "user", user.ID, map[string]interface{}{"email": user.Email, "roles": roleNames})

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
	// Fase 0: role changes must propagate immediately despite the 5-min cache.
	if len(roleNames) > 0 {
		middleware.InvalidateRolesCache(id)
	}
	middleware.AttachAudit(c, "user.update", "user", id, map[string]interface{}{"roles": roleNames})

	return c.JSON(user)
}

// Delete removes a user (Admin only).
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.userService.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.InvalidateRolesCache(id)
	middleware.AttachAudit(c, "user.delete", "user", id, nil)

	return c.JSON(fiber.Map{
		"success": true,
	})
}
