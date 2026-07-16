package handlers

import (
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// SearchHandler coordinates HTTP search requests.
type SearchHandler struct {
	searchService services.SearchService
}

// NewSearchHandler constructs a new SearchHandler.
func NewSearchHandler(searchService services.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// Search processes product search filters via query parameters.
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	q := c.Query("q")
	category := c.Query("category")

	results, err := h.searchService.SearchProducts(q, category)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(results)
}

// GetCategories returns unique category filters.
func (h *SearchHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.searchService.GetCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(categories)
}
