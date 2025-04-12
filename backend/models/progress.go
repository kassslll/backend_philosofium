package models

import "time"

type MonthlyProgress struct {
	Month            time.Month
	Year             int
	StreakDays       int
	CoursesCompleted int
	LoginFrequency   map[string]int // day -> count
}

type ProgressOverview struct {
	TotalStreakDays       int
	TotalCoursesCompleted int
	TotalTestsCompleted   int
	MonthlyProgress       []MonthlyProgress
}
