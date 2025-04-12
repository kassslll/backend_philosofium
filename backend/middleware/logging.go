package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func LoggingMiddleware(logger *log.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Передаем управление следующему обработчику
		err := c.Next()

		// Логируем информацию о запросе
		logger.Printf(
			"[%s] %s %s %s %d %v",
			time.Now().Format("2006-01-02 15:04:05"),
			c.IP(),
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			time.Since(start),
		)

		return err
	}
}