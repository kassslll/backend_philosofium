package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"project/backend/config"
	"project/backend/controllers"
	"project/backend/models"
	"project/backend/routes"
	"project/backend/utils"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var (
	app      *fiber.App
	db       *gorm.DB
	cfg      *config.Config
	authCtrl *controllers.AuthController
	testUser models.User
	jwtToken string
)

func TestMain(m *testing.M) {
	// Setup
	setup()
	// Run tests
	code := m.Run()
	// Cleanup
	teardown()
	os.Exit(code)
}

func setup() {
	// Load test configuration
	cfg = &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBName:     "learning_platform_test",
		JWTSecret:  "testsecret",
		ServerPort: "8080",
	}

	// Initialize database
	var err error
	db, err = utils.InitDB(cfg)
	if err != nil {
		panic(err)
	}

	// Migrate test database
	db.AutoMigrate(
		&models.User{},
		&models.UserProgress{},
		&models.LoginHistory{},
		&models.Course{},
		&models.Lesson{},
		&models.CourseComment{},
		&models.CourseAccessSettings{},
		&models.UserCourseProgress{},
		&models.Test{},
		&models.TestQuestion{},
		&models.TestComment{},
		&models.TestAccessSettings{},
		&models.UserTestProgress{},
	)

	// Create test app
	app = fiber.New()
	authCtrl = controllers.NewAuthController(db, cfg)
	routes.SetupRoutes(app, db, cfg)

	// Create test user
	testUser = models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$XvgWZzX7J6ybBp5nD5vQj.9vqJZJQ7Q8QJZJQ7Q8QJZJQ7Q8QJZJQ7Q8", // "password"
	}
	db.Create(&testUser)
}

func teardown() {
	// Clean up test database
	db.Migrator().DropTable(
		&models.User{},
		&models.UserProgress{},
		&models.LoginHistory{},
		&models.Course{},
		&models.Lesson{},
		&models.CourseComment{},
		&models.CourseAccessSettings{},
		&models.UserCourseProgress{},
		&models.Test{},
		&models.TestQuestion{},
		&models.TestComment{},
		&models.TestAccessSettings{},
		&models.UserTestProgress{},
	)
}

func TestRegister(t *testing.T) {
	registerData := map[string]string{
		"username":      "newuser",
		"email":         "newuser@example.com",
		"password_hash": "password123",
	}
	jsonData, _ := json.Marshal(registerData)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["token"])
	assert.NotEmpty(t, result["user"])
}

func TestLogin(t *testing.T) {
	loginData := map[string]string{
		"username": "testuser",
		"password": "password",
	}
	jsonData, _ := json.Marshal(loginData)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["token"])
	assert.NotEmpty(t, result["user"])

	jwtToken = result["token"].(string)
}

func TestGetProfile(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/user/profile", nil)
	req.Header.Set("Authorization", jwtToken)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "testuser", result["username"])
	assert.Equal(t, "test@example.com", result["email"])
}
