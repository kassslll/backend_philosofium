package controllers

import (
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AnalyticsController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAnalyticsController(db *gorm.DB, cfg *config.Config) *AnalyticsController {
	return &AnalyticsController{DB: db, Cfg: cfg}
}

// GetUserProgressAnalytics возвращает аналитику прогресса пользователя
func (ac *AnalyticsController) GetUserProgressAnalytics(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, ac.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	// Получаем параметры периода
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Парсим даты или устанавливаем значения по умолчанию
	var start, end time.Time
	if startDate == "" {
		start = time.Now().AddDate(0, -1, 0) // Последний месяц по умолчанию
	} else {
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return utils.BadRequest(c, "Invalid start_date format. Use YYYY-MM-DD")
		}
	}

	if endDate == "" {
		end = time.Now()
	} else {
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return utils.BadRequest(c, "Invalid end_date format. Use YYYY-MM-DD")
		}
	}

	// Получаем данные о прогрессе курсов
	var courseProgress []models.UserCourseProgress
	if err := ac.DB.Where("user_id = ? AND updated_at BETWEEN ? AND ?",
		userID, start, end).Find(&courseProgress).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch course progress")
	}

	// Получаем данные о прогрессе тестов
	var testProgress []models.UserTestProgress
	if err := ac.DB.Where("user_id = ? AND updated_at BETWEEN ? AND ?",
		userID, start, end).Find(&testProgress).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch test progress")
	}

	// Получаем данные о посещениях
	var loginHistory []models.LoginHistory
	if err := ac.DB.Where("user_id = ? AND login_time BETWEEN ? AND ?",
		userID, start, end).Find(&loginHistory).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch login history")
	}

	// Формируем ответ
	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"course_progress": courseProgress,
		"test_progress":   testProgress,
		"login_history":   loginHistory,
		"period": fiber.Map{
			"start_date": start.Format("2006-01-02"),
			"end_date":   end.Format("2006-01-02"),
		},
	})
}

// GetCourseAnalytics возвращает аналитику по курсу
func (ac *AnalyticsController) GetCourseAnalytics(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "Invalid course ID")
	}

	// Проверяем права доступа (только для автора/админа)
	userID, err := utils.ExtractUserIDFromToken(c, ac.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	var course models.Course
	if err := ac.DB.First(&course, courseID).Error; err != nil {
		return utils.NotFound(c, "Course not found")
	}

	if course.AuthorID != userID {
		return utils.Forbidden(c, "You don't have permission to view this analytics")
	}

	// Получаем статистику по курсу
	var stats struct {
		TotalEnrollments  int64
		Completed         int64
		AvgCompletionRate float64
		AvgTimeSpent      float64
	}

	ac.DB.Model(&models.UserCourseProgress{}).
		Where("course_id = ?", courseID).
		Count(&stats.TotalEnrollments)

	ac.DB.Model(&models.UserCourseProgress{}).
		Where("course_id = ? AND completion_rate >= 100", courseID).
		Count(&stats.Completed)

	ac.DB.Model(&models.UserCourseProgress{}).
		Select("AVG(completion_rate)").
		Where("course_id = ?", courseID).
		Scan(&stats.AvgCompletionRate)

	ac.DB.Model(&models.UserCourseProgress{}).
		Select("AVG(hours_spent)").
		Where("course_id = ?", courseID).
		Scan(&stats.AvgTimeSpent)

	// Получаем прогресс по урокам
	var lessonCompletion []struct {
		LessonID    uint   `json:"lesson_id"`
		LessonTitle string `json:"lesson_title"`
		Completed   int64  `json:"completed"`
		Total       int64  `json:"total"`
	}

	ac.DB.Raw(`
		SELECT l.id as lesson_id, l.title as lesson_title, 
		COUNT(ucp.id) as completed,
		(SELECT COUNT(*) FROM user_course_progress WHERE course_id = ?) as total
		FROM lessons l
		LEFT JOIN user_course_progress ucp ON ucp.lessons_completed >= l.sequence_order AND ucp.course_id = l.course_id
		WHERE l.course_id = ?
		GROUP BY l.id, l.title
	`, courseID, courseID).Scan(&lessonCompletion)

	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"course_id":    courseID,
		"course_title": course.Title,
		"stats":        stats,
		"lesson_stats": lessonCompletion,
		"enrollments":  getEnrollmentTrends(ac.DB, uint(courseID)),
	})
}

// getEnrollmentTrends возвращает динамику регистраций на курс
func getEnrollmentTrends(db *gorm.DB, courseID uint) []map[string]interface{} {
	var trends []map[string]interface{}

	db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as enrollments
		FROM user_course_progress
		WHERE course_id = ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`, courseID).Scan(&trends)

	return trends
}

// GetPlatformAnalytics возвращает аналитику по всей платформе (только для админов)
func (ac *AnalyticsController) GetPlatformAnalytics(c *fiber.Ctx) error {
	// Проверка прав администратора
	userID, err := utils.ExtractUserIDFromToken(c, ac.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	var user models.User
	if err := ac.DB.First(&user, userID).Error; err != nil {
		return utils.NotFound(c, "User not found")
	}

	if user.Role != "admin" {
		return utils.Forbidden(c, "Admin access required")
	}

	// Основные метрики платформы
	var metrics struct {
		TotalUsers        int64   `json:"total_users"`
		ActiveUsers       int64   `json:"active_users"`
		NewUsers          int64   `json:"new_users"`
		TotalCourses      int64   `json:"total_courses"`
		ActiveCourses     int64   `json:"active_courses"`
		TotalTests        int64   `json:"total_tests"`
		AvgCourseProgress float64 `json:"avg_course_progress"`
	}

	// Получаем данные
	ac.DB.Model(&models.User{}).Count(&metrics.TotalUsers)
	ac.DB.Model(&models.User{}).Where("last_login > ?",
		time.Now().AddDate(0, 0, -30)).Count(&metrics.ActiveUsers)
	ac.DB.Model(&models.User{}).Where("created_at > ?",
		time.Now().AddDate(0, 0, -7)).Count(&metrics.NewUsers)
	ac.DB.Model(&models.Course{}).Count(&metrics.TotalCourses)
	ac.DB.Model(&models.Course{}).Where("updated_at > ?",
		time.Now().AddDate(0, -1, 0)).Count(&metrics.ActiveCourses)
	ac.DB.Model(&models.Test{}).Count(&metrics.TotalTests)
	ac.DB.Model(&models.UserCourseProgress{}).
		Select("AVG(completion_rate)").Scan(&metrics.AvgCourseProgress)

	// Динамика регистраций пользователей
	var userGrowth []map[string]interface{}
	ac.DB.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as users
		FROM users
		GROUP BY DATE(created_at)
		ORDER BY date
	`).Scan(&userGrowth)

	// Самые популярные курсы
	var popularCourses []map[string]interface{}
	ac.DB.Raw(`
		SELECT 
			c.id,
			c.title,
			COUNT(ucp.id) as enrollments,
			AVG(ucp.completion_rate) as avg_completion
		FROM courses c
		LEFT JOIN user_course_progress ucp ON ucp.course_id = c.id
		GROUP BY c.id, c.title
		ORDER BY enrollments DESC
		LIMIT 5
	`).Scan(&popularCourses)

	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"metrics":         metrics,
		"user_growth":     userGrowth,
		"popular_courses": popularCourses,
		"timestamp":       time.Now().Format(time.RFC3339),
	})
}
