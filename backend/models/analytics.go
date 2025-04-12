package models

import "gorm.io/gorm"

type CourseAnalytics struct {
	gorm.Model
	CourseID         uint
	UserID           uint
	UserName         string
	LessonsCompleted int
	HoursSpent       float64
	LastAccessed     string
	CompletionRate   float64
}

type TestAnalytics struct {
	gorm.Model
	TestID            uint
	UserID            uint
	UserName          string
	QuestionsAnswered int
	CorrectAnswers    int
	WrongAnswers      int
	Score             float64
	AttemptNumber     int
	TimeSpent         float64 // in minutes
}

type UserActivity struct {
	gorm.Model
	UserID      uint
	ActionType  string // "course_start", "course_complete", "test_start", "test_complete"
	TargetID    uint   // course_id or test_id
	TargetTitle string
	Timestamp   string
	Duration    float64 // for completed actions
}

type PlatformAnalytics struct {
	gorm.Model
	TotalUsers        int
	ActiveUsers       int
	CoursesCreated    int
	TestsCreated      int
	AvgCourseProgress float64
	AvgTestScore      float64
	Date              string // for daily analytics
}
