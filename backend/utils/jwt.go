package utils

import (
	"project/backend/config"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

func GenerateJWTToken(userID uint, cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func ExtractUserIDFromToken(c *fiber.Ctx, cfg *config.Config) (uint, error) {
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Missing authorization token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Invalid token claims")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID in token")
	}

	return uint(userIDFloat), nil
}
