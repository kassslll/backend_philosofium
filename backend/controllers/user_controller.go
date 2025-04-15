package controllers

import (
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewUserController(db *gorm.DB, cfg *config.Config) *UserController {
	return &UserController{DB: db, Cfg: cfg}
}

type UpdateUserRequest struct {
	Username    string `json:"username" example:"john_doe" minLength:"3" maxLength:"20"`
	Email       string `json:"email" example:"user@example.com" format:"email"`
	OldPassword string `json:"old_password" example:"oldPassword123" minLength:"8"`
	NewPassword string `json:"new_password" example:"newPassword123" minLength:"8"`
	Group       string `json:"group" example:"students" enums:"students,professors,admins"`
	University  string `json:"university" example:"Stanford University"`
}

// GetProfile godoc
// @Summary Get user profile
// @Description Returns authenticated user's profile data
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /users/profile [get]
func (uc *UserController) GetProfile(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, uc.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	var user models.User
	if err := uc.DB.First(&user, userID).Error; err != nil {
		return utils.NotFound(c, "User not found")
	}

	// Получаем прогресс пользователя
	var progress models.UserProgress
	uc.DB.Where("user_id = ?", userID).First(&progress)

	// Получаем активные курсы
	var activeCourses []models.UserCourseProgress
	uc.DB.Preload("Course").
		Where("user_id = ? AND completion_rate < 100", userID).
		Order("updated_at DESC").
		Limit(3).
		Find(&activeCourses)

	// Формируем ответ без чувствительных данных
	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"role":           user.Role,
		"group":          user.Group,
		"university":     user.University,
		"created_at":     user.CreatedAt,
		"progress":       progress,
		"active_courses": activeCourses,
	})
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Updates authenticated user's profile data
// @Tags users
// @Accept json
// @Produce json
// @Param input body UpdateUserRequest true "Profile update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /users/profile [put]
func (uc *UserController) UpdateProfile(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, uc.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	var input struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
		Group       string `json:"group"`
		University  string `json:"university"`
	}

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Cannot parse JSON")
	}

	var user models.User
	if err := uc.DB.First(&user, userID).Error; err != nil {
		return utils.NotFound(c, "User not found")
	}

	// Обновление имени пользователя
	if input.Username != "" && input.Username != user.Username {
		// Проверяем, не занято ли имя
		var existingUser models.User
		if err := uc.DB.Where("username = ?", input.Username).First(&existingUser).Error; err == nil {
			if existingUser.ID != user.ID {
				return utils.BadRequest(c, "Username already taken")
			}
		}
		user.Username = input.Username
	}

	// Обновление email
	if input.Email != "" && input.Email != user.Email {
		// Проверяем, не занят ли email
		var existingUser models.User
		if err := uc.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
			if existingUser.ID != user.ID {
				return utils.BadRequest(c, "Email already taken")
			}
		}
		user.Email = input.Email
	}

	// Обновление пароля
	if input.NewPassword != "" {
		if input.OldPassword == "" {
			return utils.BadRequest(c, "Old password is required to set new password")
		}

		// Проверяем старый пароль
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword)); err != nil {
			return utils.Unauthorized(c, "Invalid old password")
		}

		// Хешируем новый пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return utils.InternalServerError(c, "Could not hash password")
		}
		user.PasswordHash = string(hashedPassword)
	}

	// Обновление группы и университета
	if input.Group != "" {
		user.Group = input.Group
	}
	if input.University != "" {
		user.University = input.University
	}

	// Сохраняем изменения
	if err := uc.DB.Save(&user).Error; err != nil {
		return utils.InternalServerError(c, "Could not update user")
	}

	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"message": "Profile updated successfully",
	})
}

