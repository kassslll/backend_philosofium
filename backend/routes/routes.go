package routes

import (
	"project/backend/config"
	"project/backend/controllers"
	"project/backend/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *config.Config) {
	// Auth routes
	authController := controllers.NewAuthController(db, cfg)
	app.Post("/api/auth/register", authController.Register)
	app.Post("/api/auth/login", authController.Login)

	// Middleware
	authMiddleware := middleware.AuthMiddleware(cfg)
	adminMiddleware := middleware.AdminMiddleware(cfg)

	// Progress routes
	progressController := controllers.NewProgressController(db, cfg)
	app.Get("/api/progress", authMiddleware, progressController.GetProgress)
	app.Get("/api/progress/overview", authMiddleware, progressController.GetProgressOverview)

	// Courses routes
	coursesController := controllers.NewCoursesController(db, cfg)
	courses := app.Group("/api/courses", authMiddleware)
	courses.Get("/", coursesController.GetUserCourses)
	courses.Get("/available", coursesController.GetAvailableCourses)
	courses.Get("/:id", coursesController.GetCourseDetails)
	courses.Post("/:id/progress", coursesController.UpdateCourseProgress)
	courses.Get("/:id/analytics", adminMiddleware, coursesController.GetCourseAnalytics)

	// Tests routes
	testsController := controllers.NewTestsController(db, cfg)
	tests := app.Group("/api/tests", authMiddleware)
	tests.Get("/", testsController.GetUserTests)
	tests.Get("/available", testsController.GetAvailableTests)
	tests.Get("/:id", testsController.GetTestDetails)
	tests.Post("/:id/progress", testsController.UpdateTestProgress)
	tests.Get("/:id/analytics", adminMiddleware, testsController.GetTestAnalytics)
	tests.Get("/:id/result", testsController.GetTestResult)

	// Admin routes for courses
	adminCourses := app.Group("/api/admin/courses", authMiddleware, adminMiddleware)
	adminCourses.Post("/", coursesController.CreateCourse)
	adminCourses.Put("/:id/description", coursesController.UpdateCourseDescription)
	adminCourses.Post("/:id/lessons", coursesController.AddLesson)
	adminCourses.Put("/:id/lessons/:lessonId", coursesController.UpdateLesson)
	adminCourses.Get("/:id/comments", coursesController.GetCourseComments)
	adminCourses.Put("/:id/settings", coursesController.UpdateCourseSettings)

	// Admin routes for tests
	adminTests := app.Group("/api/admin/tests", authMiddleware, adminMiddleware)
	adminTests.Post("/", testsController.CreateTest)
	adminTests.Put("/:id/description", testsController.UpdateTestDescription)
	adminTests.Post("/:id/questions", testsController.AddQuestion)
	adminTests.Put("/:id/questions/:questionId", testsController.UpdateQuestion)
	adminTests.Get("/:id/comments", testsController.GetTestComments)
	adminTests.Put("/:id/settings", testsController.UpdateTestSettings)

	// Comments routes
	commentsController := controllers.NewCommentsController(db, cfg)
	comments := app.Group("/api/comments", middleware.AuthMiddleware(cfg))
	comments.Post("/course/:id", commentsController.AddCourseComment)
	comments.Get("/course/:id", commentsController.GetCourseComments)

	// User routes
	userController := controllers.NewUserController(db, cfg)
	user := app.Group("/api/user", middleware.AuthMiddleware(cfg))
	user.Get("/profile", userController.GetProfile)
	user.Put("/profile", userController.UpdateProfile)
	user.Get("/courses", userController.GetUserCourses)
	user.Get("/tests", userController.GetUserTests)
	user.Get("/activity", userController.GetUserActivity)

	// Analytics routes
	analyticsController := controllers.NewAnalyticsController(db, cfg)
	analytics := app.Group("/api/analytics", middleware.AuthMiddleware(cfg))
	analytics.Get("/progress", analyticsController.GetUserProgressAnalytics)
	analytics.Get("/course/:id", analyticsController.GetCourseAnalytics)
	analytics.Get("/test/:id", analyticsController.GetTestAnalytics)
	analytics.Get("/platform", analyticsController.GetPlatformAnalytics)

	// Overview routes
	overviewController := controllers.NewOverviewController(db, cfg)
	overview := app.Group("/api/overview", middleware.AuthMiddleware(cfg))
	overview.Get("/", overviewController.GetUserOverview)
	overview.Get("/courses", overviewController.SearchCourses)
	overview.Get("/tests", overviewController.SearchTests)
}
