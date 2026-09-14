package domain

const (
	QueueMeal   = "meal"
	QueueReport = "report"
)

const (
	JobMealAnalysis   = "meal_analysis"
	JobMealCorrection = "meal_correction"
	JobDailyReport    = "daily_report"
)

type Job struct {
	ID          string
	Queue       string
	Kind        string
	Payload     []byte
	Attempts    int
	MaxAttempts int
}
