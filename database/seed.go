package database

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"refer_api/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// Seed cleans the database and populates it with default seed data.
func Seed(db *bun.DB) error {
	ctx := context.Background()

	// Disable FK checks for PostgreSQL (requires superuser / table owner; fallback gracefully)
	// Using TRUNCATE ... CASCADE is preferred for Postgres
	_, _ = db.ExecContext(ctx, "SET session_replication_role = 'replica'")
	defer func() {
		_, _ = db.ExecContext(ctx, "SET session_replication_role = 'origin'")
	}()

	// Truncate all tables (CASCADE handles FK dependencies)
	tables := []string{
		"users_roles", "roles", "star_transactions", "redeemed_rewards", "rewards",
		"cart_item_enhancements", "cart_items", "order_items", "orders",
		"product_available_sizes", "product_sizes", "product_types", "product_enhancements",
		"seasonal_products", "products", "product_categories", "locations", "users",
	}
	for _, table := range tables {
		// Use TRUNCATE CASCADE for Postgres; fallback to DELETE if table missing
		if _, err := db.ExecContext(ctx, fmt.Sprintf(`TRUNCATE TABLE "%s" CASCADE`, table)); err != nil {
			// Fallback: DELETE (handles non-existing table gracefully)
			if _, delErr := db.ExecContext(ctx, "DELETE FROM \""+table+"\""); delErr != nil {
				// If both fail, check if table simply doesn't exist yet (first run) -> ignore
				// But if it's a real error, return it
				if !strings.Contains(err.Error(), "does not exist") && !strings.Contains(delErr.Error(), "does not exist") {
					return fmt.Errorf("failed to clean %s: %w (truncate err: %v)", table, delErr, err)
				}
			}
		}
	}

	// 1. Seed Roles & Users
	adminRole := &models.Role{
		ID:   uuid.New().String(),
		Name: "ADMIN",
	}
	userRole := &models.Role{
		ID:   uuid.New().String(),
		Name: "USER",
	}

	if _, err := db.NewInsert().Model(adminRole).Exec(ctx); err != nil {
		return err
	}
	if _, err := db.NewInsert().Model(userRole).Exec(ctx); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), 10)
	if err != nil {
		return err
	}

	photoAdmin := "https://i.pravatar.cc/150?u=sebastian"
	admin := &models.User{
		ID:          "00000000-0000-0000-0000-000000000001",
		Name:        "Sebastian Abal",
		Email:       "admin@admin.com",
		Password:    string(hashedPassword),
		PhotoURL:    &photoAdmin,
		KeepUpdated: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	photoUser := "https://i.pravatar.cc/150?u=regular"
	user := &models.User{
		ID:          "00000000-0000-0000-0000-000000000002",
		Name:        "Regular User",
		Email:       "user@user.com",
		Password:    string(hashedPassword),
		PhotoURL:    &photoUser,
		KeepUpdated: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if _, err := db.NewInsert().Model(admin).Exec(ctx); err != nil {
		return err
	}
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		return err
	}

	// Insert UserRoles pivots
	adminPivot := &models.UserRole{
		UserID: admin.ID,
		RoleID: adminRole.ID,
	}
	userPivot := &models.UserRole{
		UserID: user.ID,
		RoleID: userRole.ID,
	}
	if _, err := db.NewInsert().Model(adminPivot).Exec(ctx); err != nil {
		return err
	}
	if _, err := db.NewInsert().Model(userPivot).Exec(ctx); err != nil {
		return err
	}

	// Welcome star transactions
	tx1 := &models.StarTransaction{
		ID:     uuid.New().String(),
		UserID: admin.ID,
		Type:   "earn",
		Title:  "Welcome Gift",
		Stars:  85,
		Date:   time.Now(),
	}
	tx2 := &models.StarTransaction{
		ID:     uuid.New().String(),
		UserID: user.ID,
		Type:   "earn",
		Title:  "Welcome Gift",
		Stars:  10,
		Date:   time.Now(),
	}
	if _, err := db.NewInsert().Model(tx1).Exec(ctx); err != nil {
		return err
	}
	if _, err := db.NewInsert().Model(tx2).Exec(ctx); err != nil {
		return err
	}

	// 2. Seed Categories
	categories := []models.ProductCategory{
		{
			ID:        uuid.New().String(),
			Name:      "Café",
			IconURL:   stringPtr("https://cdn-icons-png.flaticon.com/512/924/924514.png"),
			ImageURL:  stringPtr("https://images.unsplash.com/photo-1509042239860-f550ce710b93?auto=format&fit=crop&w=800&q=80"),
			BannerURL: stringPtr("https://images.unsplash.com/photo-1498804103079-a6351b050096?auto=format&fit=crop&w=1200&q=80"),
		},
		{
			ID:        uuid.New().String(),
			Name:      "Té",
			IconURL:   stringPtr("https://cdn-icons-png.flaticon.com/512/3504/3504827.png"),
			ImageURL:  stringPtr("https://images.unsplash.com/photo-1576092768241-dec231879fc3?auto=format&fit=crop&w=800&q=80"),
			BannerURL: stringPtr("https://images.unsplash.com/photo-1597481499750-3e6b22637e12?auto=format&fit=crop&w=1200&q=80"),
		},
		{
			ID:        uuid.New().String(),
			Name:      "Panadería",
			IconURL:   stringPtr("https://cdn-icons-png.flaticon.com/512/3014/3014534.png"),
			ImageURL:  stringPtr("https://images.unsplash.com/photo-1509440159596-0249088772ff?auto=format&fit=crop&w=800&q=80"),
			BannerURL: stringPtr("https://images.unsplash.com/photo-1517433456452-f9633a875f6f?auto=format&fit=crop&w=1200&q=80"),
		},
		{
			ID:        uuid.New().String(),
			Name:      "Mercancía",
			IconURL:   stringPtr("https://cdn-icons-png.flaticon.com/512/3081/3081559.png"),
			ImageURL:  stringPtr("https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?auto=format&fit=crop&w=800&q=80"),
			BannerURL: stringPtr("https://images.unsplash.com/photo-1563245372-f21724e3856d?auto=format&fit=crop&w=1200&q=80"),
		},
	}
	for i := range categories {
		if _, err := db.NewInsert().Model(&categories[i]).Exec(ctx); err != nil {
			return err
		}
	}

	getCatID := func(name string) *string {
		for _, cat := range categories {
			if strings.ToLower(cat.Name) == strings.ToLower(name) {
				return &cat.ID
			}
		}
		return nil
	}

	// 3. Seed Products
	productsData := []struct {
		name        string
		description string
		category    string
		price       float64
		rating      float64
		numReviews  int
		starsReward int
		imageUrl    string
	}{
		{"Cascara Cold Brew", "Cherry-steeped morning bliss", "Café", 6.50, 4.9, 124, 15, "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=400&h=400"},
		{"Spiced Autumn Latte", "Warming seasonal notes", "Café", 5.75, 4.8, 89, 12, "https://images.unsplash.com/photo-1570968915860-54d5c301fa9f?auto=format&fit=crop&w=400&h=400"},
		{"Honey Almond Frappuccino", "Sweet and nutty refreshment", "Café", 6.25, 4.7, 45, 14, "https://images.unsplash.com/photo-1572490122747-3968b75cc699?auto=format&fit=crop&w=400&h=400"},
		{"Matcha Mint Fizz", "Sparkling green tea energy", "Té", 5.50, 4.6, 32, 10, "https://images.unsplash.com/photo-1515823064-d6e0c04616a7?auto=format&fit=crop&w=400&h=400"},
		{"Vanilla Sweet Cream", "Smooth and creamy classic", "Café", 5.25, 4.7, 210, 10, "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=400&h=400"},
		{"Caramel Macchiato", "Espresso with silky caramel", "Café", 5.50, 4.8, 180, 12, "https://images.unsplash.com/photo-1485808191679-5f86510681a2?auto=format&fit=crop&w=400&h=400"},
		{"Butter Croissant", "Flaky, buttery French specialty", "Panadería", 3.50, 4.9, 340, 8, "https://images.unsplash.com/photo-1555507036-ab1f4038808a?auto=format&fit=crop&w=400&h=400"},
		{"Blueberry Muffin", "Bursting with fresh berries", "Panadería", 3.25, 4.6, 156, 8, "https://images.unsplash.com/photo-1607958996333-41aef7caefaa?auto=format&fit=crop&w=400&h=400"},
		{"Chocolate Chip Cookie", "Warm with gooey chocolate", "Panadería", 2.75, 4.9, 245, 5, "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?auto=format&fit=crop&w=400&h=400"},
	}

	var products []models.Product
	for _, p := range productsData {
		prod := models.Product{
			ID:          uuid.New().String(),
			Name:        p.name,
			Description: p.description,
			CategoryID:  getCatID(p.category),
			Price:       p.price,
			ImageURL:    stringPtr(p.imageUrl),
			IsAvailable: true,
			StarsReward: p.starsReward,
			Rating:      p.rating,
			NumReviews:  p.numReviews,
			CreatedAt:   time.Now(),
		}
		if _, err := db.NewInsert().Model(&prod).Exec(ctx); err != nil {
			return err
		}
		products = append(products, prod)
	}

	// Seasonal Products
	seasonalNames := map[string]bool{
		"Spiced Autumn Latte":      true,
		"Honey Almond Frappuccino": true,
		"Matcha Mint Fizz":         true,
	}
	for _, prod := range products {
		if seasonalNames[prod.Name] {
			sp := &models.SeasonalProduct{
				ID:        uuid.New().String(),
				ProductID: prod.ID,
			}
			if _, err := db.NewInsert().Model(sp).Exec(ctx); err != nil {
				return err
			}
		}
	}

	// 4. Seed Sizes
	sizeNames := []string{"Normal", "Grande", "Extragrande", "Expreso"}
	var sizes []models.ProductSize
	for _, sName := range sizeNames {
		sz := models.ProductSize{
			ID:   uuid.New().String(),
			Name: sName,
		}
		if _, err := db.NewInsert().Model(&sz).Exec(ctx); err != nil {
			return err
		}
		sizes = append(sizes, sz)
	}

	findSize := func(name string) *models.ProductSize {
		for _, s := range sizes {
			if s.Name == name {
				return &s
			}
		}
		return nil
	}

	r := rand.New(rand.NewSource(99))
	for _, prod := range products {
		catName := ""
		for _, cat := range categories {
			if prod.CategoryID != nil && *prod.CategoryID == cat.ID {
				catName = cat.Name
				break
			}
		}

		if catName == "Café" || catName == "Té" {
			if strings.Contains(strings.ToLower(prod.Name), "espresso") {
				expreso := findSize("Expreso")
				if expreso != nil {
					pas := &models.ProductAvailableSize{
						ID:        uuid.New().String(),
						ProductID: prod.ID,
						SizeID:    expreso.ID,
						Price:     r.Float64() * 2.0,
					}
					if _, err := db.NewInsert().Model(pas).Exec(ctx); err != nil {
						return err
					}
				}
			} else {
				for _, size := range sizes {
					if size.Name != "Expreso" {
						pas := &models.ProductAvailableSize{
							ID:        uuid.New().String(),
							ProductID: prod.ID,
							SizeID:    size.ID,
							Price:     r.Float64() * 2.0,
						}
						if _, err := db.NewInsert().Model(pas).Exec(ctx); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// 5. Seed Options (ProductType)
	for _, prod := range products {
		catName := ""
		for _, cat := range categories {
			if prod.CategoryID != nil && *prod.CategoryID == cat.ID {
				catName = cat.Name
				break
			}
		}

		if catName == "Café" || catName == "Té" {
			milks := []struct {
				name  string
				price float64
			}{
				{"Original", 0.0},
				{"Oat Milk", 0.5},
				{"Almond Milk", 0.5},
				{"Soy Milk", 0.5},
			}
			for _, milk := range milks {
				opt := &models.ProductType{
					ID:        uuid.New().String(),
					Name:      milk.name,
					Price:     milk.price,
					ProductID: prod.ID,
				}
				if _, err := db.NewInsert().Model(opt).Exec(ctx); err != nil {
					return err
				}
			}
		} else if catName == "Panadería" {
			options := []string{"Warmed", "Not Warmed"}
			for _, option := range options {
				opt := &models.ProductType{
					ID:        uuid.New().String(),
					Name:      option,
					Price:     0,
					ProductID: prod.ID,
				}
				if _, err := db.NewInsert().Model(opt).Exec(ctx); err != nil {
					return err
				}
			}
		}
	}

	// 6. Seed Enhancements
	for _, prod := range products {
		catName := ""
		for _, cat := range categories {
			if prod.CategoryID != nil && *prod.CategoryID == cat.ID {
				catName = cat.Name
				break
			}
		}

		if catName == "Café" || catName == "Té" {
			enhancements := []struct {
				name  string
				price float64
			}{
				{"Shot de espresso extra", 1.25},
				{"Endulzando", 3.00},
				{"Caramel Drizzle", 0.50},
			}
			for _, enh := range enhancements {
				pe := &models.ProductEnhancement{
					ID:        uuid.New().String(),
					Name:      enh.name,
					Price:     enh.price,
					ProductID: prod.ID,
				}
				if _, err := db.NewInsert().Model(pe).Exec(ctx); err != nil {
					return err
				}
			}
		}
	}

	// 7. Seed Rewards
	findProdID := func(namePart string) string {
		for _, p := range products {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(namePart)) {
				return p.ID
			}
		}
		if len(products) > 0 {
			return products[0].ID
		}
		return ""
	}

	rewards := []models.Reward{
		{
			ID:            uuid.New().String(),
			Title:         "Extra Espresso Shot",
			Description:   "Perfect for that extra kick",
			StarsRequired: 25,
			Icon:          stringPtr("add_circle_outline"),
			ImageURL:      stringPtr("https://images.unsplash.com/photo-1510707577719-ae7c14805e3a?auto=format&fit=crop&w=400&h=400"),
			ProductID:     stringPtr(findProdID("Cascara Cold Brew")),
			IsActive:      true,
		},
		{
			ID:            uuid.New().String(),
			Title:         "Fresh Bakery Item",
			Description:   "Any croissant, muffin or cookie",
			StarsRequired: 50,
			Icon:          stringPtr("bakery_dining_rounded"),
			ImageURL:      stringPtr("https://images.unsplash.com/photo-1555507036-ab1f4038808a?auto=format&fit=crop&w=400&h=400"),
			ProductID:     stringPtr(findProdID("Butter Croissant")),
			IsActive:      true,
		},
		{
			ID:            uuid.New().String(),
			Title:         "Handcrafted Drink",
			Description:   "Any coffee or tea prepared by our baristas",
			StarsRequired: 100,
			Icon:          stringPtr("local_cafe_rounded"),
			ImageURL:      stringPtr("https://images.unsplash.com/photo-1541167760496-162955ed8a9f?auto=format&fit=crop&w=400&h=400"),
			ProductID:     stringPtr(findProdID("Spiced Autumn Latte")),
			IsActive:      true,
		},
		{
			ID:            uuid.New().String(),
			Title:         "Lunch Sandwich",
			Description:   "Try our gourmet avocado toast or panini",
			StarsRequired: 200,
			Icon:          stringPtr("lunch_dining_rounded"),
			ImageURL:      stringPtr("https://images.unsplash.com/photo-1538087833039-58ae2ff355d2?auto=format&fit=crop&w=400&h=400"),
			ProductID:     stringPtr(findProdID("Blueberry Muffin")),
			IsActive:      true,
		},
		{
			ID:            uuid.New().String(),
			Title:         "Bag of Whole Bean Coffee",
			Description:   "250g of our signature roast",
			StarsRequired: 400,
			Icon:          stringPtr("shopping_bag_rounded"),
			ImageURL:      stringPtr("https://images.unsplash.com/photo-1559056199-641a0ac8b55e?auto=format&fit=crop&w=400&h=400"),
			ProductID:     stringPtr(findProdID("Chocolate Chip Cookie")),
			IsActive:      true,
		},
	}
	for i := range rewards {
		if _, err := db.NewInsert().Model(&rewards[i]).Exec(ctx); err != nil {
			return err
		}
	}

	// 8. Seed Locations
	locations := []models.Location{
		{
			ID:        uuid.New().String(),
			Name:      "Downtown Studio",
			Address:   "1248 Editorial Way, Suite 4",
			City:      stringPtr("Portland"),
			State:     stringPtr("OR"),
			ZipCode:   stringPtr("97204"),
			Latitude:  float64Ptr(45.523062),
			Longitude: float64Ptr(-122.676482),
			IsActive:  true,
		},
		{
			ID:        uuid.New().String(),
			Name:      "Pearl District Roastery",
			Address:   "815 NW 13th Ave",
			City:      stringPtr("Portland"),
			State:     stringPtr("OR"),
			ZipCode:   stringPtr("97209"),
			Latitude:  float64Ptr(45.5283),
			Longitude: float64Ptr(-122.6841),
			IsActive:  true,
		},
		{
			ID:        uuid.New().String(),
			Name:      "Eastside Garden",
			Address:   "2239 SE Hawthorne Blvd",
			City:      stringPtr("Portland"),
			State:     stringPtr("OR"),
			ZipCode:   stringPtr("97214"),
			Latitude:  float64Ptr(45.5126),
			Longitude: float64Ptr(-122.6428),
			IsActive:  true,
		},
	}
	for i := range locations {
		if _, err := db.NewInsert().Model(&locations[i]).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
