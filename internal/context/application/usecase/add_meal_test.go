package usecase

import (
	"context"
	"testing"
	"time"

	"ojirun/internal/context/domain"
)

type countingMeals struct {
	created  int
	gotSince time.Time
	gotLimit int
}

func (c *countingMeals) CreateEntryWithinLimit(_ context.Context, _ string, _, _ int64, _, since time.Time, limit int) (domain.MealEntry, error) {
	c.gotSince, c.gotLimit = since, limit
	return domain.MealEntry{ID: "m1"}, nil
}

func (c *countingMeals) CountCreatedSince(_ context.Context, _ string, since time.Time) (int, error) {
	c.gotSince = since
	return c.created, nil
}

func TestAddMealDailyLimit(t *testing.T) {
	ctx := context.Background()
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skip("timezone database unavailable")
	}
	author := AuthorizedAuthor{User: domain.User{ID: "u1"}}

	meals := &countingMeals{created: 9}
	addMeal := NewAddMeal(nil, nil, meals, 10, loc)
	if reached, err := addMeal.LimitReached(ctx, author); err != nil || reached {
		t.Fatalf("9 of 10 meals: reached=%v err=%v", reached, err)
	}
	since := meals.gotSince.In(loc)
	if since.Hour() != 0 || since.Minute() != 0 || since.Day() != time.Now().In(loc).Day() {
		t.Fatalf("the day must start at local midnight, got %v", since)
	}

	meals.created = 10
	if reached, _ := addMeal.LimitReached(ctx, author); !reached {
		t.Fatal("10 of 10 meals should reach the limit")
	}
	if _, err := addMeal.CreateEntry(ctx, author, AddMealInput{}); err != nil || meals.gotLimit != 10 {
		t.Fatalf("CreateEntry must pass the limit to the repository (limit=%d, err=%v)", meals.gotLimit, err)
	}

	unlimited := NewAddMeal(nil, nil, &countingMeals{created: 1000}, 0, loc)
	if reached, _ := unlimited.LimitReached(ctx, author); reached || unlimited.DailyLimit() != 0 {
		t.Fatal("a zero limit disables the check")
	}
}
