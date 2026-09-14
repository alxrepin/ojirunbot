package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type recordingAPI struct {
	*fakeAPI
	opts    []*tg.SendOptions
	rich    []*tg.SendOptions
	deleted []int64
}

func (a *recordingAPI) SendMessage(ctx context.Context, chatID any, text string, opts *tg.SendOptions) (tg.Message, error) {
	_, _ = a.fakeAPI.SendMessage(ctx, chatID, text, opts)
	a.opts = append(a.opts, opts)
	return sentMessage(len(a.sent), opts), nil
}

func (a *recordingAPI) SendRichMarkdown(_ context.Context, _ any, _ string, opts *tg.SendOptions) (tg.Message, error) {
	a.rich = append(a.rich, opts)
	return sentMessage(100+len(a.rich), opts), nil
}

func (a *recordingAPI) DeleteMessage(_ context.Context, _ int64, messageID int64) error {
	a.deleted = append(a.deleted, messageID)
	return nil
}

func (a *recordingAPI) DeleteEphemeralMessage(_ context.Context, _, _ int64, ephemeralMessageID int64) error {
	a.deleted = append(a.deleted, ephemeralMessageID)
	return nil
}

func sentMessage(id int, opts *tg.SendOptions) tg.Message {
	if opts != nil && opts.Ephemeral != nil {
		return tg.Message{EphemeralMessageID: int64(id)}
	}
	return tg.Message{MessageID: int64(id)}
}

type inputSessions struct {
	rows []*domain.Session
}

func (s *inputSessions) Start(_ context.Context, kind domain.SessionKind, telegramUserID, chatID int64, step string, payload any) (string, error) {
	for _, row := range s.rows {
		if row.TelegramUserID == telegramUserID && row.ChatID == chatID {
			row.Status = "cancelled"
		}
	}
	raw, _ := json.Marshal(payload)
	row := &domain.Session{ID: fmt.Sprintf("s%d", len(s.rows)+1), Kind: kind, TelegramUserID: telegramUserID, ChatID: chatID, Step: step, PayloadJSON: raw, Status: "active"}
	s.rows = append(s.rows, row)
	return row.ID, nil
}

func (s *inputSessions) GetActive(_ context.Context, kind domain.SessionKind, telegramUserID, chatID int64) (domain.Session, error) {
	for _, row := range s.rows {
		if row.Kind == kind && row.TelegramUserID == telegramUserID && row.ChatID == chatID && row.Status == "active" {
			return *row, nil
		}
	}
	return domain.Session{}, domain.ErrSessionNotFound
}

func (s *inputSessions) SetBotMessage(_ context.Context, kind domain.SessionKind, telegramUserID, chatID, botMessageID int64) error {
	for _, row := range s.rows {
		if row.Kind == kind && row.TelegramUserID == telegramUserID && row.ChatID == chatID && row.Status == "active" {
			row.BotMessageID = botMessageID
		}
	}
	return nil
}

func (s *inputSessions) Advance(context.Context, string, string, string, any) (bool, error) {
	return false, nil
}

func (s *inputSessions) Close(_ context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return s.finish(sessionID, telegramUserID, "completed"), nil
}

func (s *inputSessions) Cancel(_ context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return s.finish(sessionID, telegramUserID, "cancelled"), nil
}

func (s *inputSessions) Complete(context.Context, string, string) error { return nil }

func (s *inputSessions) finish(sessionID string, telegramUserID int64, status string) bool {
	for _, row := range s.rows {
		if row.ID == sessionID && row.TelegramUserID == telegramUserID && row.Status == "active" {
			row.Status = status
			return true
		}
	}
	return false
}

type noopBinder struct{}

func (noopBinder) SetBotMessage(context.Context, string, int64) error       { return nil }
func (noopBinder) SetEphemeralMessage(context.Context, string, int64) error { return nil }

func newInputRouter(api API, sessions *inputSessions) *Router {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	addMeal := usecase.NewAddMeal(
		fakeUsers{user: domain.User{ID: "u1", TelegramUserID: 42}},
		fakeProfiles{profile: domain.Profile{UserID: "u1"}},
		fakeMeals{entry: domain.MealEntry{ID: "m1", ChatID: -100, SourceMessageID: 2}},
		domain.MealLimits{}, time.UTC,
	)
	return &Router{
		api:       api,
		addMeal:   addMeal,
		mealInput: service.NewMealInput(sessions),
		mealJobs:  service.NewMealJobs(&fakeJobQueue{}, nil, nil, nil),
		messenger: NewMealMessenger(api, noopBinder{}, log),
		loc:       time.UTC,
		log:       log,
	}
}

func TestBareAddStartsInputMode(t *testing.T) {
	ctx := context.Background()
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	sessions := &inputSessions{}
	r := newInputRouter(api, sessions)

	command := groupMessage()
	command.Text = "/add"
	r.handleGroupMessage(ctx, command)

	if len(api.sent) != 1 || api.sent[0] != render.MealInputPrompt() {
		t.Fatalf("expected the input prompt, got %v", api.sent)
	}
	keyboard, ok := api.opts[0].ReplyMarkup.(tg.InlineKeyboardMarkup)
	if !ok || len(keyboard.InlineKeyboard) != 1 {
		t.Fatalf("expected a cancel keyboard, got %#v", api.opts[0].ReplyMarkup)
	}
	button := keyboard.InlineKeyboard[0][0]
	sessionID, ok := tg.DecodeMealInputCancel(button.CallbackData)
	session, err := sessions.GetActive(ctx, domain.SessionMealInput, 42, -100)
	if err != nil || !ok || session.ID != sessionID || button.Style != tg.ButtonStyleDanger {
		t.Fatalf("cancel button %#v does not match session %+v (%v)", button, session, err)
	}
	if session.BotMessageID != 1 {
		t.Fatalf("prompt not bound to the session, got %d", session.BotMessageID)
	}
}

func TestInputModeTakesNextText(t *testing.T) {
	ctx := context.Background()
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	sessions := &inputSessions{}
	r := newInputRouter(api, sessions)

	command := groupMessage()
	command.Text = "/add"
	r.handleGroupMessage(ctx, command)

	food := groupMessage()
	food.MessageID = 2
	food.Text = "гречка с курицей"
	r.handleGroupMessage(ctx, food)

	if len(api.deleted) != 1 || api.deleted[0] != 1 {
		t.Fatalf("prompt message 1 should be deleted, got %v", api.deleted)
	}
	if last := api.sent[len(api.sent)-1]; last != render.MealStageStatus(domain.StageReceived) {
		t.Fatalf("expected the meal status reply, got %v", api.sent)
	}
	if _, err := sessions.GetActive(ctx, domain.SessionMealInput, 42, -100); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("input mode should be closed, got %v", err)
	}
}
