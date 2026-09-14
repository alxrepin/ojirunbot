package domain

import "time"

type SessionKind string

const (
	SessionRegistration SessionKind = "registration"
	SessionSettings     SessionKind = "settings"
	SessionMealInput    SessionKind = "meal_input"
)

type Session struct {
	ID             string
	Kind           SessionKind
	UserID         string
	TelegramUserID int64
	ChatID         int64
	Step           string
	PayloadJSON    []byte
	BotMessageID   int64
	Status         string
	ExpiresAt      time.Time
}

type RegistrationDraft struct {
	Sex           string
	Age           int
	HeightCM      float64
	WeightKG      float64
	ActivityLevel string
	Goal          string
}
