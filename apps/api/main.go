package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"

	"hotspotos/packages/database"
	"hotspotos/packages/logger"
)

func main() {
	logger.Init("development")
	logger.Info("Starting HotspotOS API Gateway...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=hotspotos port=5432 sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super_secret_jwt_key_hotspotos"
	}

	// 1. Connect to PostgreSQL
	db, err := database.ConnectPostgres(dsn)
	if err != nil {
		logger.Error("API server DB connection failed", "error", err)
		os.Exit(1)
	}

	// Run migrations and seed data
	if err := database.Migrate(db); err != nil {
		logger.Error("API server database migration failed", "error", err)
		os.Exit(1)
	}

	// 2. Connect to Redis
	_, err = database.ConnectRedis(redisAddr, "", 0)
	if err != nil {
		logger.Error("API server Redis connection failed", "error", err)
		os.Exit(1)
	}

	// 3. Start Redis Pub/Sub listener for real-time WebSocket dashboard sync
	go startRedisListener()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Enable CORS for dashboard interface
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Internal-Token",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Log HTTP requests
	app.Use(func(c *fiber.Ctx) error {
		logger.Info("API Gateway Request", "method", c.Method(), "path", c.Path())
		return c.Next()
	})

	// Serve static files for the captive portal splash screen
	app.Static("/", "./services/captive-portal/dist")

	// Public Routes (Captive Portal client-facing APIs)
	app.Post("/auth/login", handleLogin)
	app.Get("/plans", handleGetPlans)
	app.Post("/payments/stk", handleSTKPush)
	app.Post("/payments/callback", handlePaymentCallback)

	// Admin-Only Protected Routes
	admin := app.Group("/admin", JWTMiddleware(jwtSecret))
	admin.Get("/devices", handleGetDevices)
	admin.Get("/analytics", handleGetAnalytics)
	admin.Get("/reports", handleGetReports)
	admin.Post("/disconnect", handleAdminDisconnect)
	admin.Post("/plans", handleAdminCreatePlan)
	admin.Put("/plans/:id", handleAdminUpdatePlan)
	admin.Delete("/plans/:id", handleAdminDeletePlan)

	// Internal Inter-service APIs (Secured by preshared key)
	internal := app.Group("/internal", InternalAuthMiddleware())
	internal.Post("/sessions/authorize", handleInternalAuthorize)

	// WebSocket handler for real-time notifications
	app.Get("/ws", websocket.New(handleWebSocket))

	// Catch-all route for captive portal detection & redirection
	app.Get("/*", handleCaptivePortalRedirect)

	logger.Info("API Gateway listening", "port", port)
	if err := app.Listen(":" + port); err != nil {
		logger.Error("API Gateway stopped", "error", err)
	}
}
