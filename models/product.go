package models

import (
	"time"

	"github.com/uptrace/bun"
)

type ProductCategory struct {
	bun.BaseModel `bun:"table:product_categories"`

	ID        string  `bun:"id,pk,type:char(36)" json:"id"`
	Name      string  `bun:"name,unique,notnull" json:"name"`
	IconURL   *string `bun:"iconUrl" json:"iconUrl"`
	ImageURL  *string `bun:"imageUrl" json:"imageUrl"`
	BannerURL *string `bun:"bannerUrl" json:"bannerUrl"`
}

type ProductSize struct {
	bun.BaseModel `bun:"table:product_sizes"`

	ID   string `bun:"id,pk,type:char(36)" json:"id"`
	Name string `bun:"name,unique,notnull" json:"name"`
}

type ProductAvailableSize struct {
	bun.BaseModel `bun:"table:product_available_sizes"`

	ID        string      `bun:"id,pk,type:char(36)" json:"id"`
	ProductID string      `bun:"productId,notnull" json:"productId"`
	SizeID    string      `bun:"sizeId,notnull" json:"sizeId"`
	Price     float64     `bun:"price" json:"price"`
	Size      ProductSize `bun:"rel:belongs-to,join:sizeId=id" json:"size"`
}

type ProductType struct {
	bun.BaseModel `bun:"table:product_types"`

	ID        string  `bun:"id,pk,type:char(36)" json:"id"`
	Name      string  `bun:"name,notnull" json:"name"`
	Price     float64 `bun:"price,default:0" json:"price"`
	ProductID string  `bun:"productId,notnull" json:"productId"`
}

type ProductEnhancement struct {
	bun.BaseModel `bun:"table:product_enhancements"`

	ID        string  `bun:"id,pk,type:char(36)" json:"id"`
	Name      string  `bun:"name,notnull" json:"name"`
	Price     float64 `bun:"price,default:0" json:"price"`
	ProductID string  `bun:"productId,notnull" json:"productId"`
}

type Product struct {
	bun.BaseModel `bun:"table:products"`

	ID           string                 `bun:"id,pk,type:char(36)" json:"id"`
	Name         string                 `bun:"name,notnull" json:"name"`
	Description  string                 `bun:"description,notnull" json:"description"`
	CategoryID   *string                `bun:"categoryId" json:"categoryId"`
	Price        float64                `bun:"price,notnull" json:"price"`
	ImageURL     *string                `bun:"imageUrl" json:"imageUrl"`
	IsAvailable  bool                   `bun:"isAvailable,default:true" json:"isAvailable"`
	StarsReward  int                    `bun:"starsReward,default:0" json:"starsReward"`
	Rating       float64                `bun:"rating,default:0" json:"rating"`
	NumReviews   int                    `bun:"numReviews,default:0" json:"numReviews"`
	CreatedAt    time.Time              `bun:"createdAt" json:"createdAt"`

	Category     *ProductCategory       `bun:"rel:belongs-to,join:categoryId=id" json:"category,omitempty"`
	SizeOptions  []ProductAvailableSize `bun:"rel:has-many,join:id=productId" json:"sizeOptions,omitempty"`
	Options      []ProductType          `bun:"rel:has-many,join:id=productId" json:"options,omitempty"`
	Enhancements []ProductEnhancement   `bun:"rel:has-many,join:id=productId" json:"enhancements,omitempty"`
}

type SeasonalProduct struct {
	bun.BaseModel `bun:"table:seasonal_products"`

	ID        string   `bun:"id,pk,type:char(36)" json:"id"`
	ProductID string   `bun:"productId,notnull" json:"productId"`
	Product   *Product `bun:"rel:belongs-to,join:productId=id" json:"product"`
}

// Mapped Response DTOs (unchanged)
type MappedSizeOption struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type MappedTypeOption struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type MappedEnhancementOpt struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type MappedProductResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Price          float64                `json:"price"`
	ImageURL       *string                `json:"imageUrl"`
	IsAvailable    bool                   `json:"isAvailable"`
	StarsReward    int                    `json:"starsReward"`
	Rating         float64                `json:"rating"`
	NumReviews     int                    `json:"numReviews"`
	CreatedAt      time.Time              `json:"createdAt"`
	Category       string                 `json:"category"`
	CategoryIcon   *string                `json:"categoryIcon"`
	AvailableSizes []MappedSizeOption     `json:"availableSizes"`
	Types          []MappedTypeOption     `json:"types"`
	Enhancements   []MappedEnhancementOpt `json:"enhancements"`
}

// MapProduct flattens a Product with relations into MappedProductResponse
func MapProduct(p *Product) *MappedProductResponse {
	if p == nil {
		return nil
	}

	categoryName := "All"
	var categoryIcon *string
	if p.Category != nil {
		categoryName = p.Category.Name
		categoryIcon = p.Category.IconURL
	}

	availableSizes := []MappedSizeOption{}
	for _, sizeOpt := range p.SizeOptions {
		availableSizes = append(availableSizes, MappedSizeOption{
			ID:    sizeOpt.Size.ID,
			Name:  sizeOpt.Size.Name,
			Price: sizeOpt.Price,
		})
	}

	types := []MappedTypeOption{}
	for _, opt := range p.Options {
		types = append(types, MappedTypeOption{
			ID:    opt.ID,
			Name:  opt.Name,
			Price: opt.Price,
		})
	}

	enhancements := []MappedEnhancementOpt{}
	for _, enh := range p.Enhancements {
		enhancements = append(enhancements, MappedEnhancementOpt{
			ID:    enh.ID,
			Name:  enh.Name,
			Price: enh.Price,
		})
	}

	return &MappedProductResponse{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Price:          p.Price,
		ImageURL:       p.ImageURL,
		IsAvailable:    p.IsAvailable,
		StarsReward:    p.StarsReward,
		Rating:         p.Rating,
		NumReviews:     p.NumReviews,
		CreatedAt:      p.CreatedAt,
		Category:       categoryName,
		CategoryIcon:   categoryIcon,
		AvailableSizes: availableSizes,
		Types:          types,
		Enhancements:   enhancements,
	}
}
