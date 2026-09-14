package telegram

import (
	"context"
	"errors"
	"strings"
	"time"

	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) routeMeal(ctx context.Context, msg tg.Message, command, args, text string) bool {
	hasPhoto := len(msg.Photo) > 0
	switch {
	case command == "/add" && args == "" && !hasPhoto:
		r.startMealInput(ctx, msg)
	case command == "/add":
		r.takeMealInput(ctx, msg)
		r.handleAdd(ctx, msg, args)
	case command == "/readd":
		r.handleReadd(ctx, msg)
	case command == "/yesterday":
		r.handleYesterday(ctx, msg)
	case hasPhoto:
		r.takeMealInput(ctx, msg)
		r.startMeal(ctx, msg, text, msg.Chat.Type != "private")
	case command == "" && text != "" && r.takeMealInput(ctx, msg):
		r.handleAdd(ctx, msg, text)
	default:
		return false
	}
	return true
}

func (r *Router) handleAdd(ctx context.Context, msg tg.Message, description string) {
	r.startMeal(ctx, msg, description, false)
}

func (r *Router) handleReadd(ctx context.Context, msg tg.Message) {
	original := msg.ReplyToMessage
	if original == nil || original.From == nil || original.From.IsBot {
		_, _ = r.notify(ctx, msg, render.ReaddNeedReply(), nil)
		return
	}
	_, description := splitCommand(original.CommandText())
	r.startMeal(ctx, *original, description, false)
}

func (r *Router) handleYesterday(ctx context.Context, msg tg.Message) {
	target := msg.ReplyToMessage
	if target == nil {
		_, _ = r.notify(ctx, msg, render.YesterdayNeedReply(), nil)
		return
	}
	if !r.requireSubscriber(ctx, msg) {
		return
	}

	entry, err := r.actions.FindByMessage(ctx, msg.Chat.ID, target.MessageID)
	if err != nil {
		if !errors.Is(err, domain.ErrMealNotFound) {
			r.log.Error("find meal by message failed", "error", err, "chat_id", msg.Chat.ID)
		}
		_, _ = r.notify(ctx, msg, render.YesterdayNoEntry(), nil)
		return
	}
	if entry.TelegramUserID != msg.From.ID {
		_, _ = r.notify(ctx, msg, render.YesterdayNotAuthor(), nil)
		return
	}

	date := dayBefore(entry.CreatedAt, r.loc)
	moved, err := r.actions.SetMealDate(ctx, entry.ID, msg.From.ID, date)
	if err != nil {
		r.log.Error("move meal to yesterday failed", "error", err, "meal_id", entry.ID)
		return
	}
	if !moved {
		_, _ = r.notify(ctx, msg, render.YesterdayNotMovable(), nil)
		return
	}
	_, _ = r.notify(ctx, msg, render.YesterdayMoved(date), nil)
}

func dayBefore(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	day := t.In(loc)
	return time.Date(day.Year(), day.Month(), day.Day()-1, 0, 0, 0, 0, loc)
}

func (r *Router) startMeal(ctx context.Context, source tg.Message, description string, quietIfUnregistered bool) {
	author, err := r.addMeal.Authorize(ctx, source.From.ID)
	if err != nil {
		if !quietIfUnregistered || !errors.Is(err, domain.ErrNotFound) {
			r.replyAddRejection(ctx, source, err)
		}
		return
	}
	if !r.requireSubscriber(ctx, source) {
		return
	}
	if strings.TrimSpace(description) == "" && len(source.Photo) == 0 {
		_, _ = r.notify(ctx, source, render.MealNeedContent(), nil)
		return
	}

	sourceMessageID := source.MessageID
	if source.EphemeralMessageID != 0 {
		sourceMessageID = 0
	}
	entry, err := r.addMeal.CreateEntry(ctx, author, usecase.AddMealInput{
		ChatID:          source.Chat.ID,
		SourceMessageID: sourceMessageID,
		MealDate:        time.Now().In(r.loc),
	})
	if errors.Is(err, domain.ErrDailyMealLimit) {
		_, _ = r.notify(ctx, source, render.MealDailyLimit(r.addMeal.DailyLimit()), nil)
		return
	}
	if err != nil {
		r.log.Error("create meal entry failed", "error", err)
		return
	}
	entry.TelegramUserID = source.From.ID

	status := render.MealStageStatus(domain.StageReceived)
	if backlog := r.mealBacklog(ctx); backlog > 0 {
		status = render.MealQueued(backlog)
	}
	entry = r.messenger.sendStatusReply(ctx, entry, status)

	var photo domain.PhotoRef
	if best, ok := source.BestPhoto(); ok {
		photo = domain.PhotoRef{FileID: best.FileID, FileUniqueID: best.FileUniqueID}
	}
	if err := r.mealJobs.EnqueueAnalysis(ctx, entry.ID, photo, description); err != nil {
		r.log.Error("enqueue meal analysis failed", "error", err, "meal_entry_id", entry.ID)
		r.messenger.Fail(ctx, entry, domain.FailAnalyze)
	}
}

func (r *Router) mealBacklog(ctx context.Context) int {
	backlog, err := r.mealJobs.Backlog(ctx)
	if err != nil {
		r.log.Warn("read meal queue backlog failed", "error", err)
		return 0
	}
	return backlog
}

func (r *Router) replyAddRejection(ctx context.Context, msg tg.Message, err error) {
	switch {
	case !errors.Is(err, domain.ErrNotFound):
		r.log.Error("authorize meal author failed", "error", err)
	case msg.Chat.Type == "private":
		_, _ = r.notify(ctx, msg, render.NeedRegisterPrivate(), nil)
	case errors.Is(err, domain.ErrProfileNotFound):
		_, _ = r.notify(ctx, msg, render.FinishRegister(r.botUsername), htmlOptions())
	default:
		_, _ = r.notify(ctx, msg, render.NeedRegister(r.botUsername), htmlOptions())
	}
}
