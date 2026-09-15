package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"refer_api/middleware"
	"refer_api/models"
	"refer_api/services"
	"refer_api/utils"

	"github.com/gofiber/fiber/v2"
)

// ProductHandler coordinates HTTP logic for product endpoints.
type ProductHandler struct {
	productService services.ProductService
	storageService services.StorageService
}

// NewProductHandler constructs a new ProductHandler.
func NewProductHandler(productService services.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// WithStorage injects the upload backend (Supabase Storage w/ disk fallback).
func (h *ProductHandler) WithStorage(s services.StorageService) *ProductHandler {
	h.storageService = s
	return h
}

// GetAll returns all products in mapped structure.
// Fase 0: supports ?page&limit (1-based). Without them it keeps returning
// the legacy bare array so Flutter stays untouched; with them it returns
// {data, total, page, limit} to avoid full-table scans on every scroll.
func (h *ProductHandler) GetAll(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 30)
	pg := utils.ParsePagination(c, 20, 100)

	if !pg.Active {
		products, err := h.productService.FindAll()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
		mapped := make([]models.MappedProductResponse, 0, len(products))
		for _, p := range products {
			mapped = append(mapped, *models.MapProduct(&p))
		}
		return c.JSON(mapped)
	}

	products, total, err := h.productService.FindAllPaged(pg.Page, pg.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	mapped := make([]models.MappedProductResponse, 0, len(products))
	for _, p := range products {
		mapped = append(mapped, *models.MapProduct(&p))
	}
	return c.JSON(utils.PagedEnvelope(mapped, total, pg.Page, pg.Limit))
}

// GetTopRated returns top 5 rated products.
func (h *ProductHandler) GetTopRated(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 60)
	products, err := h.productService.FindTopRated()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	var mapped []models.MappedProductResponse
	for _, p := range products {
		mapped = append(mapped, *models.MapProduct(&p))
	}

	return c.JSON(mapped)
}

// GetSeasonal returns seasonal products.
func (h *ProductHandler) GetSeasonal(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 60)
	products, err := h.productService.FindSeasonal()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	var mapped []models.MappedProductResponse
	for _, p := range products {
		mapped = append(mapped, *models.MapProduct(&p))
	}

	return c.JSON(mapped)
}

// GetCategories returns all product categories.
func (h *ProductHandler) GetCategories(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 120)
	categories, err := h.productService.FindCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(categories)
}

