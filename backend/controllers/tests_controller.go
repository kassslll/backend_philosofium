package controllers

import (
	"encoding/json"
	"errors"
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"
	"strconv"
	"strings"
	"time"

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

func (tc *TestsController) GetTestDetails(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var test models.Test
	if err := tc.DB.Preload("Questions").Preload("Comments").First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var progress models.UserTestProgress
	tc.DB.Where("user_id = ? AND test_id = ?", userID, testID).First(&progress)

	// Parse question options from JSON string to array
	var questions []map[string]interface{}
	for _, q := range test.Questions {
		var options []string
		json.Unmarshal([]byte(q.Options), &options)

		questions = append(questions, map[string]interface{}{
			"id":          q.ID,
			"title":       q.Title,
			"description": q.Description,
			"question":    q.Question,
			"options":     options,
			"order":       q.SequenceOrder,
		})
	}

	return c.JSON(fiber.Map{
		"test": fiber.Map{
			"id":              test.ID,
			"title":           test.Title,
			"description":     test.Description,
			"short_desc":      test.ShortDesc,
			"difficulty":      test.Difficulty,
			"recommended":     test.RecommendedFor,
			"university":      test.University,
			"topic":           test.Topic,
			"logo_url":        test.LogoURL,
			"author":          test.AuthorID,
			"questions":       questions,
			"comments":        test.Comments,
			"completion_rate": test.CompletionRate,
		},
		"progress": progress,
	})
}

func (tc *TestsController) UpdateTestProgress(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	type AnswerInput struct {
		QuestionID uint `json:"question_id"`
		Answer     int  `json:"answer"`
	}

	type ProgressInput struct {
		Answers []AnswerInput `json:"answers"`
	}

	var input ProgressInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var test models.Test
	if err := tc.DB.Preload("Questions").First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var progress models.UserTestProgress
	if err := tc.DB.Where("user_id = ? AND test_id = ?", userID, testID).First(&progress).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			progress = models.UserTestProgress{
				UserID:            userID,
				TestID:            uint(testID),
				QuestionsAnswered: 0,
				CorrectAnswers:    0,
				Score:             0,
				AttemptsUsed:      0,
			}
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not query database",
			})
		}
	}

	// Check attempts
	var accessSettings models.TestAccessSettings
	tc.DB.Where("test_id = ?", testID).First(&accessSettings)
	if progress.AttemptsUsed >= accessSettings.AttemptsAllowed && accessSettings.AttemptsAllowed > 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "No attempts left",
		})
	}

	// Process answers
	correctAnswers := 0
	for _, answer := range input.Answers {
		var question models.TestQuestion
		if err := tc.DB.Where("id = ? AND test_id = ?", answer.QuestionID, testID).First(&question).Error; err != nil {
			continue
		}

		if answer.Answer == question.CorrectAnswer {
			correctAnswers++
		}
	}

	progress.QuestionsAnswered = len(input.Answers)
	progress.CorrectAnswers = correctAnswers
	progress.Score = float64(correctAnswers) / float64(len(test.Questions)) * 100
	progress.AttemptsUsed++
	progress.LastAttempt = time.Now().Format(time.RFC3339)

	if err := tc.DB.Save(&progress).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save progress",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Progress updated",
		"progress": fiber.Map{
			"questions_answered": progress.QuestionsAnswered,
			"correct_answers":    progress.CorrectAnswers,
			"score":              progress.Score,
			"attempts_used":      progress.AttemptsUsed,
			"attempts_left":      accessSettings.AttemptsAllowed - progress.AttemptsUsed,
		},
	})
}

func (tc *TestsController) GetTestAnalytics(c *fiber.Ctx) error {
	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var progresses []models.UserTestProgress
	if err := tc.DB.Where("test_id = ?", testID).Find(&progresses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var users []fiber.Map
	for _, progress := range progresses {
		var user models.User
		if err := tc.DB.First(&user, progress.UserID).Error; err != nil {
			continue
		}

		users = append(users, fiber.Map{
			"user_id":            user.ID,
			"username":           user.Username,
			"questions_answered": progress.QuestionsAnswered,
			"correct_answers":    progress.CorrectAnswers,
			"score":              progress.Score,
			"attempts_used":      progress.AttemptsUsed,
		})
	}

	return c.JSON(fiber.Map{
		"analytics": users,
	})
}

func (tc *TestsController) CreateTest(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var test models.Test
	if err := c.BodyParser(&test); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	test.AuthorID = userID
	test.CompletionRate = 0

	if err := tc.DB.Create(&test).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create test",
		})
	}

	// Create default access settings
	accessSettings := models.TestAccessSettings{
		TestID:          test.ID,
		AccessLevel:     "private",
		Admins:          strconv.Itoa(int(userID)),
		AttemptsAllowed: 1,
	}

	if err := tc.DB.Create(&accessSettings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create access settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Test created",
		"test":    test,
	})
}

