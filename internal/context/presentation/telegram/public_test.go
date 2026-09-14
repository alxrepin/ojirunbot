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

type fakeJobQueue struct {
	enqueued []string
}

func (q *fakeJobQueue) Enqueue(_ context.Context, _, kind string, _ any, _ string) error {
	q.enqueued = append(q.enqueued, kind)
	return nil
}

func (q *fakeJobQueue) Claim(context.Context, string, time.Duration) (domain.Job, bool, error) {
	return domain.Job{}, false, nil
}
func (q *fakeJobQueue) Complete(context.Context, string) error                     { return nil }
func (q *fakeJobQueue) Retry(context.Context, string, time.Duration, string) error { return nil }
func (q *fakeJobQueue) Fail(context.Context, string, string) error                 { return nil }
func (q *fakeJobQueue) Backlog(context.Context, string) (int, error)               { return 0, nil }

func privateMessage(text string) tg.Message {
	return tg.Message{MessageID: 1, From: &tg.User{ID: 42}, Chat: tg.Chat{ID: 42, Type: "private"}, Text: text}
}

func TestPrivateChatRequiresSubscription(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	notSubscribed := service.ChatMemberStatusFunc(func(context.Context, any, int64) (string, error) { return "left", nil })
	r := &Router{
		api:          api,
		subscription: service.NewSubscription(notSubscribed, "@ojirun", discardLog()),
		channel:      Channel{Label: "@ojirun", URL: "https://t.me/ojirun"},
		log:          discardLog(),
	}

	r.handlePrivateMessage(context.Background(), privateMessage("/stats"))

	if len(api.sent) != 1 || api.sent[0] != render.SubscribeRequiredPrivate("@ojirun", "https://t.me/ojirun") {
		t.Fatalf("expected the subscribe prompt, got %v", api.sent)
	}
	keyboard := api.opts[0].ReplyMarkup.(tg.InlineKeyboardMarkup).InlineKeyboard
	if keyboard[0][0].URL != "https://t.me/ojirun" || keyboard[1][0].CallbackData != tg.SubscriptionCheckCallback {
		t.Fatalf("expected subscribe link and re-check buttons, got %#v", keyboard)
	}
}

func TestGroupPhotoFromStrangerIsIgnored(t *testing.T) {
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := &Router{
		api:       api,
		addMeal:   usecase.NewAddMeal(fakeUsers{err: domain.ErrUserNotFound}, fakeProfiles{}, fakeMeals{}, 0, time.UTC),
		mealInput: service.NewMealInput(&inputSessions{}),
		log:       discardLog(),
	}
	photo := groupMessage()
	photo.Photo = []tg.PhotoSize{{FileID: "f", FileUniqueID: "u", Width: 1, Height: 1}}

	r.handleGroupMessage(context.Background(), photo)

	if len(api.sent) != 0 {
		t.Fatalf("a stranger's photo must not get a reply, got %v", api.sent)
	}
}

func TestPrivateChatLogsMealsThroughQueue(t *testing.T) {
	ctx := context.Background()
	api := &recordingAPI{fakeAPI: &fakeAPI{}}
	r := newInputRouter(api, &inputSessions{})
	queue := &fakeJobQueue{}
	r.mealJobs = service.NewMealJobs(queue, nil, nil, nil)

	r.handlePrivateMessage(ctx, privateMessage("/add"))
	if len(api.sent) != 1 || api.sent[0] != render.MealInputPrompt() {
		t.Fatalf("/add in the private chat should start input mode, got %v", api.sent)
	}

	food := privateMessage("омлет из двух яиц")
	food.MessageID = 2
	r.handlePrivateMessage(ctx, food)

	if last := api.sent[len(api.sent)-1]; last != render.MealStageStatus(domain.StageReceived) {
		t.Fatalf("expected the meal status reply, got %v", api.sent)
	}
	if len(queue.enqueued) != 1 || queue.enqueued[0] != domain.JobMealAnalysis {
		t.Fatalf("the analysis should be queued, got %v", queue.enqueued)
	}
}
