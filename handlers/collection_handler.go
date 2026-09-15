package handlers

import (
	"errors"

	"refer_api/middleware"
	"refer_api/models"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// CollectionHandler coordinates HTTP logic for featured-collection endpoints.
type CollectionHandler struct {
	collectionService services.CollectionService
}

// NewCollectionHandler constructs a new CollectionHandler.
func NewCollectionHandler(collectionService services.CollectionService) *CollectionHandler {
	return &CollectionHandler{collectionService: collectionService}
}

// collectionResponse is the wire shape: collection fields plus mapped products.
type collectionResponse struct {
	ID          string                         `json:"id"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	IsActive    bool                           `json:"isActive"`
	CreatedAt   string                         `json:"createdAt"`
	Products    []models.MappedProductResponse `json:"products"`
}

func mapCollection(c *models.Collection) collectionResponse {
	products := []models.MappedProductResponse{}
	if c != nil {
		for _, item := range c.Items {
			if item.Product == nil {
				continue
			}
			if mapped := models.MapProduct(item.Product); mapped != nil {
				products = append(products, *mapped)
			}
		}
		return collectionResponse{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			IsActive:    c.IsActive,
			CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Products:    products,
		}
	}
	return collectionResponse{Products: products}
}

// GetCollections returns all collections with their mapped products.
func (h *CollectionHandler) GetCollections(c *fiber.Ctx) error {
	collections, err := h.collectionService.FindAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	response := []collectionResponse{}
	for _, collection := range collections {
		collection := collection
		response = append(response, mapCollection(&collection))
	}
	return c.JSON(response)
}

// CreateCollection inserts a new collection.
func (h *CollectionHandler) CreateCollection(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	collection, err := h.collectionService.Create(body.Name, body.Description)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCollectionExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCollectionNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	middleware.AttachAudit(c, "collection.create", "collection", collection.ID, map[string]interface{}{"name": body.Name})
	return c.Status(fiber.StatusCreated).JSON(mapCollection(collection))
}

// UpdateCollection patches name, description, or active flag.
func (h *CollectionHandler) UpdateCollection(c *fiber.Ctx) error {
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"isActive"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	collection, err := h.collectionService.Update(c.Params("id"), body.Name, body.Description, body.IsActive)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCollectionNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCollectionExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCollectionNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	middleware.AttachAudit(c, "collection.update", "collection", c.Params("id"), nil)
	return c.JSON(mapCollection(collection))
}

// DeleteCollection removes a collection and unassigns its items (products survive).
func (h *CollectionHandler) DeleteCollection(c *fiber.Ctx) error {
	if err := h.collectionService.Delete(c.Params("id")); err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	middleware.AttachAudit(c, "collection.delete", "collection", c.Params("id"), nil)
	return c.SendStatus(fiber.StatusNoContent)
}

// AddCollectionItem features a product inside a collection.
func (h *CollectionHandler) AddCollectionItem(c *fiber.Ctx) error {
	var body struct {
		ProductID string `json:"productId"`
	}
	if err := c.BodyParser(&body); err != nil || body.ProductID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "productId is required"})
	}

	collection, err := h.collectionService.AddItem(c.Params("id"), body.ProductID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCollectionNotFound),
			errors.Is(err, services.ErrProductNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCollectionItemExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	middleware.AttachAudit(c, "collection.item.add", "collection", c.Params("id"), map[string]interface{}{"productId": body.ProductID})
	return c.Status(fiber.StatusCreated).JSON(mapCollection(collection))
}

// RemoveCollectionItem unfeatures a product (idempotent).
func (h *CollectionHandler) RemoveCollectionItem(c *fiber.Ctx) error {
	collection, err := h.collectionService.RemoveItem(c.Params("id"), c.Params("productId"))
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	middleware.AttachAudit(c, "collection.item.remove", "collection", c.Params("id"), map[string]interface{}{"productId": c.Params("productId")})
	return c.JSON(mapCollection(collection))
}
