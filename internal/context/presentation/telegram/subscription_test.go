package telegram

import (
	"context"
	"errors"
	"testing"

	"ojirun/internal/context/application/service"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func brokenSubscription() *service.Subscription {
	broken := service.ChatMemberStatusFunc(func(context.Context, any, int64) (string, error) {
		return "", errors.New("telegram is down")
	})
	return service.NewSubscription(broken, "@ojirun", discardLog())
}

func TestSubscriptionCheckFailureBlocksMessages(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := &Router{
		api:          api,
		subscription: brokenSubscription(),
		channel:      Channel{Label: "@ojirun", URL: "https://t.me/ojirun"},
		log:          discardLog(),
	}

	r.handlePrivateMessage(context.Background(), privateMessage("/stats"))

	if len(api.sent) != 1 || api.sent[0] != render.SubscriptionCheckFailed() {
		t.Fatalf("a failed check must refuse with an explanation, got %v", api.sent)
	}
	if api.opts[0] != nil && api.opts[0].ReplyMarkup != nil {
		t.Fatalf("a system failure is not a subscribe prompt, got %#v", api.opts[0].ReplyMarkup)
	}
}

func TestSubscriptionCheckFailureBlocksCallbacks(t *testing.T) {
	api := &answeringAPI{recordingAPI: &recordingAPI{fakeAPI: &fakeAPI{}}}
	r := &Router{
		api:          api,
		subscription: brokenSubscription(),
		channel:      Channel{Label: "@ojirun", URL: "https://t.me/ojirun"},
		log:          discardLog(),
	}
	cb := tg.CallbackQuery{
		ID:      "cb",
		From:    tg.User{ID: 42},
		Data:    tg.EncodeMealCallback(tg.ActionAccept, "m1"),
		Message: &tg.Message{MessageID: 3, Chat: tg.Chat{ID: 42, Type: "private"}},
	}

	r.handleCallback(context.Background(), cb)

	if len(api.alerts) != 1 || api.alerts[0] != render.SubscriptionCheckFailed() {
		t.Fatalf("expected a failure alert, got %v", api.alerts)
	}

	cb.Data = tg.SubscriptionCheckCallback
	r.handleCallback(context.Background(), cb)

	if len(api.alerts) != 2 || api.alerts[1] != render.SubscriptionCheckFailed() {
		t.Fatalf("the re-check button must report the failure too, got %v", api.alerts)
	}
}
