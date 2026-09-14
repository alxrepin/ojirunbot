package domain

import "time"

type DailyMeal struct {
	CreatedAt    time.Time
	Summary      string
	CaloriesKCal float64
	ProteinG     float64
	FatG         float64
	CarbsG       float64
}

type DailyTotals struct {
	CaloriesKCal float64
	ProteinG     float64
	FatG         float64
	CarbsG       float64
}

type ReportUser struct {
	User
	Profile Profile
}

type MealResultView struct {
	Analysis      NutritionAnalysis
	Profile       Profile
	ConsumedToday float64
}

const (
	ReportSent                = "sent"
	ReportFailed              = "failed"
	ReportSkippedUnsubscribed = "skipped_unsubscribed"
	ReportSkippedEmpty        = "skipped_empty"
)
