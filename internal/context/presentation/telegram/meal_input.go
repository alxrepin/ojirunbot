package telegram

import (
	"context"

	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) startMealInput(ctx context.Context, msg tg.Message) {
	author, err := r.addMeal.Authorize(ctx, msg.From.ID)
	if err != nil {
		r.replyAddRejection(ctx, msg, err)
		return
	}
	if !r.requireSubscriber(ctx, msg) {
		return
	}
	if reached, err := r.addMeal.LimitReached(ctx, author); err != nil {
		r.log.Warn("check daily meal limit failed", "error", err)
	} else if reached {
		_, _ = r.notify(ctx, msg, render.MealDailyLimit(r.addMeal.DailyLimit()), nil)
		return
	}
	sessionID, stalePromptID, err := r.mealInput.Begin(ctx, msg.From.ID, msg.Chat.ID)
	if err != nil {
		r.log.Error("start meal input failed", "error", err)
		return
	}
	r.discard(ctx, senderMessage(msg, stalePromptID))

	prompt, err := r.notify(ctx, msg, render.MealInputPrompt(), &tg.SendOptions{ReplyMarkup: tg.MealInputKeyboard(sessionID)})
	if err != nil {
		r.log.Error("send meal input prompt failed", "error", err)
		return
	}
	if err := r.mealInput.BindPrompt(ctx, msg.From.ID, msg.Chat.ID, prompt.ID); err != nil {
		r.log.Warn("bind meal input prompt failed", "error", err)
	}
}

func (r *Router) takeMealInput(ctx context.Context, msg tg.Message) bool {
	promptID, ok, err := r.mealInput.Claim(ctx, msg.From.ID, msg.Chat.ID)
	if err != nil {
		r.log.Error("claim meal input failed", "error", err)
		return false
	}
	if ok {
		r.discard(ctx, senderMessage(msg, promptID))
	}
	return ok
}

func (r *Router) handleMealInputCancel(ctx context.Context, cb tg.CallbackQuery, sessionID string, answer *callbackAnswer) {
	cancelled, err := r.mealInput.Cancel(ctx, sessionID, cb.From.ID)
	if err != nil {
		r.log.Error("cancel meal input failed", "error", err)
		return
	}
	if !cancelled {
		answer.Alert("Этот ввод уже закрыт или начат не вами.")
		return
	}
	prompt := callbackMessage(cb)
	if err := deleteBotMessage(ctx, r.api, prompt); err != nil {
		_ = editBotMessage(ctx, r.api, prompt, render.MealInputCancelled(), nil, false)
	}
}
