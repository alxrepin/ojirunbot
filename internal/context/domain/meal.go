package domain

import (
	"slices"
	"time"
)

type MealEntry struct {
	ID                 string
	UserID             string
	TelegramUserID     int64
	ChatID             int64
	SourceMessageID    int64
	BotMessageID       int64
	EphemeralMessageID int64
	Status             Status
	MealDate           time.Time
	CreatedAt          time.Time
}

type Status string

const (
	StatusReceived         Status = "received"
	StatusDownloadingPhoto Status = "downloading_photo"
	StatusAnalyzing        Status = "analyzing"
	StatusPending          Status = "pending_confirmation"
	StatusAwaitingFix      Status = "awaiting_correction"
	StatusReanalyzing      Status = "reanalyzing"
	StatusAccepted         Status = "accepted"
	StatusDeleted          Status = "deleted"
	StatusFailed           Status = "failed"
)

var transitions = map[Status][]Status{
	StatusReceived:         {StatusDownloadingPhoto, StatusAnalyzing, StatusFailed, StatusDeleted},
	StatusDownloadingPhoto: {StatusAnalyzing, StatusFailed, StatusDeleted},
	StatusAnalyzing:        {StatusPending, StatusFailed, StatusDeleted},
	StatusPending:          {StatusAwaitingFix, StatusAccepted, StatusDeleted},
	StatusAwaitingFix:      {StatusReanalyzing, StatusPending, StatusDeleted},
	StatusReanalyzing:      {StatusPending, StatusFailed, StatusDeleted},
	StatusAccepted:         {StatusAccepted, StatusDeleted},
	StatusFailed:           {StatusDeleted},
	StatusDeleted:          {},
}

func CanTransition(from, to Status) bool {
	return slices.Contains(transitions[from], to)
}

func (s Status) IsTerminal() bool {
	return s == StatusDeleted
}

func (s Status) InFlight() bool {
	switch s {
	case StatusReceived, StatusDownloadingPhoto, StatusAnalyzing, StatusReanalyzing:
		return true
	default:
		return false
	}
}

type MealStage int

const (
	StageReceived MealStage = iota
	StageDownloadingPhoto
	StageAnalyzing
	StageReanalyzing
)

type MealFailure int

const (
	FailDownloadPhoto MealFailure = iota
	FailSavePhoto
	FailAnalyze
	FailPersistResult
	FailReanalyze
	FailPersistCorrection
)
