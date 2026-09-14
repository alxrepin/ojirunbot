package telegram

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type fakeMealActions struct {
	entry     domain.MealEntry
	revision  domain.MealRevision
	findErr   error
	movable   bool
	gotDate   time.Time
	gotMeal   string
	gotUser   int64
	setCalls  int
	deleted   []string
	edits     []string
	cancelled []string
}

func (f *fakeMealActions) FindByMessage(context.Context, int64, int64) (domain.MealEntry, error) {
	return f.entry, f.findErr
}

func (f *fakeMealActions) SetMealDate(_ context.Context, mealID string, telegramUserID int64, date time.Time) (bool, error) {
	f.setCalls++
	f.gotMeal, f.gotUser, f.gotDate = mealID, telegramUserID, date
	return f.movable, nil
}

func (f *fakeMealActions) GetForAction(context.Context, string) (domain.MealEntry, domain.MealRevision, error) {
	return f.entry, f.revision, nil
}

func (f *fakeMealActions) Accept(context.Context, string, int64) (bool, error) { return false, nil }

func (f *fakeMealActions) AwaitCorrection(_ context.Context, mealID string, _ int64) (bool, error) {
	f.edits = append(f.edits, mealID)
	return true, nil
}

func (f *fakeMealActions) CancelCorrection(_ context.Context, mealID string, _ int64) (bool, error) {
	f.cancelled = append(f.cancelled, mealID)
	return true, nil
}

func (f *fakeMealActions) Delete(_ context.Context, mealID string, telegramUserID int64) (bool, error) {
	if f.entry.ID != mealID || f.entry.TelegramUserID != telegramUserID {
		return false, nil
	}
	f.deleted = append(f.deleted, mealID)
	return true, nil
}

func (f *fakeMealActions) FindAwaitingCorrection(context.Context, int64, int64) (domain.MealEntry, domain.MealRevision, error) {
	if f.entry.Status != domain.StatusAwaitingFix {
		return domain.MealEntry{}, domain.MealRevision{}, domain.ErrMealNotFound
	}
	return f.entry, f.revision, nil
}

func yesterdayRouter(api API, repo *fakeMealActions, loc *time.Location) *Router {
	return &Router{
		api:     api,
		actions: usecase.NewMealActions(repo),
		loc:     loc,
		log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func replyTo(sender, targetID int64) tg.Message {
	return tg.Message{
		MessageID:      9,
		From:           &tg.User{ID: sender},
		Chat:           tg.Chat{ID: 100, Type: "supergroup"},
		ReplyToMessage: &tg.Message{MessageID: targetID, From: &tg.User{ID: sender}},
	}
}

func TestHandleYesterdayNeedsReply(t *testing.T) {
	api := &fakeAPI{}
	repo := &fakeMealActions{}
	r := yesterdayRouter(api, repo, time.UTC)

	r.handleYesterday(context.Background(), tg.Message{
		MessageID: 9,
		From:      &tg.User{ID: 42},
		Chat:      tg.Chat{ID: 100, Type: "supergroup"},
	})

	if len(api.sent) != 1 || api.sent[0] != render.YesterdayNeedReply() {
		t.Fatalf("expected need-reply notice, got %v", api.sent)
	}
	if repo.setCalls != 0 {
		t.Fatalf("SetMealDate must not run, got %d calls", repo.setCalls)
	}
}

func TestHandleYesterdayRejectsNonAuthor(t *testing.T) {
	api := &fakeAPI{}
	repo := &fakeMealActions{
		entry:   domain.MealEntry{ID: "m1", TelegramUserID: 42, Status: domain.StatusAccepted},
		movable: true,
	}
	r := yesterdayRouter(api, repo, time.UTC)

	r.handleYesterday(context.Background(), replyTo(99, 1))

	if len(api.sent) != 1 || api.sent[0] != render.YesterdayNotAuthor() {
		t.Fatalf("expected not-author notice, got %v", api.sent)
	}
	if repo.setCalls != 0 {
		t.Fatalf("SetMealDate must not run for a non-author, got %d calls", repo.setCalls)
	}
}

func TestHandleYesterdayUnknownMessage(t *testing.T) {
	api := &fakeAPI{}
	repo := &fakeMealActions{findErr: domain.ErrMealNotFound}
	r := yesterdayRouter(api, repo, time.UTC)

	r.handleYesterday(context.Background(), replyTo(42, 1))

	if len(api.sent) != 1 || api.sent[0] != render.YesterdayNoEntry() {
		t.Fatalf("expected no-entry notice, got %v", api.sent)
	}
}

func TestHandleYesterdayNotMovable(t *testing.T) {
	api := &fakeAPI{}
	repo := &fakeMealActions{
		entry:   domain.MealEntry{ID: "m1", TelegramUserID: 42, Status: domain.StatusDeleted},
		movable: false,
	}
	r := yesterdayRouter(api, repo, time.UTC)

	r.handleYesterday(context.Background(), replyTo(42, 1))

	if len(api.sent) != 1 || api.sent[0] != render.YesterdayNotMovable() {
		t.Fatalf("expected not-movable notice, got %v", api.sent)
	}
}

func TestHandleYesterdayMovesToDayBeforeCreation(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)
	created := time.Date(2026, 7, 10, 0, 30, 0, 0, moscow)
	repo := &fakeMealActions{
		entry: domain.MealEntry{
			ID:             "m1",
			TelegramUserID: 42,
			Status:         domain.StatusAccepted,
			MealDate:       time.Date(2026, 7, 10, 0, 0, 0, 0, moscow),
			CreatedAt:      created.UTC(),
		},
		movable: true,
	}
	api := &fakeAPI{}
	r := yesterdayRouter(api, repo, moscow)

	r.handleYesterday(context.Background(), replyTo(42, 1))
	first := repo.gotDate
	r.handleYesterday(context.Background(), replyTo(42, 1))

	want := time.Date(2026, 7, 9, 0, 0, 0, 0, moscow)
	if !first.Equal(want) {
		t.Fatalf("first date = %s, want %s", first, want)
	}
	if !repo.gotDate.Equal(want) {
		t.Fatalf("repeated /yesterday drifted to %s, want %s", repo.gotDate, want)
	}
	if repo.gotMeal != "m1" || repo.gotUser != 42 {
		t.Fatalf("SetMealDate(%q, %d), want (\"m1\", 42)", repo.gotMeal, repo.gotUser)
	}
	if len(api.sent) != 2 || api.sent[0] != render.YesterdayMoved(want) {
		t.Fatalf("expected moved confirmation, got %v", api.sent)
	}
}
