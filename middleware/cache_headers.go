package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// SetPublicCache marks a hot read-heavy GET as cacheable by clients/CDN.
// Fase 0: no server-side store, only headers — safe because callers
// invalidate by TTL (30s default for catalog/config).
func SetPublicCache(c *fiber.Ctx, maxAgeSeconds int) {
	c.Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeSeconds))
	c.Set("Vary", "Accept-Encoding")
}

// SetPrivateNoStore prevents proxies from caching per-user payloads.
func SetPrivateNoStore(c *fiber.Ctx) {
	c.Set("Cache-Control", "private, no-store")
}
