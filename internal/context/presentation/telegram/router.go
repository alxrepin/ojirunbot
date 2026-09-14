package telegram

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"ojirun/internal/context/application/service"
	"ojirun/internal/context/application/usecase"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type Router struct {
	api          API
	messenger    *MealMessenger
	addMeal      *usecase.AddMeal
	actions      *usecase.MealActions
	profiles     *usecase.GetProfile
	stats        *usecase.GetStats
	registration *service.Registration
	settings     *service.Settings
	mealInput    *service.MealInput
	mealJobs     *service.MealJobs
	pipeline     *service.MealPipeline
	subscription *service.Subscription

	channel     Channel
	botUsername string
	loc         *time.Location
	log         *slog.Logger
}

type Channel struct {
	Label string
	URL   string
}

type Deps struct {
	API          API
	Messenger    *MealMessenger
	AddMeal      *usecase.AddMeal
	Actions      *usecase.MealActions
	Profiles     *usecase.GetProfile
	Stats        *usecase.GetStats
	Registration *service.Registration
	Settings     *service.Settings
	MealInput    *service.MealInput
	MealJobs     *service.MealJobs
	Pipeline     *service.MealPipeline
	Subscription *service.Subscription

	Channel     Channel
	BotUsername string
	Loc         *time.Location
	Log         *slog.Logger
}

func NewRouter(d Deps) *Router {
	return &Router{
		api:          d.API,
		messenger:    d.Messenger,
		addMeal:      d.AddMeal,
		actions:      d.Actions,
		profiles:     d.Profiles,
		stats:        d.Stats,
		registration: d.Registration,
		settings:     d.Settings,
		mealInput:    d.MealInput,
		mealJobs:     d.MealJobs,
		pipeline:     d.Pipeline,
		subscription: d.Subscription,
		channel:      d.Channel,
		botUsername:  d.BotUsername,
		loc:          d.Loc,
		log:          d.Log,
	}
}

func (r *Router) HandleUpdate(ctx context.Context, update tg.Update) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("panic while handling update", "panic", rec)
		}
	}()

	if update.CallbackQuery != nil {
		r.handleCallback(ctx, *update.CallbackQuery)
		return
	}
	msg := update.Message
	if msg == nil || msg.From == nil || msg.From.IsBot {
		return
	}

	switch msg.Chat.Type {
	case "private":
		r.handlePrivateMessage(ctx, *msg)
	case "group", "supergroup":
		r.handleGroupMessage(ctx, *msg)
	}
}

func (r *Router) handlePrivateMessage(ctx context.Context, msg tg.Message) {
	text := strings.TrimSpace(msg.CommandText())
	if command, ok := menuCommand(text); ok {
		text = command
	}
	command, args, foreign := r.parseCommand(text)
	if foreign {
		return
	}
	if !r.requireSubscriber(ctx, msg) {
		return
	}

	switch command {
	case "/start":
		r.startRegistration(ctx, msg)
		return
	case "/profile":
		r.showProfile(ctx, msg)
		return
	case "/stats":
		r.handleStats(ctx, msg, "")
		return
	case "/settings":
		r.startSettings(ctx, msg)
		return
	case "/help":
		r.showHelp(ctx, msg)
		return
	}
	if r.routeMeal(ctx, msg, command, args, text) {
		return
	}

	if session, err := r.settings.ActiveSession(ctx, msg.From.ID, msg.Chat.ID); err == nil {
		r.advanceSettings(ctx, msg, session, text)
		return
	}
	if session, err := r.registration.ActiveSession(ctx, msg.From.ID, msg.Chat.ID); err == nil {
		r.advanceRegistration(ctx, msg, session, text)
		return
	}
	if command == "" && text != "" && r.handleCorrection(ctx, msg, text) {
		return
	}
	_, _ = r.api.SendMessage(ctx, msg.Chat.ID, render.PrivateHelp(), withMenu(nil))
}

func (r *Router) handleGroupMessage(ctx context.Context, msg tg.Message) {
	text := strings.TrimSpace(msg.CommandText())
	command, args, foreign := r.parseCommand(text)
	if foreign {
		return
	}

	switch command {
	case "/start", "/profile", "/settings":
		_, _ = r.notify(ctx, msg, render.NeedPrivateMessage(), nil)
		return
	case "/help":
		r.showHelp(ctx, msg)
		return
	case "/stats":
		r.handleStats(ctx, msg, args)
		return
	}
	if r.routeMeal(ctx, msg, command, args, text) {
		return
	}
	if command == "" && text != "" {
		r.handleCorrection(ctx, msg, text)
	}
}

func (r *Router) parseCommand(text string) (command, args string, foreign bool) {
	command, args = splitCommand(text)
	if command == "" {
		return "", args, false
	}
	head, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	if _, mention, ok := strings.Cut(head, "@"); ok && r.botUsername != "" && !strings.EqualFold(mention, r.botUsername) {
		return "", "", true
	}
	return command, args, false
}

func splitCommand(text string) (command, args string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	head, tail, _ := strings.Cut(text, " ")
	if at := strings.IndexByte(head, '@'); at >= 0 {
		head = head[:at]
	}
	return strings.ToLower(head), strings.TrimSpace(tail)
}
