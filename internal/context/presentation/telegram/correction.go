package telegram

import (
	"context"
	"errors"

	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) handleCorrection(ctx context.Context, msg tg.Message, text string) bool {
	entry, _, err := r.actions.FindAwaitingCorrection(ctx, msg.From.ID, msg.Chat.ID)
	if errors.Is(err, domain.ErrMealNotFound) {
		return false
	}
	if err != nil {
		r.log.Error("find awaiting correction failed", "error", err)
		return false
	}
	if !r.requireSubscriber(ctx, msg) {
		return true
	}
	if backlog := r.mealBacklog(ctx); backlog > 0 {
		_ = r.messenger.updateMessage(ctx, entry, render.MealQueued(backlog), nil)
	}
	if err := r.mealJobs.EnqueueCorrection(ctx, entry.ID, text); err != nil {
		r.log.Error("enqueue meal correction failed", "error", err, "meal_entry_id", entry.ID)
		r.messenger.Fail(ctx, entry, domain.FailReanalyze)
	}
	return true
}
