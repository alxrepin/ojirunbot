package telegram

import (
	"context"
	"testing"

	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type recordingBinder struct {
	bot       int64
	ephemeral int64
}

func (b *recordingBinder) SetBotMessage(_ context.Context, _ string, id int64) error {
	b.bot = id
	return nil
}

func (b *recordingBinder) SetEphemeralMessage(_ context.Context, _ string, id int64) error {
	b.ephemeral = id
	return nil
}

func TestGroupMealStatusIsEphemeral(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := newInputRouter(api, &inputSessions{})
	command := groupMessage()
	command.Text = "/add гречка с курицей"

	r.handleGroupMessage(context.Background(), command)

	if len(api.opts) != 1 || api.sent[0] != render.MealStageStatus(domain.StageReceived) {
		t.Fatalf("expected one status message, got %v", api.sent)
	}
	opts := api.opts[0]
	if opts.Ephemeral == nil || opts.Ephemeral.ReceiverUserID != 42 {
		t.Fatalf("status must be ephemeral for the author, got %+v", opts)
	}
	if opts.ReplyParameters != nil {
		t.Fatalf("ephemeral status must not quote the food message, got %+v", opts.ReplyParameters)
	}
}

func TestGroupNoticesAreEphemeral(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := &Router{api: api, log: discardLog()}
	command := groupMessage()
	command.Text = "/profile"

	r.handleGroupMessage(context.Background(), command)

	if len(api.sent) != 1 || api.sent[0] != render.NeedPrivateMessage() || api.opts[0].Ephemeral == nil {
		t.Fatalf("expected an ephemeral notice, got %v %+v", api.sent, api.opts)
	}
}

func TestAcceptedMealIsPublishedToGroup(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	binder := &recordingBinder{}
	messenger := NewMealMessenger(api, binder, discardLog())
	entry := domain.MealEntry{ID: "m1", ChatID: -100, TelegramUserID: 42, SourceMessageID: 7, EphemeralMessageID: 5}

	messenger.Accepted(context.Background(), entry, domain.MealResultView{})

	if len(api.rich) != 1 || api.rich[0].Ephemeral != nil || api.rich[0].ReplyParameters.MessageID != 7 {
		t.Fatalf("accepted meal must be a public reply to the food message, got %+v", api.rich)
	}
	if len(api.deleted) != 1 || api.deleted[0] != 5 {
		t.Fatalf("the author's ephemeral card should be removed, got %v", api.deleted)
	}
	if binder.bot == 0 {
		t.Fatal("the public card should become the meal's bot message")
	}
}

func TestPrivateMenu(t *testing.T) {
	if command, ok := menuCommand("📊 Статистика"); !ok || command != "/stats" {
		t.Fatalf("menuCommand = %q, %v", command, ok)
	}
	if _, ok := menuCommand("гречка"); ok {
		t.Fatal("ordinary text is not a menu button")
	}

	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := &Router{api: api, log: discardLog()}
	r.handlePrivateMessage(context.Background(), privateMessage("💡 Как это работает?"))

	if len(api.sent) != 1 || api.sent[0] != render.HowItWorks() {
		t.Fatalf("expected the help text, got %v", api.sent)
	}
	keyboard, ok := api.opts[0].ReplyMarkup.(tg.ReplyKeyboardMarkup)
	if !ok || !keyboard.IsPersistent || keyboard.Keyboard[0][0].Style != tg.ButtonStyleSuccess {
		t.Fatalf("expected the persistent menu with a green first button, got %#v", api.opts[0].ReplyMarkup)
	}
	if api.opts[0].Ephemeral != nil {
		t.Fatal("private chat messages are regular messages")
	}
}
