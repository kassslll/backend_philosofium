package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestCreateCourse(t *testing.T) {
	courseData := map[string]interface{}{
		"title":           "Test Course",
		"short_desc":      "Short description",
		"description":     "Full description",
		"difficulty":      "beginner",
		"recommended_for": "students",
		"university":      "Test University",
		"topic":           "Programming",
	}
	jsonData, _ := json.Marshal(courseData)

	req := httptest.NewRequest("POST", "/api/admin/courses", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", jwtToken)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Course created", result["message"])
	assert.Equal(t, "Test Course", result["course"].(map[string]interface{})["title"])
}

func TestGetCourseDetails(t *testing.T) {
	// First create a course
	courseData := map[string]interface{}{
		"title":      "Test Course Details",
		"short_desc": "Short description",
	}
	jsonData, _ := json.Marshal(courseData)

	createReq := httptest.NewRequest("POST", "/api/admin/courses", bytes.NewBuffer(jsonData))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", jwtToken)

	createResp, _ := app.Test(createReq)
	var createResult map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&createResult)
	courseID := createResult["course"].(map[string]interface{})["id"]

	// Now get course details
	req := httptest.NewRequest("GET", "/api/courses/"+courseID.(string), nil)
	req.Header.Set("Authorization", jwtToken)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Test Course Details", result["course"].(map[string]interface{})["title"])
}

func TestUpdateCourseProgress(t *testing.T) {
	// First create a course
	courseData := map[string]interface{}{
		"title": "Progress Test Course",
	}
	jsonData, _ := json.Marshal(courseData)

	createReq := httptest.NewRequest("POST", "/api/admin/courses", bytes.NewBuffer(jsonData))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", jwtToken)

	createResp, _ := app.Test(createReq)
	var createResult map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&createResult)
	courseID := createResult["course"].(map[string]interface{})["id"]

	// Add a lesson to the course
	lessonData := map[string]interface{}{
		"title":       "Test Lesson",
		"description": "Lesson description",
		"content":     "Lesson content",
	}
	lessonJson, _ := json.Marshal(lessonData)

	lessonReq := httptest.NewRequest("POST", "/api/admin/courses/"+courseID.(string)+"/lessons", bytes.NewBuffer(lessonJson))
	lessonReq.Header.Set("Content-Type", "application/json")
	lessonReq.Header.Set("Authorization", jwtToken)

	app.Test(lessonReq)

	// Update progress
	progressData := map[string]interface{}{
		"lesson_id":      1,
		"hours_spent":    2.5,
		"mark_completed": true,
	}
	progressJson, _ := json.Marshal(progressData)

	progressReq := httptest.NewRequest("POST", "/api/courses/"+courseID.(string)+"/progress", bytes.NewBuffer(progressJson))
	progressReq.Header.Set("Content-Type", "application/json")
	progressReq.Header.Set("Authorization", jwtToken)

	progressResp, err := app.Test(progressReq)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, progressResp.StatusCode)

	var progressResult map[string]interface{}
	json.NewDecoder(progressResp.Body).Decode(&progressResult)
	assert.Equal(t, "Progress updated", progressResult["message"])
	assert.Equal(t, 1, int(progressResult["progress"].(map[string]interface{})["lessons_completed"].(float64)))
	assert.Equal(t, 2.5, progressResult["progress"].(map[string]interface{})["hours_spent"].(float64))
}
