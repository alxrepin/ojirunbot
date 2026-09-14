package telegram

import (
	"context"
	"testing"
	"time"

	"ojirun/internal/context/application/service"
	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type answeringAPI struct {
	*recordingAPI
	alerts []string
}

func (a *answeringAPI) AnswerCallbackQuery(_ context.Context, _, text string, alert bool) error {
	if alert {
		a.alerts = append(a.alerts, text)
	}
	return nil
}

type fakeDay struct{}

func (fakeDay) MealsForReport(context.Context, string, time.Time) ([]domain.DailyMeal, domain.DailyTotals, error) {
	return nil, domain.DailyTotals{}, nil
}

func TestEditButtonRefusesAfterCorrectionLimit(t *testing.T) {
	api := &answeringAPI{recordingAPI: &recordingAPI{fakeAPI: &fakeAPI{}}}
	actions := &fakeMealActions{
		entry:    domain.MealEntry{ID: "m1", ChatID: 42, TelegramUserID: 42, Status: domain.StatusPending, BotMessageID: 3},
		revision: domain.MealRevision{Revision: domain.MaxCorrectionsPerEntry + 1},
	}
	r := &Router{
		api:       api,
		actions:   usecase.NewMealActions(actions),
		messenger: NewMealMessenger(api, noopBinder{}, discardLog()),
		log:       discardLog(),
	}
	cb := tg.CallbackQuery{
		ID:      "cb",
		From:    tg.User{ID: 42},
		Data:    tg.EncodeMealCallback(tg.ActionEdit, "m1"),
		Message: &tg.Message{MessageID: 3, Chat: tg.Chat{ID: 42, Type: "private"}},
	}

	r.handleCallback(context.Background(), cb)

	if len(api.alerts) != 1 || api.alerts[0] != render.CorrectionLimitAlert(domain.MaxCorrectionsPerEntry) || len(actions.edits) != 0 {
		t.Fatalf("expected a limit alert and no edit, got alerts %v edits %v", api.alerts, actions.edits)
	}

	actions.revision.Revision = domain.MaxCorrectionsPerEntry
	r.handleCallback(context.Background(), cb)

	if len(actions.edits) != 1 || len(api.alerts) != 1 {
		t.Fatalf("the last allowed correction should start editing, got edits %v alerts %v", actions.edits, api.alerts)
	}
}

func TestCorrectionTextAfterLimitRestoresCard(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	actions := &fakeMealActions{
		entry:    domain.MealEntry{ID: "m1", ChatID: 42, TelegramUserID: 42, Status: domain.StatusAwaitingFix, BotMessageID: 3},
		revision: domain.MealRevision{Revision: domain.MaxCorrectionsPerEntry + 1},
	}
	queue := &fakeJobQueue{}
	messenger := NewMealMessenger(api, noopBinder{}, discardLog())
	r := &Router{
		api:       api,
		actions:   usecase.NewMealActions(actions),
		mealJobs:  service.NewMealJobs(queue, nil, nil, nil),
		messenger: messenger,
		pipeline:  service.NewMealPipeline(nil, nil, nil, nil, fakeProfiles{profile: domain.Profile{DailyCaloriesKCal: 2000}}, fakeDay{}, messenger, "", "", time.UTC, discardLog()),
		log:       discardLog(),
	}

	if !r.handleCorrection(context.Background(), privateMessage("курицы было 200 г"), "курицы было 200 г") {
		t.Fatal("the text should be consumed as a correction attempt")
	}
	if len(actions.cancelled) != 1 || actions.cancelled[0] != "m1" {
		t.Fatalf("editing should be cancelled, got %v", actions.cancelled)
	}
	if len(queue.enqueued) != 0 {
		t.Fatalf("no correction should be queued, got %v", queue.enqueued)
	}
	if last := api.sent[len(api.sent)-1]; last != render.CorrectionLimit(domain.MaxCorrectionsPerEntry) {
		t.Fatalf("expected the limit notice, got %v", api.sent)
	}
}
