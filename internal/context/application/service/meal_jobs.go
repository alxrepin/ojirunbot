package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"ojirun/internal/context/domain"
)

type mealJobLoader interface {
	GetEntry(ctx context.Context, mealID string) (domain.MealEntry, error)
	GetForAction(ctx context.Context, mealID string) (domain.MealEntry, domain.MealRevision, error)
}

type mealAnalysisPayload struct {
	MealID            string `json:"meal_id"`
	PhotoFileID       string `json:"photo_file_id,omitempty"`
	PhotoFileUniqueID string `json:"photo_file_unique_id,omitempty"`
	Description       string `json:"description"`
}

type mealCorrectionPayload struct {
	MealID     string `json:"meal_id"`
	Correction string `json:"correction"`
}

var analysisStatuses = []domain.Status{domain.StatusReceived, domain.StatusDownloadingPhoto, domain.StatusAnalyzing}

type MealJobs struct {
	queue    jobQueue
	meals    mealJobLoader
	profiles profileGetter
	pipeline *MealPipeline
	wake     func()
}

func NewMealJobs(queue jobQueue, meals mealJobLoader, profiles profileGetter, pipeline *MealPipeline) *MealJobs {
	return &MealJobs{queue: queue, meals: meals, profiles: profiles, pipeline: pipeline}
}

func (m *MealJobs) OnEnqueue(wake func()) {
	m.wake = wake
}

func (m *MealJobs) Backlog(ctx context.Context, chatID int64) (int, error) {
	return m.queue.Backlog(ctx, domain.QueueMeal, chatID)
}

func (m *MealJobs) EnqueueAnalysis(ctx context.Context, mealID string, chatID int64, photo domain.PhotoRef, description string) error {
	return m.enqueue(ctx, domain.JobMealAnalysis, chatID, mealAnalysisPayload{
		MealID:            mealID,
		PhotoFileID:       photo.FileID,
		PhotoFileUniqueID: photo.FileUniqueID,
		Description:       description,
	})
}

func (m *MealJobs) EnqueueCorrection(ctx context.Context, mealID string, chatID int64, correction string) error {
	return m.enqueue(ctx, domain.JobMealCorrection, chatID, mealCorrectionPayload{MealID: mealID, Correction: correction})
}

func (m *MealJobs) enqueue(ctx context.Context, kind string, chatID int64, payload any) error {
	if err := m.queue.Enqueue(ctx, domain.QueueMeal, kind, payload, "", chatID); err != nil {
		return err
	}
	if m.wake != nil {
		m.wake()
	}
	return nil
}

func (m *MealJobs) Handle(ctx context.Context, job domain.Job) error {
	switch job.Kind {
	case domain.JobMealAnalysis:
		return m.handleAnalysis(ctx, job)
	case domain.JobMealCorrection:
		return m.handleCorrection(ctx, job)
	default:
		return permanentError{fmt.Errorf("unknown meal job kind %q", job.Kind)}
	}
}

func (m *MealJobs) handleAnalysis(ctx context.Context, job domain.Job) error {
	var payload mealAnalysisPayload
	if err := decodeJob(job, &payload); err != nil {
		return err
	}
	entry, err := m.meals.GetEntry(ctx, payload.MealID)
	if errors.Is(err, domain.ErrMealNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !slices.Contains(analysisStatuses, entry.Status) {
		return nil
	}
	profile, err := m.profiles.GetByUserID(ctx, entry.UserID)
	if err != nil {
		return err
	}
	photo := domain.PhotoRef{FileID: payload.PhotoFileID, FileUniqueID: payload.PhotoFileUniqueID}
	m.pipeline.Process(ctx, entry, profile, photo, payload.Description)
	return nil
}

func (m *MealJobs) handleCorrection(ctx context.Context, job domain.Job) error {
	var payload mealCorrectionPayload
	if err := decodeJob(job, &payload); err != nil {
		return err
	}
	entry, previous, err := m.meals.GetForAction(ctx, payload.MealID)
	if errors.Is(err, domain.ErrMealNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if entry.Status != domain.StatusAwaitingFix && entry.Status != domain.StatusReanalyzing {
		return nil
	}
	m.pipeline.ProcessCorrection(ctx, entry, previous, payload.Correction)
	return nil
}