// GetUserCourses godoc
// @Summary Get user's courses
// @Description Returns paginated list of user's courses with progress
// @Tags users
// @Accept json
// @Produce json
// @Param status query string false "Filter by status (all|in_progress|completed)" default(all)
// @Param search query string false "Search term"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} utils.PaginatedResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /users/courses [get]
func (uc *UserController) GetUserCourses(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, uc.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	status := c.Query("status", "all")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	query := uc.DB.Model(&models.UserCourseProgress{}).Where("user_id = ?", userID)

	switch status {
	case "in_progress":
		query = query.Where("completion_rate < 100")
	case "completed":
		query = query.Where("completion_rate >= 100")
	}

	if search != "" {
		query = query.Joins("JOIN courses ON courses.id = user_course_progress.course_id").
			Where("courses.title ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var progresses []models.UserCourseProgress
	if err := query.Offset(offset).Limit(pageSize).Find(&progresses).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch progress data")
	}

	var courses []map[string]interface{}
	for _, progress := range progresses {
		var course models.Course
		if err := uc.DB.Where("id = ?", progress.CourseID).First(&course).Error; err != nil {
			continue // если курс не найден — пропускаем
		}

		var lessonCount int64
		uc.DB.Model(&models.Lesson{}).Where("course_id = ?", course.ID).Count(&lessonCount)

		courses = append(courses, map[string]interface{}{
			"id":            course.ID,
			"title":         course.Title,
			"short_desc":    course.ShortDesc,
			"logo_url":      course.LogoURL,
			"progress":      progress.CompletionRate,
			"lessons":       lessonCount,
			"completed":     progress.LessonsCompleted,
			"last_accessed": progress.LastAccessed,
		})
	}

	return utils.Paginate(c, courses, total, page, pageSize)
}

// GetUserTests godoc
// @Summary Get user's tests
// @Description Returns paginated list of user's tests with progress
// @Tags users
// @Accept json
// @Produce json
// @Param status query string false "Filter by status (all|in_progress|completed)" default(all)
// @Param search query string false "Search term"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} utils.PaginatedResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /users/tests [get]
func (uc *UserController) GetUserTests(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, uc.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	status := c.Query("status", "all")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	query := uc.DB.Model(&models.UserTestProgress{}).Where("user_id = ?", userID)

	switch status {
	case "in_progress":
		query = query.Where("score IS NULL OR attempts_used = 0")
	case "completed":
		query = query.Where("score IS NOT NULL AND attempts_used > 0")
	}

	if search != "" {
		query = query.Joins("JOIN tests ON tests.id = user_test_progress.test_id").
			Where("tests.title ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var progresses []models.UserTestProgress
	if err := query.Offset(offset).Limit(pageSize).Find(&progresses).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch tests")
	}

	var tests []map[string]interface{}
	for _, progress := range progresses {
		var test models.Test
		if err := uc.DB.Where("id = ?", progress.TestID).First(&test).Error; err != nil {
			continue // если тест не найден — пропускаем
		}

		tests = append(tests, map[string]interface{}{
			"id":            test.ID,
			"title":         test.Title,
			"short_desc":    test.ShortDesc,
			"logo_url":      test.LogoURL,
			"score":         progress.Score,
			"attempts_used": progress.AttemptsUsed,
			"last_attempt":  progress.LastAttempt,
		})
	}

	return utils.Paginate(c, tests, total, page, pageSize)
}

// GetUserActivity godoc
// @Summary Get user activity
// @Description Returns user's recent activity data
// @Tags users
// @Accept json
// @Produce json
// @Param days query int false "Number of days to look back" default(7)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /users/activity [get]
func (uc *UserController) GetUserActivity(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, uc.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	// Параметры периода
	days, _ := strconv.Atoi(c.Query("days", "7")) // По умолчанию за последние 7 дней

	// Получаем историю входов
	var logins []models.LoginHistory
	if err := uc.DB.Where("user_id = ? AND login_time >= ?",
		userID, time.Now().AddDate(0, 0, -days)).
		Order("login_time DESC").
		Find(&logins).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch login history")
	}

	// Получаем активность по курсам
	var courseActivity []struct {
		Date    string  `json:"date"`
		Courses int     `json:"courses"`
		Lessons int     `json:"lessons"`
		Hours   float64 `json:"hours"`
	}

	uc.DB.Raw(`
		SELECT 
			DATE(updated_at) as date,
			COUNT(DISTINCT course_id) as courses,
			SUM(lessons_completed) as lessons,
			SUM(hours_spent) as hours
		FROM user_course_progress
		WHERE user_id = ? AND updated_at >= ?
		GROUP BY DATE(updated_at)
		ORDER BY date DESC
	`, userID, time.Now().AddDate(0, 0, -days)).Scan(&courseActivity)

	// Получаем активность по тестам
	var testActivity []struct {
		Date     string  `json:"date"`
		Tests    int     `json:"tests"`
		Attempts int     `json:"attempts"`
		AvgScore float64 `json:"avg_score"`
	}

	uc.DB.Raw(`
		SELECT 
			DATE(updated_at) as date,
			COUNT(DISTINCT test_id) as tests,
			SUM(attempts_used) as attempts,
			AVG(score) as avg_score
		FROM user_test_progress
		WHERE user_id = ? AND updated_at >= ?
		GROUP BY DATE(updated_at)
		ORDER BY date DESC
	`, userID, time.Now().AddDate(0, 0, -days)).Scan(&testActivity)

	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"logins":          logins,
		"course_activity": courseActivity,
		"test_activity":   testActivity,
		"period_days":     days,
	})
}
