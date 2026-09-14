package telegram

import (
	"context"
	"testing"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/presentation/telegram/render"
)

type draftingAPI struct {
	*recordingAPI
	drafts []string
}

func (a *draftingAPI) SendRichHTMLDraft(_ context.Context, _ any, _ int64, html string) error {
	a.drafts = append(a.drafts, html)
	return nil
}

func TestPrivateProgressIsDraftOnly(t *testing.T) {
	api := &draftingAPI{recordingAPI: &recordingAPI{fakeAPI: &fakeAPI{}}}
	messenger := NewMealMessenger(api, &recordingBinder{}, discardLog())
	entry := domain.MealEntry{ID: "m1", ChatID: 42, TelegramUserID: 42, SourceMessageID: 7}

	entry = messenger.Received(context.Background(), entry, 0)
	messenger.Stage(context.Background(), entry, domain.StageAnalyzing)
	messenger.Queued(context.Background(), entry, 2)

	if len(api.sent) != 0 || len(api.rich) != 0 || entry.BotMessageID != 0 {
		t.Fatalf("private progress must not create messages, got sent %v rich %v entry %+v", api.sent, api.rich, entry)
	}
	if len(api.drafts) != 3 || api.drafts[0] != render.MealReceivedDraft(0) || api.drafts[1] != render.MealStageDraft(domain.StageAnalyzing) || api.drafts[2] != render.MealReceivedDraft(2) {
		t.Fatalf("expected received, stage and queue drafts, got %v", api.drafts)
	}
}

func TestGroupProgressIsEphemeralStatus(t *testing.T) {
	api := &draftingAPI{recordingAPI: &recordingAPI{fakeAPI: &fakeAPI{}}}
	messenger := NewMealMessenger(api, &recordingBinder{}, discardLog())
	entry := domain.MealEntry{ID: "m1", ChatID: -100, TelegramUserID: 42, SourceMessageID: 7}

	entry = messenger.Received(context.Background(), entry, 3)

	if len(api.drafts) != 0 || len(api.sent) != 1 || api.sent[0] != render.MealQueued(3) || api.opts[0].Ephemeral == nil || entry.EphemeralMessageID == 0 {
		t.Fatalf("group progress must be an ephemeral status message, got drafts %v sent %v entry %+v", api.drafts, api.sent, entry)
	}
}

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
