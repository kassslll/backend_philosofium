package middleware

import (
	"backend/config"
	"backend/utils"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := utils.ExtractUserIDFromToken(c, cfg)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		return c.Next()
	}
}

func AdminMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := utils.ExtractUserIDFromToken(c, cfg)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Здесь должна быть проверка, что пользователь - администратор
		// Это пример, вам нужно реализовать проверку в вашей базе данных
		if userID != 1 { // Пример: предполагаем, что пользователь с ID 1 - администратор
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden - Admin access required",
			})
		}

		return c.Next()
	}
}
