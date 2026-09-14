package service

import (
	"context"
	"log/slog"
	"time"

	"ojirun/internal/context/domain"
)

type MealNotifier interface {
	Stage(ctx context.Context, entry domain.MealEntry, stage domain.MealStage)
	Result(ctx context.Context, entry domain.MealEntry, view domain.MealResultView)
	Fail(ctx context.Context, entry domain.MealEntry, failure domain.MealFailure)
	Discard(ctx context.Context, entry domain.MealEntry)
	Accepted(ctx context.Context, entry domain.MealEntry, view domain.MealResultView)
}

type mealStatusWriter interface {
	SetStatus(ctx context.Context, mealID string, status domain.Status) error
	SaveRevision(ctx context.Context, rev domain.MealRevision) (domain.MealRevision, error)
}

type photoArchiver interface {
	Archive(ctx context.Context, mealID string, ref domain.PhotoRef) (domain.Photo, error)
}

type photoStore interface {
	Save(ctx context.Context, p domain.Photo) error
	GetByMeal(ctx context.Context, mealID string) (domain.Photo, error)
}

type profileGetter interface {
	GetByUserID(ctx context.Context, userID string) (domain.Profile, error)
}

type dayReader interface {
	MealsForReport(ctx context.Context, userID string, date time.Time) ([]domain.DailyMeal, domain.DailyTotals, error)
}

type MealPipeline struct {
	analyzer          Analyzer
	archiver          photoArchiver
	meals             mealStatusWriter
	photos            photoStore
	profiles          profileGetter
	day               dayReader
	notifier          MealNotifier
	analysisPrompt    string
	correctionPrompt  string
	loc               *time.Location
	processTimeout    time.Duration
	consistencyTolere float64
	log               *slog.Logger
}

func NewMealPipeline(
	analyzer Analyzer,
	archiver photoArchiver,
	meals mealStatusWriter,
	photos photoStore,
	profiles profileGetter,
	day dayReader,
	notifier MealNotifier,
	analysisPrompt, correctionPrompt string,
	loc *time.Location,
	log *slog.Logger,
) *MealPipeline {
	return &MealPipeline{
		analyzer:          analyzer,
		archiver:          archiver,
		meals:             meals,
		photos:            photos,
		profiles:          profiles,
		day:               day,
		notifier:          notifier,
		analysisPrompt:    analysisPrompt,
		correctionPrompt:  correctionPrompt,
		loc:               loc,
		processTimeout:    3 * time.Minute,
		consistencyTolere: 0.2,
		log:               log,
	}
}

func (p *MealPipeline) Process(ctx context.Context, entry domain.MealEntry, profile domain.Profile, photo domain.PhotoRef, userText string) {
	ctx, cancel := context.WithTimeout(ctx, p.processTimeout)
	defer cancel()

	p.notifier.Stage(ctx, entry, domain.StageReceived)

	var photoMeta domain.Photo
	if photo.FileID != "" {
		p.notifier.Stage(ctx, entry, domain.StageDownloadingPhoto)
		p.setStatus(ctx, entry.ID, domain.StatusDownloadingPhoto)
		saved, err := p.archiver.Archive(ctx, entry.ID, photo)
		if err != nil {
			p.fail(ctx, entry, domain.FailDownloadPhoto, "save telegram photo failed", err)
			return
		}
		if err := p.photos.Save(ctx, saved); err != nil {
			p.fail(ctx, entry, domain.FailSavePhoto, "save photo meta failed", err)
			return
		}
		photoMeta = saved
	}

	p.notifier.Stage(ctx, entry, domain.StageAnalyzing)
	p.setStatus(ctx, entry.ID, domain.StatusAnalyzing)

	result, err := p.analyze(ctx, domain.AnalyzeRequest{
		Prompt:    p.analysisPrompt,
		UserText:  userText,
		ImagePath: photoMeta.LocalPath,
		ImageMime: photoMeta.MimeType,
	})
	if err != nil {
		p.fail(ctx, entry, domain.FailAnalyze, "analyze nutrition failed", err)
		return
	}

	if !result.Value.HasFood() {
		p.setStatus(ctx, entry.ID, domain.StatusDeleted)
		p.notifier.Discard(ctx, entry)
		return
	}

	revision := domain.NewRevision(entry.ID, userText, "", result.Value.Normalize(), result.RequestJSON)
	if _, err := p.meals.SaveRevision(ctx, revision); err != nil {
		p.fail(ctx, entry, domain.FailPersistResult, "save meal revision failed", err)
		return
	}

	p.present(ctx, entry, result.Value, profile)
}

