package controllers

import (
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

type CoursesController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

type ProgressInput struct {
	LessonID      uint    `json:"lesson_id"`
	HoursSpent    float64 `json:"hours_spent"`
	MarkCompleted bool    `json:"mark_completed"`
}

type UpdateCourseRequest struct {
	Title          string `json:"title"`
	ShortDesc      string `json:"short_desc"`
	Description    string `json:"description"`
	Difficulty     string `json:"difficulty"`
	RecommendedFor string `json:"recommended_for"`
	University     string `json:"university"`
	Topic          string `json:"topic"`
	LogoURL        string `json:"logo_url"`
}

type CreateLessonRequest struct {
	Title       string `json:"title" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"required,max=500"`
	Content     string `json:"content" validate:"required"`
}

type UpdateLessonRequest struct {
	Title         string `json:"title" example:"Introduction to Philosophy"`
	Description   string `json:"description" example:"Basic concepts"`
	Content       string `json:"content" example:"<p>Lesson content</p>"`
	SequenceOrder int    `json:"sequence_order" example:"1"`
}

type CourseAccessRequest struct {
	AccessLevel string   `json:"access_level" validate:"required,oneof=public private restricted"`
	StartDate   string   `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate     string   `json:"end_date" validate:"required,datetime=2006-01-02,gtfield=StartDate"`
	Admins      []string `json:"admins" validate:"dive,email"`
}

func NewCoursesController(db *gorm.DB, cfg *config.Config) *CoursesController {
	return &CoursesController{DB: db, Cfg: cfg}
}

// GetUserCourses godoc
// @Summary Get user's enrolled courses
// @Description Returns all courses the user is enrolled in
// @Tags courses
// @Accept json
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/my [get]
func (cc *CoursesController) GetUserCourses(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var courses []models.Course
	cc.DB.Joins("JOIN user_course_progress ON user_course_progress.course_id = courses.id").
		Where("user_course_progress.user_id = ?", userID).
		Find(&courses)

	var result []fiber.Map
	for _, course := range courses {
		var progress models.UserCourseProgress
		cc.DB.Where("user_id = ? AND course_id = ?", userID, course.ID).First(&progress)

		result = append(result, fiber.Map{
			"id":            course.ID,
			"title":         course.Title,
			"progress":      progress.CompletionRate,
			"group":         course.RecommendedFor,
			"lessons":       len(course.Lessons),
			"completed":     progress.LessonsCompleted,
			"hours_spent":   progress.HoursSpent,
			"last_accessed": progress.LastAccessed,
		})
	}

	return c.JSON(result)
}

// GetAvailableCourses godoc
// @Summary Get available courses
// @Description Returns all public courses available to the user
// @Tags courses
// @Accept json
// @Produce json
// @Param topic query string false "Filter by topic"
// @Param university query string false "Filter by university"
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/available [get]
func (cc *CoursesController) GetAvailableCourses(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get query parameters
	topic := c.Query("topic")
	university := c.Query("university")

	query := cc.DB.Model(&models.Course{}).Where("access_level = 'public'")

	if topic != "" {
		query = query.Where("topic LIKE ?", "%"+topic+"%")
	}

	if university != "" {
		query = query.Where("university LIKE ?", "%"+university+"%")
	}

	var courses []models.Course
	query.Find(&courses)

	var result []fiber.Map
	for _, course := range courses {
		var progress models.UserCourseProgress
		cc.DB.Where("user_id = ? AND course_id = ?", userID, course.ID).First(&progress)

		result = append(result, fiber.Map{
			"id":          course.ID,
			"title":       course.Title,
			"progress":    progress.CompletionRate,
			"group":       course.RecommendedFor,
			"description": course.ShortDesc,
			"difficulty":  course.Difficulty,
			"university":  course.University,
			"topic":       course.Topic,
			"author":      course.AuthorID,
			"logo_url":    course.LogoURL,
		})
	}

	return c.JSON(result)
}

// GetCourseDetails godoc
// @Summary Get course details
// @Description Returns detailed information about a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id} [get]
func (cc *CoursesController) GetCourseDetails(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var course models.Course
	if err := cc.DB.Preload("Lessons").Preload("Comments").First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Course not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var progress models.UserCourseProgress
	cc.DB.Where("user_id = ? AND course_id = ?", userID, courseID).First(&progress)

	return c.JSON(fiber.Map{
		"course": fiber.Map{
			"id":              course.ID,
			"title":           course.Title,
			"description":     course.Description,
			"short_desc":      course.ShortDesc,
			"difficulty":      course.Difficulty,
			"recommended":     course.RecommendedFor,
			"university":      course.University,
			"topic":           course.Topic,
			"logo_url":        course.LogoURL,
			"author":          course.AuthorID,
			"lessons":         course.Lessons,
			"comments":        course.Comments,
			"completion_rate": course.CompletionRate,
		},
		"progress": progress,
	})
}

// UpdateCourseProgress godoc
// @Summary Update course progress
// @Description Updates user's progress in a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param input body ProgressInput true "Progress data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id}/progress [post]
func (cc *CoursesController) UpdateCourseProgress(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var input ProgressInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var course models.Course
	if err := cc.DB.Preload("Lessons").First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Course not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var progress models.UserCourseProgress
	if err := cc.DB.Where("user_id = ? AND course_id = ?", userID, courseID).First(&progress).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			progress = models.UserCourseProgress{
				UserID:           userID,
				CourseID:         uint(courseID),
				LessonsCompleted: 0,
				HoursSpent:       0,
				CompletionRate:   0,
			}
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not query database",
			})
		}
	}

	if input.MarkCompleted {
		progress.LessonsCompleted++
	}

	progress.HoursSpent += input.HoursSpent
	progress.CompletionRate = float64(progress.LessonsCompleted) / float64(len(course.Lessons)) * 100
	progress.LastAccessed = time.Now().Format(time.RFC3339)

	if err := cc.DB.Save(&progress).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save progress",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Progress updated",
		"progress": progress,
	})
}

// GetCourseAnalytics godoc
// @Summary Get course analytics
// @Description Returns analytics for a course (author/admin only)
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id}/analytics [get]
func (cc *CoursesController) GetCourseAnalytics(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var progresses []models.UserCourseProgress
	if err := cc.DB.Where("course_id = ?", courseID).Find(&progresses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	var users []fiber.Map
	for _, progress := range progresses {
		var user models.User
		if err := cc.DB.First(&user, progress.UserID).Error; err != nil {
			continue
		}

		users = append(users, fiber.Map{
			"user_id":           user.ID,
			"username":          user.Username,
			"lessons_completed": progress.LessonsCompleted,
			"hours_spent":       progress.HoursSpent,
			"completion_rate":   progress.CompletionRate,
		})
	}

	return c.JSON(fiber.Map{
		"analytics": users,
	})
}

// CreateCourse godoc
// @Summary Create a new course
// @Description Creates a new course (author/admin only)
// @Tags courses
// @Accept json
// @Produce json
// @Param course body models.Course true "Course data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses [post]
func (cc *CoursesController) CreateCourse(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var course models.Course
	if err := c.BodyParser(&course); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	course.AuthorID = userID
	course.CompletionRate = 0

	if err := cc.DB.Create(&course).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create course",
		})
	}

	// Create default access settings
	accessSettings := models.CourseAccessSettings{
		CourseID:    course.ID,
		AccessLevel: "private",
		Admins:      strconv.Itoa(int(userID)),
	}

	if err := cc.DB.Create(&accessSettings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create access settings",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Course created",
		"course":  course,
	})
}

// UpdateCourseDescription godoc
// @Summary Update course description
// @Description Updates course metadata (author/admin only)
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param input body UpdateCourseRequest true "Course update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id} [put]
func (cc *CoursesController) UpdateCourseDescription(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
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

	var course models.Course
	if err := cc.DB.First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Course not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if course.AuthorID != userID && !strings.Contains(course.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to edit this course",
		})
	}

	// Update fields
	if input.Title != "" {
		course.Title = input.Title
	}
	if input.ShortDesc != "" {
		course.ShortDesc = input.ShortDesc
	}
	if input.Description != "" {
		course.Description = input.Description
	}
	if input.Difficulty != "" {
		course.Difficulty = input.Difficulty
	}
	if input.RecommendedFor != "" {
		course.RecommendedFor = input.RecommendedFor
	}
	if input.University != "" {
		course.University = input.University
	}
	if input.Topic != "" {
		course.Topic = input.Topic
	}
	if input.LogoURL != "" {
		course.LogoURL = input.LogoURL
	}

	if err := cc.DB.Save(&course).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update course",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Course updated",
		"course":  course,
	})
}

// AddLesson godoc
// @Summary Add lesson to course
// @Description Adds a new lesson to a course (author/admin only)
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param input body CreateLessonRequest true "Lesson data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id}/lessons [post]
func (cc *CoursesController) AddLesson(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var input CreateLessonRequest

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var course models.Course
	if err := cc.DB.First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Course not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if course.AuthorID != userID && !strings.Contains(course.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to add lessons to this course",
		})
	}

	// Get current lesson count to set sequence order
	var lessonCount int64
	cc.DB.Model(&models.Lesson{}).Where("course_id = ?", courseID).Count(&lessonCount)

	lesson := models.Lesson{
		CourseID:      uint(courseID),
		Title:         input.Title,
		Description:   input.Description,
		Content:       input.Content,
		SequenceOrder: int(lessonCount) + 1,
	}

	if err := cc.DB.Create(&lesson).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create lesson",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Lesson added",
		"lesson":  lesson,
	})
}

// UpdateLesson godoc
// @Summary Update lesson
// @Description Updates lesson content (author/admin only)
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param lessonId path int true "Lesson ID"
// @Param input body UpdateLessonRequest true "Lesson update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id}/lessons/{lessonId} [put]
func (cc *CoursesController) UpdateLesson(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	lessonID, err := strconv.Atoi(c.Params("lessonId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid lesson ID",
		})
	}

	var input struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		Content       string `json:"content"`
		SequenceOrder int    `json:"sequence_order"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var course models.Course
	if err := cc.DB.First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Course not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if course.AuthorID != userID && !strings.Contains(course.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to edit lessons in this course",
		})
	}

	var lesson models.Lesson
	if err := cc.DB.Where("id = ? AND course_id = ?", lessonID, courseID).First(&lesson).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Lesson not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Update fields
	if input.Title != "" {
		lesson.Title = input.Title
	}
	if input.Description != "" {
		lesson.Description = input.Description
	}
	if input.Content != "" {
		lesson.Content = input.Content
	}
	if input.SequenceOrder != 0 {
		lesson.SequenceOrder = input.SequenceOrder
	}

	if err := cc.DB.Save(&lesson).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update lesson",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Lesson updated",
		"lesson":  lesson,
	})
}

