package main

import (
	"log"
	"os"
	"refer_api/database"
	"refer_api/gateway"
	"refer_api/handlers"
	"refer_api/models"
	"refer_api/repository"
	"refer_api/routes"
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found or error reading it, relying on system environment variables")
	}

	// Initialize Database Connection
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Register many-to-many join models so Bun can resolve them with correct column names
	database.DB.RegisterModel((*models.UserRole)(nil))
	database.DB.RegisterModel((*models.CartItemEnhancement)(nil))

	// Auto-migrate tables (PostgreSQL - creates if not exists)
	if err := database.Migrate(database.DB); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Ensure single admin user exists (admin@admin.com / admin123@) — created once, idempotent
	if err := database.EnsureAdmin(database.DB); err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	// Initialize SocketGateway
	socketGateway := gateway.InitSocketGateway()

	// Initialize Layers (Clean Architecture)
	userRepo := repository.NewUserRepository(database.DB)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)

	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	productRepo := repository.NewProductRepository(database.DB)
	productService := services.NewProductService(productRepo)
	productHandler := handlers.NewProductHandler(productService)

	starRepo := repository.NewStarRepository(database.DB)
	starService := services.NewStarService(starRepo)
	starHandler := handlers.NewStarHandler(starService)

	dashboardService := services.NewDashboardService(database.DB, userRepo, starRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	searchService := services.NewSearchService(productRepo)
	searchHandler := handlers.NewSearchHandler(searchService)

	configHandler := handlers.NewConfigHandler()

	locationRepo := repository.NewLocationRepository(database.DB)
	locationService := services.NewLocationService(locationRepo)
	locationHandler := handlers.NewLocationHandler(locationService)

	paymentService := services.NewPaymentService()
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	orderRepo := repository.NewOrderRepository(database.DB)
	orderService := services.NewOrderService(orderRepo, productRepo)
	orderHandler := handlers.NewOrderHandler(orderService)

	cartRepo := repository.NewCartRepository(database.DB)
	cartService := services.NewCartService(cartRepo)
	gateway.CartServiceInstance = cartService

	app := fiber.New()

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:4200,https://refer.cobyzero.com,https://refer-api.cobyzero.com"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: true,
	}))

	// Serve uploaded files statically
	app.Static("/uploads", "./uploads")

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Route Socket.io requests using adaptor
	app.All("/socket.io/*", adaptor.HTTPHandler(socketGateway.GetServer().ServeHandler(nil)))

	// Setup Router Groups
	api := app.Group("/api")

	// Register Routes
	routes.RegisterAuthRoutes(api, authHandler)
	routes.RegisterUserRoutes(api, userHandler)
	routes.RegisterProductRoutes(api, productHandler)
	routes.RegisterStarRoutes(api, starHandler)
	routes.RegisterDashboardRoutes(api, dashboardHandler)
	routes.RegisterSearchRoutes(api, searchHandler)
	routes.RegisterConfigRoutes(api, configHandler)
	routes.RegisterLocationRoutes(api, locationHandler)
	routes.RegisterPaymentRoutes(api, paymentHandler)
	routes.RegisterOrderRoutes(api, orderHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(app.Listen(":" + port))
}
