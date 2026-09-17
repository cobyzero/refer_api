package main

import (
	"log"
	"os"
	"refer_api/database"
	"refer_api/gateway"
	"refer_api/handlers"
	"refer_api/middleware"
	"refer_api/models"
	"refer_api/repository"
	"refer_api/routes"
	"refer_api/services"
	"refer_api/supabase"

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

	// DB externa supabase/postgres: verifica roles/schemas auth/storage o corta
	// con mensaje explícito (evita PGRST000/3F000 crípticos después).
	if err := database.CheckSupabaseDB(database.DB); err != nil {
		log.Fatalf("Supabase DB check failed: %v", err)
	}

	// Buckets storage idempotentes (antes manual con make supabase-storage-init).
	if err := database.EnsureStorage(database.DB); err != nil {
		log.Fatalf("Failed to ensure storage buckets: %v", err)
	}

	// Initialize SocketGateway
	socketGateway := gateway.InitSocketGateway()

	// Supabase Auth is the only auth flow: refuse to boot without it.
	supaCfg := supabase.LoadConfig()
	if err := supaCfg.Validate(); err != nil {
		log.Fatalf("Supabase Auth is required: %v", err)
	}
	supaClient := supabase.NewClient(supaCfg)
	log.Println("Supabase Auth enabled")

	// Provision the seed admin in Supabase Auth, then mirror it in
	// public.users with the same id (idempotent, created once).
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@admin.com"
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123@"
	}
	authAdminID, err := supaClient.EnsureUser(adminEmail, adminPassword, "Admin")
	if err != nil {
		log.Fatalf("Failed to provision Supabase admin user: %v", err)
	}
	if err := database.EnsureAdmin(database.DB, authAdminID); err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	// Initialize Layers (Clean Architecture)
	userRepo := repository.NewUserRepository(database.DB)
	configRepo := repository.NewConfigRepository(database.DB)
	// Se inyecta configRepo para que registro/recuperación respeten los toggles
	// smtp_enabled/signup_email_enabled/recovery_email_enabled del módulo de
	// configuraciones (refer_web): si se deshabilitan, no se aplica SMTP.
	authService := services.NewSupabaseAuthService(userRepo, supaClient, supaCfg, configRepo)
	authHandler := handlers.NewAuthHandler(authService)

	userService := services.NewUserService(userRepo, supaClient)
	userHandler := handlers.NewUserHandler(userService)

	productRepo := repository.NewProductRepository(database.DB)
	productService := services.NewProductService(productRepo)
	productHandler := handlers.NewProductHandler(productService)

	// Uploads backend: Supabase Storage w/ transparent disk fallback.
	// Same Flutter contract either way ({imageUrl}/{photoUrl}).
	storageService := services.NewStorageService()
	if storageService.Enabled() {
		log.Println("Supabase Storage adapter enabled")
	} else {
		log.Println("Supabase Storage not configured; uploads use local disk")
	}
	userHandler.WithStorage(storageService)
	productHandler.WithStorage(storageService)

	collectionRepo := repository.NewCollectionRepository(database.DB)
	collectionService := services.NewCollectionService(collectionRepo, productRepo)
	collectionHandler := handlers.NewCollectionHandler(collectionService)

	starRepo := repository.NewStarRepository(database.DB)
	starService := services.NewStarService(starRepo)
	starHandler := handlers.NewStarHandler(starService)

	dashboardService := services.NewDashboardService(database.DB, userRepo, starRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	searchService := services.NewSearchService(productRepo)
	searchHandler := handlers.NewSearchHandler(searchService)

	configService := services.NewConfigService(configRepo)
	configHandler := handlers.NewConfigHandler(configService)

	locationRepo := repository.NewLocationRepository(database.DB)
	locationService := services.NewLocationService(locationRepo)
	locationHandler := handlers.NewLocationHandler(locationService)

	orderRepo := repository.NewOrderRepository(database.DB)
	// Fix #2: server-side pricing needs catalog + rates + locations + claims.
	orderService := services.NewOrderService(orderRepo, productRepo, configRepo, locationRepo, starRepo)
	orderHandler := handlers.NewOrderHandler(orderService)

	paymentService := services.NewPaymentService()
	// Fix #2: payment intents can charge a server-side order total (orderId).
	paymentHandler := handlers.NewPaymentHandler(paymentService, orderService)

	cartRepo := repository.NewCartRepository(database.DB)
	cartService := services.NewCartService(cartRepo)
	gateway.CartServiceInstance = cartService

	auditRepo := repository.NewAuditRepository(database.DB)
	auditService := services.NewAuditService(auditRepo)
	auditHandler := handlers.NewAuditHandler(auditService)
	// Cart socket mutations (gateway) also feed the audit log (IP unknown there).
	gateway.AuditFunc = func(userID, action string, detail map[string]interface{}) {
		_ = auditService.Log(userID, "", nil, action, "cart", "", 200, "", detail)
	}

	app := fiber.New()

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:4200,https://refer.cobyzero.com,https://refer-api.cobyzero.com"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowCredentials: true,
	}))

	// Serve uploaded files statically
	app.Static("/uploads", "./uploads")

	// Email OTP template for Supabase Auth (GoTrue fetches it over HTTP).
	// No secrets here — just the {{ .Token }} markup.
	app.Static("/templates", "./supabase/templates")

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Route Socket.io requests using adaptor
	app.All("/socket.io/*", adaptor.HTTPHandler(socketGateway.GetServer().ServeHandler(nil)))

	// Setup Router Groups
	api := app.Group("/api")

	// Auto-audit: records every mutating request (POST/PUT/PATCH/DELETE) with
	// actor (JWT locals), action, entity, status and IP. Post-Next logic runs
	// after inner JWT + handlers, so locals are populated. Never breaks requests.
	api.Use(middleware.AuditMiddleware(auditService))

	// Register Routes
	routes.RegisterAuthRoutes(api, authHandler)
	routes.RegisterUserRoutes(api, userHandler)
	routes.RegisterProductRoutes(api, productHandler)
	routes.RegisterCollectionRoutes(api, collectionHandler)
	routes.RegisterStarRoutes(api, starHandler)
	routes.RegisterDashboardRoutes(api, dashboardHandler)
	routes.RegisterSearchRoutes(api, searchHandler)
	routes.RegisterConfigRoutes(api, configHandler)
	routes.RegisterLocationRoutes(api, locationHandler)
	routes.RegisterPaymentRoutes(api, paymentHandler)
	routes.RegisterOrderRoutes(api, orderHandler)
	routes.RegisterAuditRoutes(api, auditHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(app.Listen(":" + port))
}
