package telegram

import (
	"context"
	"testing"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/presentation/telegram/render"
)

func TestPrivateResultReplacesStatusMessage(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	binder := &recordingBinder{}
	messenger := NewMealMessenger(api, binder, discardLog())
	entry := domain.MealEntry{ID: "m1", ChatID: 42, TelegramUserID: 42, SourceMessageID: 7, BotMessageID: 8}

	messenger.Result(context.Background(), entry, domain.MealResultView{})

	if len(api.rich) != 1 || api.rich[0].Ephemeral != nil || api.rich[0].ReplyParameters == nil || api.rich[0].ReplyParameters.MessageID != 7 {
		t.Fatalf("the private card must be a new reply to the food message, got %+v", api.rich)
	}
	if len(api.deleted) != 1 || api.deleted[0] != 8 {
		t.Fatalf("the status message should be removed after the card is sent, got %v", api.deleted)
	}
	if binder.bot == 0 || binder.bot == 8 {
		t.Fatalf("the new card should become the meal's bot message, got %d", binder.bot)
	}
}

func TestPrivateFailureReplacesStatusMessage(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	messenger := NewMealMessenger(api, &recordingBinder{}, discardLog())
	entry := domain.MealEntry{ID: "m1", ChatID: 42, TelegramUserID: 42, SourceMessageID: 7, BotMessageID: 8}

	messenger.Fail(context.Background(), entry, domain.FailAnalyze)

	if len(api.sent) != 1 || api.sent[0] != render.MealFailureText(domain.FailAnalyze) || len(api.deleted) != 1 || api.deleted[0] != 8 {
		t.Fatalf("the failure should be a new message replacing the status, got sent %v deleted %v", api.sent, api.deleted)
	}
}

func TestGroupResultEditsEphemeralCard(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	messenger := NewMealMessenger(api, &recordingBinder{}, discardLog())
	entry := domain.MealEntry{ID: "m1", ChatID: -100, TelegramUserID: 42, SourceMessageID: 7, EphemeralMessageID: 5}

	messenger.Result(context.Background(), entry, domain.MealResultView{})

	if len(api.rich) != 0 || len(api.sent) != 0 || len(api.deleted) != 0 {
		t.Fatalf("in a group the ephemeral card is edited in place, got rich %v sent %v deleted %v", api.rich, api.sent, api.deleted)
	}
}
