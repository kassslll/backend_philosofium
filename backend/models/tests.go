package models

import "gorm.io/gorm"

type Test struct {
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
	Questions      []TestQuestion
	Comments       []TestComment
	AccessSettings TestAccessSettings
}

type TestQuestion struct {
	gorm.Model
	TestID        uint
	Title         string
	Description   string
	Question      string
	Options       string // JSON array of options
	CorrectAnswer int
	SequenceOrder int
}

type TestAccessSettings struct {
	gorm.Model
	TestID          uint
	AccessLevel     string // public, private, restricted
	StartDate       string
	EndDate         string
	Admins          string // comma-separated IDs
	AttemptsAllowed int    `gorm:"default:1"`
}

type UserTestProgress struct {
	gorm.Model
	UserID            uint
	TestID            uint
	QuestionsAnswered int
	CorrectAnswers    int
	Score             float64
	AttemptsUsed      int
	LastAttempt       string
}
