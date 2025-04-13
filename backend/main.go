package main

import (
	"log"
	"project/backend/config"
	"project/backend/middleware"
	"project/backend/routes"
	"project/backend/utils"

	_ "project/backend/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title Learning Platform API
// @version 1.0
// @description API for educational platform
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:3000
// @BasePath /api
// @schemes http
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

	// Swagger
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:  "*",                           // Укажите явные домены
		AllowMethods:  "GET,POST,PUT,DELETE,OPTIONS", // Добавьте методы
		AllowHeaders:  "Origin,Content-Type,Accept,Authorization",
		ExposeHeaders: "Content-Length", // Доп. заголовки
		MaxAge:        86400,            // Кеширование CORS (сек)
	}))
	app.Use(middleware.LoggingMiddleware(logger))

	// Setup routes
	routes.SetupRoutes(app, db, cfg)

	app.Use(func(c *fiber.Ctx) error {
		// Логирование 404 ошибок
		logger.Printf("404 Not Found: %s %s", c.Method(), c.OriginalURL())

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Endpoint not found",
			"path":    c.Path(),
			"method":  c.Method(),
			"docs":    "http://" + c.Hostname() + "/swagger/index.html",
			"available_routes": []string{
				"/api/auth/login",
				"/api/auth/register",
				"/swagger",
			},
		})
	})
	// Start server
	log.Fatal(app.Listen(":" + cfg.ServerPort))
}