func (p *MealPipeline) ProcessCorrection(ctx context.Context, entry domain.MealEntry, previous domain.MealRevision, correction string) {
	ctx, cancel := context.WithTimeout(ctx, p.processTimeout)
	defer cancel()

	p.notifier.Stage(ctx, entry, domain.StageReanalyzing)
	p.setStatus(ctx, entry.ID, domain.StatusReanalyzing)

	var photoPath, photoMime string
	if photo, err := p.photos.GetByMeal(ctx, entry.ID); err == nil {
		photoPath = photo.LocalPath
		photoMime = photo.MimeType
	}

	result, err := p.analyze(ctx, domain.AnalyzeRequest{
		Prompt:       p.correctionPrompt,
		UserText:     previous.UserText,
		ImagePath:    photoPath,
		ImageMime:    photoMime,
		Correction:   correction,
		PreviousJSON: previous.AIResponseJSON,
	})
	if err != nil {
		p.fail(ctx, entry, domain.FailReanalyze, "reanalyze nutrition failed", err)
		return
	}

	revision := domain.NewRevision(entry.ID, previous.UserText, correction, result.Value.Normalize(), result.RequestJSON)
	if _, err := p.meals.SaveRevision(ctx, revision); err != nil {
		p.fail(ctx, entry, domain.FailPersistCorrection, "save correction revision failed", err)
		return
	}

	profile, err := p.profiles.GetByUserID(ctx, entry.UserID)
	if err != nil {
		p.log.Error("get profile for correction render failed", "error", err)
		return
	}
	p.present(ctx, entry, result.Value, profile)
}

func (p *MealPipeline) ShowResult(ctx context.Context, entry domain.MealEntry, analysis domain.NutritionAnalysis) {
	profile, err := p.profiles.GetByUserID(ctx, entry.UserID)
	if err != nil {
		p.log.Error("get profile for result render failed", "error", err, "meal_entry_id", entry.ID)
		return
	}
	p.present(ctx, entry, analysis, profile)
}

func (p *MealPipeline) ShowAccepted(ctx context.Context, entry domain.MealEntry, analysis domain.NutritionAnalysis) {
	profile, err := p.profiles.GetByUserID(ctx, entry.UserID)
	if err != nil {
		p.log.Error("get profile for accepted meal failed", "error", err, "meal_entry_id", entry.ID)
		return
	}
	_, totals, _ := p.day.MealsForReport(ctx, entry.UserID, time.Now().In(p.loc))
	p.notifier.Accepted(ctx, entry, domain.MealResultView{
		Analysis:      analysis,
		Profile:       profile,
		ConsumedToday: totals.CaloriesKCal,
	})
}

func (p *MealPipeline) analyze(ctx context.Context, req domain.AnalyzeRequest) (domain.AIResult[domain.NutritionAnalysis], error) {
	result, err := p.analyzer.AnalyzeNutrition(ctx, req)
	if err == nil && !result.Value.TotalsConsistent(p.consistencyTolere) {
		p.log.Warn("ai meal totals inconsistent with item sum", "total_kcal", result.Value.Total.CaloriesKCal)
	}
	return result, err
}

func (p *MealPipeline) present(ctx context.Context, entry domain.MealEntry, analysis domain.NutritionAnalysis, profile domain.Profile) {
	_, totals, _ := p.day.MealsForReport(ctx, entry.UserID, time.Now().In(p.loc))
	p.notifier.Result(ctx, entry, domain.MealResultView{
		Analysis:      analysis,
		Profile:       profile,
		ConsumedToday: totals.CaloriesKCal + analysis.Total.CaloriesKCal,
	})
}

func (p *MealPipeline) setStatus(ctx context.Context, mealID string, status domain.Status) {
	if err := p.meals.SetStatus(ctx, mealID, status); err != nil {
		p.log.Warn("set meal status failed", "error", err, "meal_entry_id", mealID, "status", status)
	}
}

func (p *MealPipeline) fail(ctx context.Context, entry domain.MealEntry, failure domain.MealFailure, logMsg string, err error) {
	p.setStatus(ctx, entry.ID, domain.StatusFailed)
	p.notifier.Fail(ctx, entry, failure)
	p.log.Error(logMsg, "error", err, "meal_entry_id", entry.ID)
}
