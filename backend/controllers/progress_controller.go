package controllers

import (
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ProgressController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewProgressController(db *gorm.DB, cfg *config.Config) *ProgressController {
	return &ProgressController{DB: db, Cfg: cfg}
}

// GetProgress godoc
// @Summary Get user progress
// @Description Returns user's progress data for last 4 months
// @Tags progress
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /progress [get]
func (pc *ProgressController) GetProgress(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, pc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get last 4 months progress
	now := time.Now()
	months := make([]models.MonthlyProgress, 4)

	for i := 0; i < 4; i++ {
		month := now.AddDate(0, -i, 0)
		startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		endOfMonth := startOfMonth.AddDate(0, 1, -1)

		var streakDays int
		var coursesCompleted int64
		loginFrequency := make(map[string]int)

		// Get streak days for the month
		pc.DB.Model(&models.UserProgress{}).
			Where("user_id = ? AND last_active BETWEEN ? AND ?", userID, startOfMonth, endOfMonth).
			Select("MAX(streak_days)").
			Scan(&streakDays)

		// Get courses completed in the month
		pc.DB.Model(&models.UserCourseProgress{}).
			Where("user_id = ? AND updated_at BETWEEN ? AND ? AND completion_rate = 100", userID, startOfMonth, endOfMonth).
			Count(&coursesCompleted)

		// Get login frequency (simplified - count logins per day)
		var logins []models.LoginHistory
		pc.DB.Where("user_id = ? AND login_time BETWEEN ? AND ?", userID, startOfMonth, endOfMonth).
			Find(&logins)

		for _, login := range logins {
			day := login.LoginTime.Format("2006-01-02")
			loginFrequency[day]++
		}

		months[i] = models.MonthlyProgress{
			Month:            month.Month(),
			Year:             month.Year(),
			StreakDays:       streakDays,
			CoursesCompleted: coursesCompleted,
			LoginFrequency:   loginFrequency,
		}
	}

	return c.JSON(fiber.Map{
		"progress": months,
	})
}

// GetProgressOverview godoc
// @Summary Get progress overview
// @Description Returns summary of user's progress
// @Tags progress
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Security ApiKeyAuth
// @Router /progress/overview [get]
func (pc *ProgressController) GetProgressOverview(c *fiber.Ctx) error {
	userID, err := utils.ExtractUserIDFromToken(c, pc.Cfg)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var userProgress models.UserProgress
	pc.DB.Where("user_id = ?", userID).First(&userProgress)

	var totalCoursesCompleted int64
	pc.DB.Model(&models.UserCourseProgress{}).
		Where("user_id = ? AND completion_rate = 100", userID).
		Count(&totalCoursesCompleted)

	var totalTestsCompleted int64
	pc.DB.Model(&models.UserTestProgress{}).
		Where("user_id = ? AND attempts_used > 0", userID).
		Count(&totalTestsCompleted)

	return c.JSON(models.ProgressOverview{
		TotalStreakDays:       userProgress.StreakDays,
		TotalCoursesCompleted: int(totalCoursesCompleted),
		TotalTestsCompleted:   int(totalTestsCompleted),
	})
}
