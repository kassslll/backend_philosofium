package controllers

import (
	"errors"
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
)

// LoginRequest represents user login credentials
// @Description User login request payload
type LoginRequest struct {
	Username string `json:"username" example:"john_doe"`    // User's username
	Password string `json:"password" example:"password123"` // User's password
}

// LoginResponse represents successful login response
// @Description Authentication response with JWT token
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // JWT token
	User  struct {
		ID       uint   `json:"id" example:"1"`                   // User ID
		Username string `json:"username" example:"john_doe"`      // Username
		Email    string `json:"email" example:"john@example.com"` // User email
	} `json:"user"` // User information
}

// ErrorResponse represents error response
// @Description Standard error response format
type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid credentials"`               // Error message
	Message string `json:"message,omitempty" example:"Authentication failed"` // Additional message
}

type AuthController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAuthController(db *gorm.DB, cfg *config.Config) *AuthController {
	return &AuthController{DB: db, Cfg: cfg}
}

func (ac *AuthController) Register(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not hash password",
		})
	}
	user.PasswordHash = string(hashedPassword)

	// Create user
	if err := ac.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create user",
		})
	}

	// Generate JWT token
	token, err := utils.GenerateJWTToken(user.ID, ac.Cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (ac *AuthController) Login(c *fiber.Ctx) error {
	type LoginInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Find user
	var user models.User
	if err := ac.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Generate JWT token
	token, err := utils.GenerateJWTToken(user.ID, ac.Cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	// Update login history
	loginHistory := models.LoginHistory{
		UserID:    user.ID,
		LoginTime: time.Now(),
	}
	ac.DB.Create(&loginHistory)

	// Update user progress streak
	var userProgress models.UserProgress
	if err := ac.DB.Where("user_id = ?", user.ID).First(&userProgress).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userProgress = models.UserProgress{
				UserID:     user.ID,
				LastActive: time.Now(),
				StreakDays: 1,
			}
			ac.DB.Create(&userProgress)
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not query database",
			})
		}
	} else {
		// Check if last active was yesterday to maintain streak
		if time.Since(userProgress.LastActive) < 48*time.Hour {
			userProgress.StreakDays++
		} else {
			userProgress.StreakDays = 1
		}
		userProgress.LastActive = time.Now()
		ac.DB.Save(&userProgress)
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}
