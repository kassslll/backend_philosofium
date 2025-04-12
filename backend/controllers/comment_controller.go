package controllers

import (
	"project/backend/config"
	"project/backend/models"
	"project/backend/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CommentsController struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewCommentsController(db *gorm.DB, cfg *config.Config) *CommentsController {
	return &CommentsController{DB: db, Cfg: cfg}
}

func (cc *CommentsController) AddCourseComment(c *fiber.Ctx) error {
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
		Text   string `json:"text"`
		Rating int    `json:"rating"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Validate rating
	if input.Rating < 0 || input.Rating > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rating must be between 0 and 5",
		})
	}

	// Get user info
	var user models.User
	if err := cc.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	comment := models.CourseComment{
		CourseID:  uint(courseID),
		UserID:    userID,
		UserName:  user.Username,
		UserImage: "", // You can add user image URL here
		Text:      input.Text,
		Rating:    input.Rating,
	}

	if err := cc.DB.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create comment",
		})
	}

	return c.JSON(comment)
}

func (cc *CommentsController) GetCourseComments(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	var comments []models.CourseComment
	result := cc.DB.Preload("Replies").Where("course_id = ?", courseID).Find(&comments)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch comments",
		})
	}

	return c.JSON(comments)
}
