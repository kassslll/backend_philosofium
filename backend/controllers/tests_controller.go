package controllers

import (
	"backend/backend/utils"
	"backend/config"
	"backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type TestsController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewTestsController(db *gorm.DB, cfg *config.Config) *TestsController {
	return &TestsController{DB: db, Cfg: cfg}
}

func (tc *TestsController) GetUserTests(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var tests []models.Test
	tc.DB.Joins("JOIN user_test_progress ON user_test_progress.test_id = tests.id").
		Where("user_test_progress.user_id = ?", userID).
		Find(&tests)

	var result []fiber.Map
	for _, test := range tests {
		var progress models.UserTestProgress
		tc.DB.Where("user_id = ? AND test_id = ?", userID, test.ID).First(&progress)

		result = append(result, fiber.Map{
			"id":            test.ID,
			"title":         test.Title,
			"progress":      float64(progress.CorrectAnswers) / float64(progress.QuestionsAnswered) * 100,
			"group":         test.RecommendedFor,
			"questions":     len(test.Questions),
			"answered":      progress.QuestionsAnswered,
			"correct":       progress.CorrectAnswers,
			"score":         progress.Score,
			"last_attempt":  progress.LastAttempt,
			"attempts_used": progress.AttemptsUsed,
		})
	}

	return c.JSON(result)
}

func (tc *TestsController) GetAvailableTests(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get query parameters
	topic := c.Query("topic")
	university := c.Query("university")

	query := tc.DB.Model(&models.Test{}).Where("access_level = 'public'")

	if topic != "" {
		query = query.Where("topic LIKE ?", "%"+topic+"%")
	}

	if university != "" {
		query = query.Where("university LIKE ?", "%"+university+"%")
	}

	var tests []models.Test
	query.Find(&tests)

	var result []fiber.Map
	for _, test := range tests {
		var progress models.UserTestProgress
		tc.DB.Where("user_id = ? AND test_id = ?", userID, test.ID).First(&progress)

		result = append(result, fiber.Map{
			"id":          test.ID,
			"title":       test.Title,
			"progress":    float64(progress.CorrectAnswers) / float64(progress.QuestionsAnswered) * 100,
			"group":       test.RecommendedFor,
			"description": test.ShortDesc,
			"difficulty":  test.Difficulty,
			"university":  test.University,
			"topic":       test.Topic,
			"author":      test.AuthorID,
			"logo_url":    test.LogoURL,
		})
	}

	return c.JSON(result)
}

// ... (остальные методы контроллера тестов)
