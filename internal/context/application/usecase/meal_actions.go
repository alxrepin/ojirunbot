package usecase

import (
	"context"
	"time"

	"ojirun/internal/context/domain"
)

type mealActionRepo interface {
	GetForAction(ctx context.Context, mealID string) (domain.MealEntry, domain.MealRevision, error)
	Accept(ctx context.Context, mealID string, telegramUserID int64) (bool, error)
	AwaitCorrection(ctx context.Context, mealID string, telegramUserID int64) (bool, error)
	CancelCorrection(ctx context.Context, mealID string, telegramUserID int64) (bool, error)
	Delete(ctx context.Context, mealID string, telegramUserID int64) (bool, error)
	FindAwaitingCorrection(ctx context.Context, telegramUserID, chatID int64) (domain.MealEntry, domain.MealRevision, error)
	FindByMessage(ctx context.Context, chatID, messageID int64) (domain.MealEntry, error)
	SetMealDate(ctx context.Context, mealID string, telegramUserID int64, date time.Time) (bool, error)
}

type MealActions struct {
	meals mealActionRepo
}

func NewMealActions(meals mealActionRepo) *MealActions {
	return &MealActions{meals: meals}
}

func (uc *MealActions) Load(ctx context.Context, mealID string) (domain.MealEntry, domain.MealRevision, error) {
	return uc.meals.GetForAction(ctx, mealID)
}

func (uc *MealActions) Accept(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	return uc.meals.Accept(ctx, mealID, telegramUserID)
}

func (uc *MealActions) RequestEdit(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	return uc.meals.AwaitCorrection(ctx, mealID, telegramUserID)
}

func (uc *MealActions) CancelEdit(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	return uc.meals.CancelCorrection(ctx, mealID, telegramUserID)
}

func (uc *MealActions) Delete(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	return uc.meals.Delete(ctx, mealID, telegramUserID)
}

func (uc *MealActions) FindByMessage(ctx context.Context, chatID, messageID int64) (domain.MealEntry, error) {
	return uc.meals.FindByMessage(ctx, chatID, messageID)
}

func (uc *MealActions) SetMealDate(ctx context.Context, mealID string, telegramUserID int64, date time.Time) (bool, error) {
	return uc.meals.SetMealDate(ctx, mealID, telegramUserID, date)
}

func (uc *MealActions) FindAwaitingCorrection(ctx context.Context, telegramUserID, chatID int64) (domain.MealEntry, domain.MealRevision, error) {
	return uc.meals.FindAwaitingCorrection(ctx, telegramUserID, chatID)
}
