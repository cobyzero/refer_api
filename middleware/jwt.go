package middleware

import (
	"context"
	"strings"

	"refer_api/database"
	"refer_api/repository"
	"refer_api/utils"

	"github.com/gofiber/fiber/v2"
)

// EnrichRolesFromDB loads app roles (ADMIN/USER) from public.users when the
// JWT itself carries none. Supabase Auth JWTs never contain our app roles,
// so without this step RequireRole("ADMIN") would reject Supabase sessions.
// Exported so the Socket.IO gateway reuses the exact same logic as HTTP.
//
// Fase 0: 5-min in-memory cache + per-user singleflight so concurrent
// requests for the same user share one DB lookup instead of stampeding it.
func EnrichRolesFromDB(userID string, roles []string) []string {
	if len(roles) > 0 || userID == "" || database.DB == nil {
		return roles
	}
	if cached, ok := getCachedRoles(userID); ok {
		return cached
	}

	// Singleflight-lite: concurrent callers for the same userID wait on one DB hit.
	c := &call{}
	c.wg.Add(1)
	if actual, loaded := rolesInflight.LoadOrStore(userID, c); loaded {
		actual.(*call).wg.Wait()
		if cached, ok := getCachedRoles(userID); ok {
			return cached
		}
		return roles
	}
	defer func() {
		c.wg.Done()
		rolesInflight.Delete(userID)
	}()

	repo := repository.NewUserRepository(database.DB)
	user, err := repo.FindByID(userID)
	if err != nil || user == nil {
		// Don't cache misses: transient DB errors shouldn't pin empty roles.
		_ = context.Background()
		return roles
	}
	out := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		out = append(out, r.Name)
	}
	setCachedRoles(userID, out)
	c.roles = out
	return out
}

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
		// Dual validation: Supabase JWT first (when SUPABASE_JWT_SECRET is set),
		// legacy JWT_SECRET as fallback so existing Flutter sessions survive.
		claims, err := utils.ValidateAnyToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid or expired token",
			})
		}

		roles := EnrichRolesFromDB(claims.UserID, claims.Roles)

		// Store user details in Fiber locals
		c.Locals("userId", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("roles", roles)
		c.Locals("authProvider", string(claims.Provider))

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
