package models

import "gorm.io/gorm"

type CourseComment struct {
	gorm.Model
	CourseID  uint
	UserID    uint
	UserName  string
	UserImage string
	Text      string
	Rating    int `gorm:"check:rating>=0 AND rating<=5"`
	Replies   []CourseCommentReply
}

type CourseCommentReply struct {
	gorm.Model
	CommentID uint
	UserID    uint
	UserName  string
	UserImage string
	Text      string
}

type TestComment struct {
	gorm.Model
	TestID    uint
	UserID    uint
	UserName  string
	UserImage string
	Text      string
	Rating    int `gorm:"check:rating>=0 AND rating<=5"`
	Replies   []TestCommentReply
}

type TestCommentReply struct {
	gorm.Model
	CommentID uint
	UserID    uint
	UserName  string
	UserImage string
	Text      string
}

type CommentReport struct {
	gorm.Model
	CommentID   uint
	CommentType string // "course" or "test"
	ReportedBy  uint
	Reason      string
	Status      string // "pending", "reviewed", "resolved"
}
