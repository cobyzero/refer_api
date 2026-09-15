package routes

import (
	"refer_api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterCollectionRoutes binds featured-collection endpoints.
// Kept beside (not inside) product routes: the legacy GET /products/seasonal
// contract used by mobile clients stays untouched.
func RegisterCollectionRoutes(router fiber.Router, collectionHandler *handlers.CollectionHandler) {
	collections := router.Group("/collections")

	collections.Get("/", collectionHandler.GetCollections)
	collections.Post("/", collectionHandler.CreateCollection)
	collections.Patch("/:id", collectionHandler.UpdateCollection)
	collections.Delete("/:id", collectionHandler.DeleteCollection)
	collections.Post("/:id/items", collectionHandler.AddCollectionItem)
	collections.Delete("/:id/items/:productId", collectionHandler.RemoveCollectionItem)
}
