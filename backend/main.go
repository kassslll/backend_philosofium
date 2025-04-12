package main

import (
	"log"
	"project/backend/config"
	"project/backend/middleware"
	"project/backend/routes"
	"project/backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize database
	db, err := utils.InitDB(cfg)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	// Initialize logger
	logger := utils.InitLogger()

	// Create Fiber app
	app := fiber.New()

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))
	app.Use(middleware.LoggingMiddleware(logger))

	// Setup routes
	routes.SetupRoutes(app, db, cfg)

	// Start server
	log.Fatal(app.Listen(":" + cfg.ServerPort))
}
