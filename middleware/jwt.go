package middleware

import (
	"strings"

	"refer_api/utils"

	"github.com/gofiber/fiber/v2"
)

// JWTProtected checks for a valid Bearer token in the Authorization header
func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Missing authorization token",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid or expired token",
			})
		}

		// Store user details in Fiber locals
		c.Locals("userId", claims.Sub)
		c.Locals("email", claims.Email)
		c.Locals("roles", claims.Roles)

		return c.Next()
	}
}

// RequireRole checks if the user possesses the required role
func RequireRole(roleName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rolesLocals := c.Locals("roles")
		if rolesLocals == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Access forbidden: missing roles",
			})
		}

		roles, ok := rolesLocals.([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Access forbidden: invalid roles format",
			})
		}

		for _, r := range roles {
			if r == roleName {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Access forbidden: insufficient permissions",
		})
	}
}
