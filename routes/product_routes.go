package routes

import (
	"refer_api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterProductRoutes binds all product-related endpoints on the router.
func RegisterProductRoutes(router fiber.Router, productHandler *handlers.ProductHandler) {
	products := router.Group("/products")

	// Public Operations
	products.Get("/", productHandler.GetAll)
	products.Get("/top-rated", productHandler.GetTopRated)
	products.Get("/seasonal", productHandler.GetSeasonal)
	products.Get("/categories", productHandler.GetCategories)
	products.Get("/:id", productHandler.GetOne)

	// Admin / Modification Operations
	products.Patch("/:id", productHandler.Update)
	products.Post("/", productHandler.Create)
	products.Post("/upload", productHandler.UploadPhoto)
	products.Post("/:id/enhancements", productHandler.AddEnhancement)
	products.Delete("/:id/enhancements/:enhancementId", productHandler.DeleteEnhancement)
}