// GetCourseComments godoc
// @Summary Get course comments
// @Description Returns all comments for a course
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Success 200 {array} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /courses/{id}/comments [get]
func (cc *CoursesController) GetCourseComments(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var comments []models.CourseComment
	if err := cc.DB.Where("course_id = ?", courseID).Find(&comments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	return c.JSON(comments)
}

// UpdateCourseSettings godoc
// @Summary Update course settings
// @Description Updates course access settings (author/admin only)
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "Course ID"
// @Param input body CourseAccessRequest true "Settings data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /courses/{id}/settings [put]
func (cc *CoursesController) UpdateCourseSettings(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, cc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var input struct {
		AccessLevel string `json:"access_level"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		Admins      string `json:"admins"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var course models.Course
	if err := cc.DB.Preload("AccessSettings").First(&course, courseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Course not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query database",
		})
	}

	// Check if user is author or admin
	if course.AuthorID != userID && !strings.Contains(course.AccessSettings.Admins, strconv.Itoa(int(userID))) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to edit settings for this course",
		})
	}

	// Update settings
	if input.AccessLevel != "" {
		course.AccessSettings.AccessLevel = input.AccessLevel
	}
	if input.StartDate != "" {
		course.AccessSettings.StartDate = input.StartDate
	}
	if input.EndDate != "" {
		course.AccessSettings.EndDate = input.EndDate
	}
	if input.Admins != "" {
		course.AccessSettings.Admins = input.Admins
	}

	if err := cc.DB.Save(&course.AccessSettings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update course settings",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Course settings updated",
		"settings": course.AccessSettings,
	})
}
