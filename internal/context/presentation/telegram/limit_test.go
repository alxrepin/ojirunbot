package telegram

import (
	"context"
	"testing"
	"time"

	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	"ojirun/internal/context/presentation/telegram/render"
)

func TestDailyMealLimit(t *testing.T) {
	ctx := context.Background()
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := newInputRouter(api, &inputSessions{})
	r.addMeal = usecase.NewAddMeal(
		fakeUsers{user: domain.User{ID: "u1", TelegramUserID: 42}},
		fakeProfiles{profile: domain.Profile{UserID: "u1"}},
		fakeMeals{err: domain.ErrDailyMealLimit, created: 10},
		10, time.UTC,
	)
	want := render.MealDailyLimit(10)

	meal := groupMessage()
	meal.Text = "/add гречка"
	r.handleGroupMessage(ctx, meal)
	if len(api.sent) != 1 || api.sent[0] != want || api.opts[0].Ephemeral == nil {
		t.Fatalf("expected an ephemeral limit notice, got %v", api.sent)
	}

	bare := groupMessage()
	bare.Text = "/add"
	r.handleGroupMessage(ctx, bare)
	if len(api.sent) != 2 || api.sent[1] != want {
		t.Fatalf("input mode must not open once the limit is reached, got %v", api.sent)
	}
}
