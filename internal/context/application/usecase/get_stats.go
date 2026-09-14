package usecase

import (
	"context"
	"time"

	"ojirun/internal/context/domain"
)

type statsUserReader interface {
	GetByTelegramID(ctx context.Context, telegramUserID int64) (domain.User, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
}

type dayTotalsReader interface {
	MealsForReport(ctx context.Context, userID string, date time.Time) ([]domain.DailyMeal, domain.DailyTotals, error)
	HasMealsInChat(ctx context.Context, userID string, chatID int64) (bool, error)
}

type StatsResult struct {
	DisplayName string
	Profile     domain.Profile
	Totals      domain.DailyTotals
	MealCount   int
}

type GetStats struct {
	users    statsUserReader
	profiles profileReader
	day      dayTotalsReader
	loc      *time.Location
}

func NewGetStats(users statsUserReader, profiles profileReader, day dayTotalsReader, loc *time.Location) *GetStats {
	return &GetStats{users: users, profiles: profiles, day: day, loc: loc}
}

func (uc *GetStats) ForSelf(ctx context.Context, telegramUserID int64) (StatsResult, error) {
	user, err := uc.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return StatsResult{}, err
	}
	return uc.build(ctx, user)
}

func (uc *GetStats) ForUsername(ctx context.Context, username string, chatID int64) (StatsResult, error) {
	user, err := uc.users.GetByUsername(ctx, username)
	if err != nil {
		return StatsResult{}, err
	}
	visible, err := uc.day.HasMealsInChat(ctx, user.ID, chatID)
	if err != nil {
		return StatsResult{}, err
	}
	if !visible {
		return StatsResult{}, domain.ErrUserNotFound
	}
	return uc.build(ctx, user)
}

func (uc *GetStats) build(ctx context.Context, user domain.User) (StatsResult, error) {
	profile, err := uc.profiles.GetByUserID(ctx, user.ID)
	if err != nil {
		return StatsResult{}, err
	}
	meals, totals, err := uc.day.MealsForReport(ctx, user.ID, time.Now().In(uc.loc))
	if err != nil {
		return StatsResult{}, err
	}
	return StatsResult{
		DisplayName: user.DisplayName,
		Profile:     profile,
		Totals:      totals,
		MealCount:   len(meals),
	}, nil
}
