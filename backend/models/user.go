package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string `gorm:"unique;not null"`
	Email        string `gorm:"unique;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"default:user"` // user, admin
	Group        string
	University   string
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
