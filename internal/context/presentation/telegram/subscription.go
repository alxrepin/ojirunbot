package telegram

import (
	"context"
	"errors"

	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) requireSubscriber(ctx context.Context, msg tg.Message) bool {
	if r.subscription.IsSubscribed(ctx, msg.From.ID) {
		return true
	}
	text := render.SubscribeRequiredGroup(r.channel.Label, r.channel.URL)
	if msg.Chat.Type == "private" {
		text = render.SubscribeRequiredPrivate(r.channel.Label, r.channel.URL)
	}
	opts := &tg.SendOptions{ParseMode: "HTML", ReplyMarkup: tg.SubscribeKeyboard(r.channel.URL)}
	_, _ = r.notify(ctx, msg, text, opts)
	return false
}

func (r *Router) handleSubscriptionCheck(ctx context.Context, cb tg.CallbackQuery, answer *callbackAnswer) {
	if !r.subscription.Recheck(ctx, cb.From.ID) {
		answer.Alert(render.SubscriptionNotFound())
		return
	}
	if err := editBotMessage(ctx, r.api, callbackMessage(cb), render.SubscriptionConfirmed(), nil, false); err != nil {
		r.log.Debug("edit subscribe prompt failed", "error", err)
	}
	chat := cb.Message.Chat
	if chat.Type != "private" {
		return
	}
	if _, err := r.profiles.Execute(ctx, cb.From.ID); errors.Is(err, domain.ErrNotFound) {
		r.startRegistration(ctx, tg.Message{From: &cb.From, Chat: chat})
	}
}
