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
	products.Get("/sizes", productHandler.GetSizes)
	products.Get("/types", productHandler.GetAllTypes)
	products.Get("/enhancements", productHandler.GetAllEnhancements)
	products.Get("/:id", productHandler.GetOne)

	// Admin / Modification Operations
	products.Patch("/:id", productHandler.Update)
	products.Post("/", productHandler.Create)
	products.Post("/upload", productHandler.UploadPhoto)
	products.Post("/categories", productHandler.CreateCategory)
	products.Patch("/categories/:id", productHandler.UpdateCategory)
	products.Delete("/categories/:id", productHandler.DeleteCategory)
	products.Post("/sizes", productHandler.CreateSize)
	products.Patch("/sizes/:id", productHandler.UpdateSize)
	products.Delete("/sizes/:id", productHandler.DeleteSize)
	products.Post("/:id/sizes", productHandler.AssignSize)
	products.Delete("/:id/sizes/:sizeId", productHandler.RemoveSize)
	products.Post("/:id/types", productHandler.AddType)
	products.Delete("/:id/types/:typeId", productHandler.DeleteType)
	products.Post("/upload", productHandler.UploadPhoto)
	products.Post("/:id/enhancements", productHandler.AddEnhancement)
	products.Delete("/:id/enhancements/:enhancementId", productHandler.DeleteEnhancement)
}