func (tc *TestsController) UpdateTestDescription(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var input struct {
		Title          string `json:"title"`
		ShortDesc      string `json:"short_desc"`
		Description    string `json:"description"`
		Difficulty     string `json:"difficulty"`
		RecommendedFor string `json:"recommended_for"`
		University     string `json:"university"`
		Topic          string `json:"topic"`
		LogoURL        string `json:"logo_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var test models.Test
	if err := tc.DB.First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if test.AuthorID != userID && !strings.Contains(test.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to edit this test",
		})
	}

	// Update fields
	if input.Title != "" {
		test.Title = input.Title
	}
	if input.ShortDesc != "" {
		test.ShortDesc = input.ShortDesc
	}
	if input.Description != "" {
		test.Description = input.Description
	}
	if input.Difficulty != "" {
		test.Difficulty = input.Difficulty
	}
	if input.RecommendedFor != "" {
		test.RecommendedFor = input.RecommendedFor
	}
	if input.University != "" {
		test.University = input.University
	}
	if input.Topic != "" {
		test.Topic = input.Topic
	}
	if input.LogoURL != "" {
		test.LogoURL = input.LogoURL
	}

	if err := tc.DB.Save(&test).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update test",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Test updated",
		"test":    test,
	})
}

func (tc *TestsController) AddQuestion(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var input struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		Question      string   `json:"question"`
		Options       []string `json:"options"`
		CorrectAnswer int      `json:"correct_answer"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var test models.Test
	if err := tc.DB.First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if test.AuthorID != userID && !strings.Contains(test.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to add questions to this test",
		})
	}

	// Validate correct answer index
	if input.CorrectAnswer < 0 || input.CorrectAnswer >= len(input.Options) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid correct answer index",
		})
	}

	// Convert options to JSON
	optionsJson, err := json.Marshal(input.Options)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not encode options",
		})
	}

	// Get current question count to set sequence order
	var questionCount int64
	tc.DB.Model(&models.TestQuestion{}).Where("test_id = ?", testID).Count(&questionCount)

	question := models.TestQuestion{
		TestID:        uint(testID),
		Title:         input.Title,
		Description:   input.Description,
		Question:      input.Question,
		Options:       string(optionsJson),
		CorrectAnswer: input.CorrectAnswer,
		SequenceOrder: int(questionCount) + 1,
	}

	if err := tc.DB.Create(&question).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create question",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Question added",
		"question": question,
	})
}

func (tc *TestsController) UpdateQuestion(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	questionID, err := strconv.Atoi(c.Params("questionId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid question ID",
		})
	}

	var input struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		Question      string   `json:"question"`
		Options       []string `json:"options"`
		CorrectAnswer int      `json:"correct_answer"`
		SequenceOrder int      `json:"sequence_order"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var test models.Test
	if err := tc.DB.First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if test.AuthorID != userID && !strings.Contains(test.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to edit questions in this test",
		})
	}

	var question models.TestQuestion
	if err := tc.DB.Where("id = ? AND test_id = ?", questionID, testID).First(&question).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Question not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Update fields
	if input.Title != "" {
		question.Title = input.Title
	}
	if input.Description != "" {
		question.Description = input.Description
	}
	if input.Question != "" {
		question.Question = input.Question
	}
	if input.Options != nil {
		optionsJson, err := json.Marshal(input.Options)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not encode options",
			})
		}
		question.Options = string(optionsJson)
	}
	if input.CorrectAnswer >= 0 {
		question.CorrectAnswer = input.CorrectAnswer
	}
	if input.SequenceOrder != 0 {
		question.SequenceOrder = input.SequenceOrder
	}

	if err := tc.DB.Save(&question).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update question",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Question updated",
		"question": question,
	})
}

func (tc *TestsController) GetTestComments(c *fiber.Ctx) error {
	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var comments []models.TestComment
	if err := tc.DB.Where("test_id = ?", testID).Find(&comments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	return c.JSON(comments)
}

func (tc *TestsController) UpdateTestSettings(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var input struct {
		AccessLevel     string `json:"access_level"`
		StartDate       string `json:"start_date"`
		EndDate         string `json:"end_date"`
		Admins          string `json:"admins"`
		AttemptsAllowed int    `json:"attempts_allowed"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var test models.Test
	if err := tc.DB.Preload("AccessSettings").First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if test.AuthorID != userID && !strings.Contains(test.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to edit settings for this test",
		})
	}

	// Update settings
	if input.AccessLevel != "" {
		test.AccessSettings.AccessLevel = input.AccessLevel
	}
	if input.StartDate != "" {
		test.AccessSettings.StartDate = input.StartDate
	}
	if input.EndDate != "" {
		test.AccessSettings.EndDate = input.EndDate
	}
	if input.Admins != "" {
		test.AccessSettings.Admins = input.Admins
	}
	if input.AttemptsAllowed >= 0 {
		test.AccessSettings.AttemptsAllowed = input.AttemptsAllowed
	}

	if err := tc.DB.Save(&test.AccessSettings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update test settings",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Test settings updated",
		"settings": test.AccessSettings,
	})
}

func (tc *TestsController) GetTestResult(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, tc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	testID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test ID",
		})
	}

	var test models.Test
	if err := tc.DB.Preload("Questions").First(&test, testID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Test not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var progress models.UserTestProgress
	if err := tc.DB.Where("user_id = ? AND test_id = ?", userID, testID).First(&progress).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Test not completed",
		})
	}

	// Prepare questions with correct answers
	var questions []map[string]interface{}
	for _, q := range test.Questions {
		var options []string
		json.Unmarshal([]byte(q.Options), &options)

		questions = append(questions, map[string]interface{}{
			"id":             q.ID,
			"title":          q.Title,
			"description":    q.Description,
			"question":       q.Question,
			"options":        options,
			"correct_answer": q.CorrectAnswer,
			"order":          q.SequenceOrder,
		})
	}

	return c.JSON(fiber.Map{
		"test": fiber.Map{
			"id":        test.ID,
			"title":     test.Title,
			"questions": questions,
		},
		"result": fiber.Map{
			"questions_answered": progress.QuestionsAnswered,
			"correct_answers":    progress.CorrectAnswers,
			"score":              progress.Score,
			"attempts_used":      progress.AttemptsUsed,
		},
	})
}
