package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ojirun/internal/context/domain"
)

type ReportNotifier interface {
	Send(ctx context.Context, user domain.ReportUser, reportDate time.Time, meals []domain.DailyMeal, totals domain.DailyTotals, rec *domain.DailyRecommendation) (botMessageID int64, err error)
}

type reportRepo interface {
	EnqueueDailyReports(ctx context.Context, reportDate time.Time) (int64, error)
	ReportUser(ctx context.Context, userID string) (domain.ReportUser, error)
	MealsForReport(ctx context.Context, userID string, date time.Time) ([]domain.DailyMeal, domain.DailyTotals, error)
	TryCreate(ctx context.Context, userID string, chatID int64, reportDate time.Time) (created bool, reportID string, err error)
	SetStatus(ctx context.Context, reportID, status string) error
	MarkSent(ctx context.Context, reportID string, botMessageID int64, requestJSON, responseJSON []byte) error
}

type subscriberChecker interface {
	IsSubscribed(ctx context.Context, telegramUserID int64) (bool, error)
}

const reportDateLayout = "2006-01-02"

type reportJobPayload struct {
	UserID     string `json:"user_id"`
	ReportDate string `json:"report_date"`
}

type Report struct {
	repo                 reportRepo
	analyzer             Analyzer
	notifier             ReportNotifier
	subscribers          subscriberChecker
	recommendationPrompt string
	loc                  *time.Location
	log                  *slog.Logger
}

func NewReport(repo reportRepo, analyzer Analyzer, notifier ReportNotifier, subscribers subscriberChecker, recommendationPrompt string, loc *time.Location, log *slog.Logger) *Report {
	return &Report{
		repo:                 repo,
		analyzer:             analyzer,
		notifier:             notifier,
		subscribers:          subscribers,
		recommendationPrompt: recommendationPrompt,
		loc:                  loc,
		log:                  log,
	}
}

func (s *Report) Schedule(ctx context.Context, reportDate time.Time) (int64, error) {
	return s.repo.EnqueueDailyReports(ctx, reportDate)
}

func (s *Report) Handle(ctx context.Context, job domain.Job) error {
	var payload reportJobPayload
	if err := decodeJob(job, &payload); err != nil {
		return err
	}
	date, err := time.ParseInLocation(reportDateLayout, payload.ReportDate, s.loc)
	if err != nil {
		return permanentError{fmt.Errorf("parse report date: %w", err)}
	}
	user, err := s.repo.ReportUser(ctx, payload.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	created, reportID, err := s.repo.TryCreate(ctx, user.ID, user.TelegramUserID, date)
	if err != nil || !created {
		return err
	}
	subscribed, err := s.subscribers.IsSubscribed(ctx, user.TelegramUserID)
	if err != nil {
		s.release(ctx, reportID)
		return err
	}
	if !subscribed {
		return s.repo.SetStatus(ctx, reportID, domain.ReportSkippedUnsubscribed)
	}

	meals, totals, err := s.repo.MealsForReport(ctx, user.ID, date)
	if err != nil {
		s.release(ctx, reportID)
		return err
	}
	if len(meals) == 0 {
		return s.repo.SetStatus(ctx, reportID, domain.ReportSkippedEmpty)
	}

	rec, requestJSON, responseJSON := s.recommend(ctx, user, meals)
	botMessageID, err := s.notifier.Send(ctx, user, date, meals, totals, rec)
	if err != nil {
		s.release(ctx, reportID)
		return fmt.Errorf("send daily report: %w", err)
	}
	return s.repo.MarkSent(ctx, reportID, botMessageID, requestJSON, responseJSON)
}

func (s *Report) release(ctx context.Context, reportID string) {
	if err := s.repo.SetStatus(ctx, reportID, domain.ReportFailed); err != nil {
		s.log.Error("release daily report slot failed", "error", err, "report_id", reportID)
	}
}

func (s *Report) recommend(ctx context.Context, user domain.ReportUser, meals []domain.DailyMeal) (*domain.DailyRecommendation, []byte, []byte) {
	reportJSON, _ := json.Marshal(meals)
	targetsJSON, _ := json.Marshal(newRecommendationTargets(user.Profile))
	result, err := s.analyzer.DailyRecommendation(ctx, domain.RecommendationRequest{
		Prompt:      s.recommendationPrompt,
		UserName:    user.DisplayName,
		ReportJSON:  reportJSON,
		TargetsJSON: targetsJSON,
	})
	if err != nil {
		s.log.Error("daily recommendation failed", "error", err, "user_id", user.ID)
		return nil, nil, nil
	}
	return &result.Value, result.RequestJSON, result.ResponseJSON
}

type recommendationTargets struct {
	Sex               string  `json:"sex"`
	Age               int     `json:"age"`
	HeightCM          float64 `json:"height_cm"`
	WeightKG          float64 `json:"weight_kg"`
	ActivityLevel     string  `json:"activity_level"`
	Goal              string  `json:"goal"`
	DailyCaloriesKCal int     `json:"daily_calories_kcal"`
	DailyProteinG     int     `json:"daily_protein_g"`
	DailyFatG         int     `json:"daily_fat_g"`
	DailyCarbsG       int     `json:"daily_carbs_g"`
}

func newRecommendationTargets(p domain.Profile) recommendationTargets {
	return recommendationTargets{
		Sex:               p.Sex,
		Age:               p.Age,
		HeightCM:          p.HeightCM,
		WeightKG:          p.WeightKG,
		ActivityLevel:     p.ActivityLevel,
		Goal:              p.Goal,
		DailyCaloriesKCal: p.DailyCaloriesKCal,
		DailyProteinG:     p.DailyProteinG,
		DailyFatG:         p.DailyFatG,
		DailyCarbsG:       p.DailyCarbsG,
	}
}
