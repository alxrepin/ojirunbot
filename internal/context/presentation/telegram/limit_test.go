package telegram

import (
	"context"
	"testing"
	"time"

	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	"ojirun/internal/context/presentation/telegram/render"
)

func limitedRouter(api API, meals fakeMeals, limits domain.MealLimits) *Router {
	r := newInputRouter(api, &inputSessions{})
	r.addMeal = usecase.NewAddMeal(
		fakeUsers{user: domain.User{ID: "u1", TelegramUserID: 42}},
		fakeProfiles{profile: domain.Profile{UserID: "u1"}},
		meals,
		limits, time.UTC,
	)
	return r
}

func TestDailyMealLimit(t *testing.T) {
	ctx := context.Background()
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := limitedRouter(api, fakeMeals{err: domain.ErrDailyMealLimit, created: 10}, domain.MealLimits{PerUserPerDay: 10})
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

func TestChatAndInFlightLimits(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		err  error
		want string
	}{
		"chat limit":      {domain.ErrChatMealLimit, render.ChatDailyLimit(100)},
		"in-flight limit": {domain.ErrMealsInFlight, render.MealsInFlight(3)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := &recordingAPI{fakeAPI: &fakeAPI{}}
			r := limitedRouter(api, fakeMeals{err: tc.err}, domain.MealLimits{PerUserPerDay: 10, PerChatPerDay: 100, InFlight: 3})

			meal := groupMessage()
			meal.Text = "/add гречка"
			r.handleGroupMessage(ctx, meal)

			if len(api.sent) != 1 || api.sent[0] != tc.want || api.opts[0].Ephemeral == nil {
				t.Fatalf("expected an ephemeral notice %q, got %v", tc.want, api.sent)
			}
		})
	}
}
