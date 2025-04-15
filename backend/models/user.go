package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string `gorm:"unique;not null" json:"username" example:"john_doe"`
	Email        string `gorm:"unique;not null" json:"email" example:"user@example.com" format:"email"`
	PasswordHash string `gorm:"not null" json:"password" example:"password"`
	Role         string `gorm:"default:user" json:"role" example:"user" enums:"user,admin"`
	Group        string `json:"group,omitempty" example:"philosophy_students"`
	University   string `json:"university,omitempty" example:"Harvard University"`
}

type UserProgress struct {
	gorm.Model
	UserID           uint
	LastActive       time.Time
	StreakDays       int `gorm:"default:0"`
	CoursesCompleted int `gorm:"default:0"`
	TestsCompleted   int `gorm:"default:0"`
}

type LoginHistory struct {
	gorm.Model
	UserID    uint
	LoginTime time.Time
}