// CreateCategory inserts a new product category.
func (h *ProductHandler) CreateCategory(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		ImageURL    string `json:"imageUrl"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	category, err := h.productService.CreateCategory(body.Name, body.ImageURL, body.Description)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCategoryExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCategoryNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}

// UpdateCategory patches name, image, or description of a category.
func (h *ProductHandler) UpdateCategory(c *fiber.Ctx) error {
	var body struct {
		Name        *string `json:"name"`
		ImageURL    *string `json:"imageUrl"`
		Description *string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	category, err := h.productService.UpdateCategory(c.Params("id"), body.Name, body.ImageURL, body.Description)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCategoryNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCategoryExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrCategoryNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	return c.JSON(category)
}

// DeleteCategory removes a category and unassigns its products (never deletes products).
func (h *ProductHandler) DeleteCategory(c *fiber.Ctx) error {
	if err := h.productService.DeleteCategory(c.Params("id")); err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetSizes returns all product sizes.
func (h *ProductHandler) GetSizes(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 120)
	sizes, err := h.productService.FindSizes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(sizes)
}

// CreateSize inserts a new product size.
func (h *ProductHandler) CreateSize(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	size, err := h.productService.CreateSize(body.Name)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSizeExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrSizeNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(size)
}

// UpdateSize renames a product size.
func (h *ProductHandler) UpdateSize(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	size, err := h.productService.UpdateSize(c.Params("id"), body.Name)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSizeNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrSizeExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrSizeNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	return c.JSON(size)
}

// DeleteSize removes a size and all its product assignments.
func (h *ProductHandler) DeleteSize(c *fiber.Ctx) error {
	if err := h.productService.DeleteSize(c.Params("id")); err != nil {
		if errors.Is(err, services.ErrSizeNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AssignSize assigns a size with its price to a product (upsert).
func (h *ProductHandler) AssignSize(c *fiber.Ctx) error {
	var body struct {
		SizeID string  `json:"sizeId"`
		Price  float64 `json:"price"`
	}
	if err := c.BodyParser(&body); err != nil || body.SizeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "sizeId is required"})
	}

	product, err := h.productService.AssignSize(c.Params("id"), body.SizeID, body.Price)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProductNotFound),
			errors.Is(err, services.ErrSizeNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	return c.JSON(models.MapProduct(product))
}

// RemoveSize unassigns a size from a product (idempotent).
func (h *ProductHandler) RemoveSize(c *fiber.Ctx) error {
	product, err := h.productService.RemoveSize(c.Params("id"), c.Params("sizeId"))
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(models.MapProduct(product))
}

// GetOne returns a single product by ID.
func (h *ProductHandler) GetOne(c *fiber.Ctx) error {
	id := c.Params("id")
	product, err := h.productService.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Product not found"})
	}

	return c.JSON(models.MapProduct(product))
}

// Create inserts a new product.
func (h *ProductHandler) Create(c *fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	product, err := h.productService.Create(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "product.create", "product", product.ID, map[string]interface{}{"name": body["name"]})

	return c.Status(fiber.StatusCreated).JSON(models.MapProduct(product))
}

// Update updates a product by ID.
func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	product, err := h.productService.Update(id, body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	middleware.AttachAudit(c, "product.update", "product", id, nil)

	return c.JSON(models.MapProduct(product))
}

// UploadPhoto stores product photos via StorageService (Supabase Storage,
// disk fallback). Contract unchanged: multipart `file` -> 201 {imageUrl}.
func (h *ProductHandler) UploadPhoto(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "No file uploaded or invalid format"})
	}

	randomBytes := make([]byte, 16)
	if _, randErr := rand.Read(randomBytes); randErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to generate unique filename"})
	}
	newFilename := fmt.Sprintf("%s%s", hex.EncodeToString(randomBytes), filepath.Ext(file.Filename))

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to read file"})
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, 50<<20))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to read file"})
	}

	imageUrl := fmt.Sprintf("%s/uploads/%s", c.BaseURL(), newFilename)
	if h.storageService != nil {
		if url, saveErr := h.storageService.Save(services.BucketProducts, newFilename, data, file.Header.Get("Content-Type"), c.BaseURL()); saveErr == nil {
			imageUrl = url
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to save file"})
		}
	} else {
		if url, saveErr := services.SaveToDisk(newFilename, data, c.BaseURL()); saveErr == nil {
			imageUrl = url
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to save file"})
		}
	}

	middleware.AttachAudit(c, "product.photo", "product", "", nil)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"imageUrl": imageUrl,
	})
}

// AddEnhancement adds an enhancement option to a product.
func (h *ProductHandler) AddEnhancement(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	product, err := h.productService.AddEnhancement(id, body.Name, body.Price)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(models.MapProduct(product))
}

// DeleteEnhancement removes a product enhancement.
func (h *ProductHandler) DeleteEnhancement(c *fiber.Ctx) error {
	enhancementId := c.Params("enhancementId")

	product, err := h.productService.DeleteEnhancement(enhancementId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(models.MapProduct(product))
}

// childListItem is the flat wire shape for cross-product children
// (variants and extras): the item plus its owning product.
type childListItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
}

// GetAllTypes returns every variant across all products, with owner names.
// Fase 0: 2 small queries (types + product names) instead of full FindAll
// with 5 relations.
func (h *ProductHandler) GetAllTypes(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 60)
	opts, names, err := h.productService.FindAllTypesLight()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	items := make([]childListItem, 0, len(opts))
	for _, opt := range opts {
		items = append(items, childListItem{
			ID:          opt.ID,
			Name:        opt.Name,
			Price:       opt.Price,
			ProductID:   opt.ProductID,
			ProductName: names[opt.ProductID],
		})
	}
	return c.JSON(items)
}

// GetAllEnhancements returns every extra across all products, with owner names.
// Fase 0: 2 small queries (extras + product names) instead of full FindAll.
func (h *ProductHandler) GetAllEnhancements(c *fiber.Ctx) error {
	middleware.SetPublicCache(c, 60)
	enhs, names, err := h.productService.FindAllEnhancementsLight()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	items := make([]childListItem, 0, len(enhs))
	for _, enh := range enhs {
		items = append(items, childListItem{
			ID:          enh.ID,
			Name:        enh.Name,
			Price:       enh.Price,
			ProductID:   enh.ProductID,
			ProductName: names[enh.ProductID],
		})
	}
	return c.JSON(items)
}

// AddType creates a variant (tipo) for one product.
func (h *ProductHandler) AddType(c *fiber.Ctx) error {
	id := c.Params("id")

	var body struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}

	product, err := h.productService.AddType(id, body.Name, body.Price)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProductNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		case errors.Is(err, services.ErrTypeNameInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}
	}

	return c.JSON(models.MapProduct(product))
}

// DeleteType removes a variant from its product.
func (h *ProductHandler) DeleteType(c *fiber.Ctx) error {
	typeID := c.Params("typeId")

	product, err := h.productService.DeleteType(typeID)
	if err != nil {
		if errors.Is(err, services.ErrTypeNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(models.MapProduct(product))
}
