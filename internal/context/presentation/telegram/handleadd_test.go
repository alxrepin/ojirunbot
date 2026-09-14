package telegram

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"ojirun/internal/context/application/service"
	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type fakeAPI struct {
	sent []string
}

func (f *fakeAPI) SendMessage(_ context.Context, _ any, text string, _ *tg.SendOptions) (tg.Message, error) {
	f.sent = append(f.sent, text)
	return tg.Message{}, nil
}

func (f *fakeAPI) SendRichMarkdown(context.Context, any, string, *tg.SendOptions) (tg.Message, error) {
	return tg.Message{}, nil
}
func (f *fakeAPI) EditMessageText(context.Context, int64, int64, string, *tg.SendOptions) error {
	return nil
}
func (f *fakeAPI) EditMessageCaption(context.Context, int64, int64, string, *tg.SendOptions) error {
	return nil
}
func (f *fakeAPI) EditRichMarkdown(context.Context, int64, int64, string, *tg.SendOptions) error {
	return nil
}
func (f *fakeAPI) EditMessageReplyMarkup(context.Context, int64, int64, any) error { return nil }
func (f *fakeAPI) SendRichHTMLDraft(context.Context, any, int64, string) error     { return nil }
func (f *fakeAPI) DeleteMessage(context.Context, int64, int64) error               { return nil }
func (f *fakeAPI) EditEphemeralMessageText(context.Context, int64, int64, int64, string, *tg.SendOptions) error {
	return nil
}
func (f *fakeAPI) EditEphemeralRichMarkdown(context.Context, int64, int64, int64, string, *tg.SendOptions) error {
	return nil
}
func (f *fakeAPI) DeleteEphemeralMessage(context.Context, int64, int64, int64) error { return nil }
func (f *fakeAPI) AnswerCallbackQuery(context.Context, string, string, bool) error   { return nil }

type fakeUsers struct {
	user domain.User
	err  error
}

func (f fakeUsers) GetByTelegramID(context.Context, int64) (domain.User, error) {
	return f.user, f.err
}

type fakeProfiles struct {
	profile domain.Profile
	err     error
}

func (f fakeProfiles) GetByUserID(context.Context, string) (domain.Profile, error) {
	return f.profile, f.err
}

type fakeMeals struct {
	entry   domain.MealEntry
	err     error
	created int
}

func (f fakeMeals) CreateEntryWithinLimit(context.Context, string, int64, int64, time.Time, time.Time, domain.MealLimits) (domain.MealEntry, error) {
	return f.entry, f.err
}

func (f fakeMeals) CountCreatedSince(context.Context, string, time.Time) (int, error) {
	return f.created, nil
}

func newRouter(api API, addMeal *usecase.AddMeal) *Router {
	return &Router{api: api, addMeal: addMeal, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func groupMessage() tg.Message {
	return tg.Message{
		MessageID: 1,
		From:      &tg.User{ID: 42},
		Chat:      tg.Chat{ID: -100, Type: "supergroup"},
	}
}

func TestHandleAddRejectsEmpty(t *testing.T) {
	api := &fakeAPI{}
	addMeal := usecase.NewAddMeal(
		fakeUsers{user: domain.User{ID: "u1", TelegramUserID: 42}},
		fakeProfiles{profile: domain.Profile{UserID: "u1"}},
		fakeMeals{},
		domain.MealLimits{}, time.UTC,
	)
	r := newRouter(api, addMeal)

	r.handleAdd(context.Background(), groupMessage(), "   ")

	if len(api.sent) != 1 || api.sent[0] != render.MealNeedContent() {
		t.Fatalf("expected need-content reply, got %v", api.sent)
	}
}

type recordingUsers struct {
	user  domain.User
	err   error
	gotID int64
}

func (f *recordingUsers) GetByTelegramID(_ context.Context, id int64) (domain.User, error) {
	f.gotID = id
	return f.user, f.err
}

func TestHandleReaddNeedsReply(t *testing.T) {
	cases := map[string]tg.Message{
		"no reply": {MessageID: 5, From: &tg.User{ID: 99}, Chat: tg.Chat{ID: 100, Type: "supergroup"}},
		"reply to bot": {
			MessageID:      5,
			From:           &tg.User{ID: 99},
			Chat:           tg.Chat{ID: 100, Type: "supergroup"},
			ReplyToMessage: &tg.Message{MessageID: 1, From: &tg.User{ID: 7, IsBot: true}},
		},
	}
	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			api := &fakeAPI{}
			users := &recordingUsers{user: domain.User{ID: "u1"}}
			addMeal := usecase.NewAddMeal(users, fakeProfiles{}, fakeMeals{}, domain.MealLimits{}, time.UTC)
			r := newRouter(api, addMeal)

			r.handleReadd(context.Background(), msg)

			if len(api.sent) != 1 || api.sent[0] != render.ReaddNeedReply() {
				t.Fatalf("expected need-reply notice, got %v", api.sent)
			}
			if users.gotID != 0 {
				t.Fatalf("authorize should not run, got id %d", users.gotID)
			}
		})
	}
}

