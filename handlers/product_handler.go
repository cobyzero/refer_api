package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"refer_api/models"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// ProductHandler coordinates HTTP logic for product endpoints.
type ProductHandler struct {
	productService services.ProductService
}

// NewProductHandler constructs a new ProductHandler.
func NewProductHandler(productService services.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// GetAll returns all products in mapped structure.
func (h *ProductHandler) GetAll(c *fiber.Ctx) error {
	products, err := h.productService.FindAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	var mapped []models.MappedProductResponse
	for _, p := range products {
		mapped = append(mapped, *models.MapProduct(&p))
	}

	return c.JSON(mapped)
}

// GetTopRated returns top 5 rated products.
func (h *ProductHandler) GetTopRated(c *fiber.Ctx) error {
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
	categories, err := h.productService.FindCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(categories)
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

	return c.JSON(models.MapProduct(product))
}

// UploadPhoto saves product photo uploads to disk.
func (h *ProductHandler) UploadPhoto(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "No file uploaded or invalid format"})
	}

	randomBytes := make([]byte, 16)
	if _, randErr := rand.Read(randomBytes); randErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to generate unique filename"})
	}
	randomName := hex.EncodeToString(randomBytes)
	extension := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf("%s%s", randomName, extension)

	uploadDir := "./uploads"
	if _, pathErr := os.Stat(uploadDir); os.IsNotExist(pathErr) {
		if makeErr := os.Mkdir(uploadDir, 0755); makeErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create upload directory"})
		}
	}

	destPath := filepath.Join(uploadDir, newFilename)
	if saveErr := c.SaveFile(file, destPath); saveErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to save file"})
	}

	imageUrl := fmt.Sprintf("%s/uploads/%s", c.BaseURL(), newFilename)

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
