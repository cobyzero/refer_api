package routes

import (
	"refer_api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterSearchRoutes binds search endpoints on the router group.
func RegisterSearchRoutes(router fiber.Router, searchHandler *handlers.SearchHandler) {
	search := router.Group("/search")

	search.Get("/referrals", searchHandler.Search)
	search.Get("/categories", searchHandler.GetCategories)
}