func readdMessage(from int64) tg.Message {
	return tg.Message{
		MessageID: 5,
		From:      &tg.User{ID: from},
		Chat:      tg.Chat{ID: -100, Type: "supergroup"},
		ReplyToMessage: &tg.Message{
			MessageID: 2,
			From:      &tg.User{ID: 42},
			Chat:      tg.Chat{ID: -100, Type: "supergroup"},
			Photo:     []tg.PhotoSize{{FileID: "f", FileUniqueID: "u", Width: 1, Height: 1}},
		},
	}
}

func TestHandleReaddOnlyByAuthor(t *testing.T) {
	api := &fakeAPI{}
	users := &recordingUsers{user: domain.User{ID: "u1"}}
	r := newRouter(api, usecase.NewAddMeal(users, fakeProfiles{}, fakeMeals{}, domain.MealLimits{}, time.UTC))
	r.actions = usecase.NewMealActions(&fakeMealActions{})

	r.handleReadd(context.Background(), readdMessage(99))

	if len(api.sent) != 1 || api.sent[0] != render.ReaddNotAuthor() {
		t.Fatalf("a stranger's /readd must be refused, got %v", api.sent)
	}
	if users.gotID != 0 {
		t.Fatalf("authorize should not run for a stranger, got id %d", users.gotID)
	}
}

func TestHandleReaddAuthorshipFromReplay(t *testing.T) {
	api := &fakeAPI{}
	users := &recordingUsers{err: domain.ErrUserNotFound}
	r := newRouter(api, usecase.NewAddMeal(users, fakeProfiles{}, fakeMeals{}, domain.MealLimits{}, time.UTC))
	r.actions = usecase.NewMealActions(&fakeMealActions{})

	r.handleReadd(context.Background(), readdMessage(42))

	if users.gotID != 42 {
		t.Fatalf("authorized id = %d, want 42 (replayed author)", users.gotID)
	}
	if len(api.sent) != 1 || api.sent[0] != render.NeedRegister(r.botUsername) {
		t.Fatalf("expected register hint for the replayed author, got %v", api.sent)
	}
}

func TestHandleReaddWaitsForProcessing(t *testing.T) {
	api := &fakeAPI{}
	users := &recordingUsers{user: domain.User{ID: "u1"}}
	r := newRouter(api, usecase.NewAddMeal(users, fakeProfiles{}, fakeMeals{}, domain.MealLimits{}, time.UTC))
	r.actions = usecase.NewMealActions(&fakeMealActions{entry: domain.MealEntry{
		ID: "m1", ChatID: -100, SourceMessageID: 2, TelegramUserID: 42, Status: domain.StatusAnalyzing,
	}})

	r.handleReadd(context.Background(), readdMessage(42))

	if len(api.sent) != 1 || api.sent[0] != render.ReaddInProgress() || users.gotID != 0 {
		t.Fatalf("an entry in flight must not be replaced, got %v (authorized %d)", api.sent, users.gotID)
	}
}

func TestHandleReaddReplacesPreviousEntry(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := newInputRouter(api, &inputSessions{})
	queue := &fakeJobQueue{}
	r.mealJobs = service.NewMealJobs(queue, nil, nil, nil)
	previous := &fakeMealActions{entry: domain.MealEntry{
		ID: "m1", ChatID: -100, SourceMessageID: 2, TelegramUserID: 42, Status: domain.StatusPending, EphemeralMessageID: 9,
	}}
	r.actions = usecase.NewMealActions(previous)

	r.handleReadd(context.Background(), readdMessage(42))

	if len(previous.deleted) != 1 || previous.deleted[0] != "m1" {
		t.Fatalf("the previous entry should be deleted, got %v", previous.deleted)
	}
	if len(api.deleted) != 1 || api.deleted[0] != 9 {
		t.Fatalf("the previous card should be removed, got %v", api.deleted)
	}
	if last := api.sent[len(api.sent)-1]; last != render.MealStageStatus(domain.StageReceived) {
		t.Fatalf("expected a fresh status reply, got %v", api.sent)
	}
	if len(queue.enqueued) != 1 || queue.enqueued[0] != domain.JobMealAnalysis {
		t.Fatalf("the analysis should be queued again, got %v", queue.enqueued)
	}
}

func TestHandleAddUnregisteredBeforeEmpty(t *testing.T) {
	api := &fakeAPI{}
	addMeal := usecase.NewAddMeal(
		fakeUsers{err: domain.ErrUserNotFound},
		fakeProfiles{},
		fakeMeals{},
		domain.MealLimits{}, time.UTC,
	)
	r := newRouter(api, addMeal)

	r.handleAdd(context.Background(), groupMessage(), "   ")

	if len(api.sent) != 1 || api.sent[0] != render.NeedRegister(r.botUsername) {
		t.Fatalf("expected register hint, got %v", api.sent)
	}
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
