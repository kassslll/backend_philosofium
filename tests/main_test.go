package tests

import (
	"testing"
)

func TestAll(t *testing.T) {
	t.Run("Auth", TestAuth)
	t.Run("Courses", TestCourses)
}

func TestCourses(t *testing.T) {
	t.Run("CreateCourse", TestCreateCourse)
	t.Run("GetCourseDetails", TestGetCourseDetails)
	t.Run("UpdateCourseProgress", TestUpdateCourseProgress)
}

func TestAuth(t *testing.T) {
	// Здесь ты можешь вызвать нужные тесты для авторизации
	t.Run("Register", TestRegister)
	t.Run("Login", TestLogin)
	t.Run("GetProfile", TestGetProfile)
}
