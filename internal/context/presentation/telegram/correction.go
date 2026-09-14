package telegram

import (
	"context"
	"errors"

	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) handleCorrection(ctx context.Context, msg tg.Message, text string) bool {
	entry, revision, err := r.actions.FindAwaitingCorrection(ctx, msg.From.ID, msg.Chat.ID)
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
	if !revision.CanCorrect() {
		if ok, err := r.actions.CancelEdit(ctx, entry.ID, msg.From.ID); err == nil && ok {
			r.pipeline.ShowResult(ctx, entry, revision.Analysis())
		}
		_, _ = r.notify(ctx, msg, render.CorrectionLimit(domain.MaxCorrectionsPerEntry), nil)
		return true
	}
	if backlog := r.mealBacklog(ctx, entry.ChatID); backlog > 0 {
		r.messenger.Queued(ctx, entry, backlog)
	}
	if err := r.mealJobs.EnqueueCorrection(ctx, entry.ID, entry.ChatID, text); err != nil {
		r.log.Error("enqueue meal correction failed", "error", err, "meal_entry_id", entry.ID)
		r.messenger.Fail(ctx, entry, domain.FailReanalyze)
	}
	return true
}
