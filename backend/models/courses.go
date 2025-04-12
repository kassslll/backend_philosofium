package models

import "gorm.io/gorm"

type Course struct {
	gorm.Model
	Title          string
	ShortDesc      string
	Description    string
	Difficulty     string // beginner, intermediate, advanced
	RecommendedFor string // group
	University     string
	Topic          string
	AuthorID       uint
	LogoURL        string
	CompletionRate float64
	Lessons        []Lesson
	Comments       []CourseComment
	AccessSettings CourseAccessSettings
}

type Lesson struct {
	gorm.Model
	CourseID      uint
	Title         string
	Description   string
	Content       string
	SequenceOrder int
}

type CourseAccessSettings struct {
	gorm.Model
	CourseID    uint
	AccessLevel string // public, private, restricted
	StartDate   string
	EndDate     string
	Admins      string // comma-separated IDs
}

type UserCourseProgress struct {
	gorm.Model
	UserID           uint
	CourseID         uint
	LessonsCompleted int
	HoursSpent       float64
	LastAccessed     string
	CompletionRate   float64
}
