package controllers

import (
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type OverviewController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewOverviewController(db *gorm.DB, cfg *config.Config) *OverviewController {
	return &OverviewController{DB: db, Cfg: cfg}
}

// SearchCourses возвращает курсы по критериям поиска
func (oc *OverviewController) SearchCourses(c *fiber.Ctx) error {
	search := c.Query("search")
	group := c.Query("group")
	sort := c.Query("sort", "popularity") // popularity, newest, rating

	query := oc.DB.Model(&models.Course{}).Where("access_level = 'public'")

	// Поиск по названию/описанию
	if search != "" {
		query = query.Where("title ILIKE ? OR short_desc ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Фильтр по группе
	if group != "" {
		query = query.Where("recommended_for = ?", group)
	}

	// Сортировка
	switch sort {
	case "newest":
		query = query.Order("created_at DESC")
	case "rating":
		query = query.Order("(SELECT AVG(rating) FROM course_comments WHERE course_id = courses.id) DESC")
	default: // popularity
		query = query.Order("(SELECT COUNT(*) FROM user_course_progress WHERE course_id = courses.id) DESC")
	}

	var courses []models.Course
	if err := query.Find(&courses).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch courses")
	}

	// Формируем упрощенный ответ
	var result []map[string]interface{}
	for _, course := range courses {
		// Получаем средний рейтинг
		var avgRating float64
		oc.DB.Model(&models.CourseComment{}).
			Select("COALESCE(AVG(rating), 0)").
			Where("course_id = ?", course.ID).
			Scan(&avgRating)

		// Получаем количество участников
		var enrollments int64
		oc.DB.Model(&models.UserCourseProgress{}).
			Where("course_id = ?", course.ID).
			Count(&enrollments)

		result = append(result, map[string]interface{}{
			"id":          course.ID,
			"title":       course.Title,
			"short_desc":  course.ShortDesc,
			"difficulty":  course.Difficulty,
			"recommended": course.RecommendedFor,
			"university":  course.University,
			"topic":       course.Topic,
			"logo_url":    course.LogoURL,
			"rating":      avgRating,
			"enrollments": enrollments,
			"created_at":  course.CreatedAt,
		})
	}

	return utils.Success(c, fiber.StatusOK, result)
}

// GetUserOverview возвращает обзорную информацию для пользователя
func (oc *OverviewController) GetUserOverview(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, oc.Cfg)
	if err != nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	// Получаем прогресс пользователя
	var progress models.UserProgress
	if err := oc.DB.Where("user_id = ?", userID).First(&progress).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch user progress")
	}

	// Получаем активные курсы
	var activeCourses []models.UserCourseProgress
	if err := oc.DB.Preload("Course").
		Where("user_id = ? AND completion_rate < 100", userID).
		Order("updated_at DESC").
		Limit(3).
		Find(&activeCourses).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch active courses")
	}

	// Получаем рекомендации курсов
	recommendedCourses, err := oc.getRecommendedCourses(userID)
	if err != nil {
		return utils.InternalServerError(c, "Failed to get recommendations")
	}

	// Формируем ответ
	return utils.Success(c, fiber.StatusOK, fiber.Map{
		"streak_days":       progress.StreakDays,
		"courses_completed": progress.CoursesCompleted,
		"tests_completed":   progress.TestsCompleted,
		"active_courses":    activeCourses,
		"recommendations":   recommendedCourses,
	})
}

// getRecommendedCourses возвращает рекомендованные курсы для пользователя
func (oc *OverviewController) getRecommendedCourses(userID uint) ([]map[string]interface{}, error) {
	var recommendations []map[string]interface{}

	// Простая реализация рекомендаций (можно улучшить)
	// 1. По группе пользователя
	var user models.User
	if err := oc.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	query := oc.DB.Model(&models.Course{}).
		Where("access_level = 'public' AND recommended_for = ?", user.Group).
		Order("(SELECT COUNT(*) FROM user_course_progress WHERE course_id = courses.id) DESC").
		Limit(3)

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var course models.Course
		oc.DB.ScanRows(rows, &course)

		recommendations = append(recommendations, map[string]interface{}{
			"id":         course.ID,
			"title":      course.Title,
			"short_desc": course.ShortDesc,
			"reason":     "Recommended for your group",
		})
	}

	// 2. По университету, если не хватило рекомендаций
	if len(recommendations) < 3 && user.University != "" {
		query = oc.DB.Model(&models.Course{}).
			Where("access_level = 'public' AND university = ?", user.University).
			Order("created_at DESC").
			Limit(3 - len(recommendations))

		rows, err = query.Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var course models.Course
			oc.DB.ScanRows(rows, &course)

			recommendations = append(recommendations, map[string]interface{}{
				"id":         course.ID,
				"title":      course.Title,
				"short_desc": course.ShortDesc,
				"reason":     "Popular in your university",
			})
		}
	}

	return recommendations, nil
}

// SearchTests возвращает тесты по критериям поиска
func (oc *OverviewController) SearchTests(c *fiber.Ctx) error {
	search := c.Query("search")
	group := c.Query("group")
	sort := c.Query("sort", "popularity") // popularity, newest, rating

	query := oc.DB.Model(&models.Test{}).Where("access_level = 'public'")

	// Поиск по названию/описанию
	if search != "" {
		query = query.Where("title ILIKE ? OR short_desc ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Фильтр по группе
	if group != "" {
		query = query.Where("recommended_for = ?", group)
	}

	// Сортировка
	switch sort {
	case "newest":
		query = query.Order("created_at DESC")
	case "rating":
		query = query.Order("(SELECT AVG(rating) FROM test_comments WHERE test_id = tests.id) DESC")
	default: // popularity
		query = query.Order("(SELECT COUNT(*) FROM user_test_progress WHERE test_id = tests.id) DESC")
	}

	var tests []models.Test
	if err := query.Find(&tests).Error; err != nil {
		return utils.InternalServerError(c, "Failed to fetch tests")
	}

	// Формируем упрощенный ответ
	var result []map[string]interface{}
	for _, test := range tests {
		// Получаем средний рейтинг
		var avgRating float64
		oc.DB.Model(&models.TestComment{}).
			Select("COALESCE(AVG(rating), 0)").
			Where("test_id = ?", test.ID).
			Scan(&avgRating)

		// Получаем количество участников
		var attempts int64
		oc.DB.Model(&models.UserTestProgress{}).
			Where("test_id = ?", test.ID).
			Count(&attempts)

		result = append(result, map[string]interface{}{
			"id":          test.ID,
			"title":       test.Title,
			"short_desc":  test.ShortDesc,
			"difficulty":  test.Difficulty,
			"recommended": test.RecommendedFor,
			"university":  test.University,
			"topic":       test.Topic,
			"logo_url":    test.LogoURL,
			"rating":      avgRating,
			"attempts":    attempts,
			"created_at":  test.CreatedAt,
		})
	}

	return utils.Success(c, fiber.StatusOK, result)
}
