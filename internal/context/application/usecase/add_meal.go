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
	CreateEntryWithinLimit(ctx context.Context, userID string, chatID, sourceMessageID int64, mealDate, since time.Time, limits domain.MealLimits) (domain.MealEntry, error)
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
	users    userReader
	profiles profileReader
	meals    mealCreator
	limits   domain.MealLimits
	loc      *time.Location
}

func NewAddMeal(users userReader, profiles profileReader, meals mealCreator, limits domain.MealLimits, loc *time.Location) *AddMeal {
	if loc == nil {
		loc = time.UTC
	}
	return &AddMeal{users: users, profiles: profiles, meals: meals, limits: limits, loc: loc}
}

func (uc *AddMeal) DailyLimit() int {
	return max(uc.limits.PerUserPerDay, 0)
}

func (uc *AddMeal) Limits() domain.MealLimits {
	return uc.limits
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
	if uc.limits.PerUserPerDay <= 0 {
		return false, nil
	}
	created, err := uc.meals.CountCreatedSince(ctx, author.User.ID, uc.dayStart())
	if err != nil {
		return false, err
	}
	return created >= uc.limits.PerUserPerDay, nil
}

func (uc *AddMeal) CreateEntry(ctx context.Context, author AuthorizedAuthor, in AddMealInput) (domain.MealEntry, error) {
	return uc.meals.CreateEntryWithinLimit(ctx, author.User.ID, in.ChatID, in.SourceMessageID, in.MealDate, uc.dayStart(), uc.limits)
}

func (uc *AddMeal) dayStart() time.Time {
	now := time.Now().In(uc.loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, uc.loc)
}
