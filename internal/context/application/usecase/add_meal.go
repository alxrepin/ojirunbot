package usecase

import (
	"context"
	"time"

	"ojirun/internal/context/domain"
)

type userReader interface {
	GetByTelegramID(ctx context.Context, telegramUserID int64) (domain.User, error)
}

type profileReader interface {
	GetByUserID(ctx context.Context, userID string) (domain.Profile, error)
}

type mealCreator interface {
	CreateEntryWithinLimit(ctx context.Context, userID string, chatID, sourceMessageID int64, mealDate, since time.Time, limit int) (domain.MealEntry, error)
	CountCreatedSince(ctx context.Context, userID string, since time.Time) (int, error)
}

type AddMealInput struct {
	ChatID          int64
	SourceMessageID int64
	MealDate        time.Time
}

type AuthorizedAuthor struct {
	User    domain.User
	Profile domain.Profile
}

type AddMeal struct {
	users      userReader
	profiles   profileReader
	meals      mealCreator
	dailyLimit int
	loc        *time.Location
}

func NewAddMeal(users userReader, profiles profileReader, meals mealCreator, dailyLimit int, loc *time.Location) *AddMeal {
	if loc == nil {
		loc = time.UTC
	}
	return &AddMeal{users: users, profiles: profiles, meals: meals, dailyLimit: dailyLimit, loc: loc}
}

func (uc *AddMeal) DailyLimit() int {
	return max(uc.dailyLimit, 0)
}

func (uc *AddMeal) Authorize(ctx context.Context, telegramUserID int64) (AuthorizedAuthor, error) {
	user, err := uc.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return AuthorizedAuthor{}, err
	}
	profile, err := uc.profiles.GetByUserID(ctx, user.ID)
	if err != nil {
		return AuthorizedAuthor{}, err
	}
	return AuthorizedAuthor{User: user, Profile: profile}, nil
}

func (uc *AddMeal) LimitReached(ctx context.Context, author AuthorizedAuthor) (bool, error) {
	if uc.dailyLimit <= 0 {
		return false, nil
	}
	created, err := uc.meals.CountCreatedSince(ctx, author.User.ID, uc.dayStart())
	if err != nil {
		return false, err
	}
	return created >= uc.dailyLimit, nil
}

func (uc *AddMeal) CreateEntry(ctx context.Context, author AuthorizedAuthor, in AddMealInput) (domain.MealEntry, error) {
	return uc.meals.CreateEntryWithinLimit(ctx, author.User.ID, in.ChatID, in.SourceMessageID, in.MealDate, uc.dayStart(), uc.dailyLimit)
}

func (uc *AddMeal) dayStart() time.Time {
	now := time.Now().In(uc.loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, uc.loc)
}
