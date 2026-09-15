package handlers

import (
	"refer_api/middleware"
	"refer_api/services"
	"refer_api/utils"

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
// Fase 0: supports ?page&limit. Without them returns the legacy bare array;
// with them returns {data, total, page, limit}.
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 15)
	q := c.Query("q")
	category := c.Query("category")
	pg := utils.ParsePagination(c, 20, 100)

	if !pg.Active {
		results, err := h.searchService.SearchProducts(q, category)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(results)
	}

	results, total, err := h.searchService.SearchProductsPaged(q, category, pg.Page, pg.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(utils.PagedEnvelope(results, total, pg.Page, pg.Limit))
}

// GetCategories returns unique category filters.
func (h *SearchHandler) GetCategories(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 120)
	categories, err := h.searchService.GetCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(categories)
}
