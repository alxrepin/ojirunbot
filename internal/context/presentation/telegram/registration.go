package telegram

import (
	"context"
	"errors"

	"ojirun/internal/context/application/service"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) startRegistration(ctx context.Context, msg tg.Message) {
	if err := r.registration.Begin(ctx, msg.From.ID, msg.Chat.ID); err != nil {
		r.log.Error("start registration failed", "error", err)
		_, _ = r.api.SendMessage(ctx, msg.Chat.ID, render.RegistrationError(), htmlOptions())
		return
	}

	text := render.RegistrationCard("sex", domain.RegistrationDraft{}, false)
	sent, err := r.api.SendMessage(ctx, msg.Chat.ID, text, registrationOptions("sex"))
	if err != nil {
		r.log.Error("send registration card failed", "error", err)
		return
	}
	if err := r.registration.BindMessage(ctx, msg.From.ID, msg.Chat.ID, sent.MessageID); err != nil {
		r.log.Warn("bind registration card failed", "error", err)
	}
}

func (r *Router) advanceRegistration(ctx context.Context, msg tg.Message, session domain.Session, text string) {
	outcome, err := r.registration.SubmitText(ctx, session, registrantInfo(msg.From), text)
	if err != nil {
		r.log.Error("advance registration failed", "error", err)
		return
	}
	r.deleteUserMessage(ctx, msg)
	r.renderRegistration(ctx, msg.From.ID, msg.Chat.ID, session.BotMessageID, outcome)
}

func (r *Router) handleRegistrationCallback(ctx context.Context, cb tg.CallbackQuery, reg tg.RegCallback) {
	session, err := r.registration.ActiveSession(ctx, cb.From.ID, cb.Message.Chat.ID)
	if err != nil {
		return
	}
	outcome, err := r.registration.SubmitChoice(ctx, session, registrantInfo(&cb.From), reg.Step, reg.Value)
	if err != nil {
		r.log.Error("registration choice failed", "error", err)
		return
	}
	messageID := session.BotMessageID
	if messageID == 0 {
		messageID = cb.Message.MessageID
	}
	r.renderRegistration(ctx, cb.From.ID, cb.Message.Chat.ID, messageID, outcome)
}

func (r *Router) renderRegistration(ctx context.Context, telegramUserID, chatID, botMessageID int64, outcome service.RegistrationOutcome) {
	switch {
	case outcome.Done:
		r.editRegistrationCard(ctx, telegramUserID, chatID, botMessageID, render.RegistrationDone(outcome.Draft, outcome.Targets), "")
		r.sendMenu(ctx, chatID, render.MenuReady())
	case outcome.Prompt != "":
		text := render.RegistrationCard(outcome.Prompt, outcome.Draft, outcome.Invalid)
		r.editRegistrationCard(ctx, telegramUserID, chatID, botMessageID, text, outcome.Prompt)
	}
}

func (r *Router) editRegistrationCard(ctx context.Context, telegramUserID, chatID, botMessageID int64, text, step string) {
	var rebind func(int64)
	if step != "" {
		rebind = func(messageID int64) {
			if err := r.registration.BindMessage(ctx, telegramUserID, chatID, messageID); err != nil {
				r.log.Warn("rebind registration card failed", "error", err)
			}
		}
	}
	r.editCard(ctx, chatID, botMessageID, text, registrationOptions(step), rebind)
}

func registrationOptions(step string) *tg.SendOptions {
	opts := &tg.SendOptions{ParseMode: "HTML"}
	if kb, ok := tg.RegistrationKeyboard(step); ok {
		opts.ReplyMarkup = kb
	}
	return opts
}

func (r *Router) deleteUserMessage(ctx context.Context, msg tg.Message) {
	if msg.MessageID == 0 {
		return
	}
	if err := r.api.DeleteMessage(ctx, msg.Chat.ID, msg.MessageID); err != nil {
		r.log.Debug("delete user message failed", "error", err, "chat_id", msg.Chat.ID, "message_id", msg.MessageID)
	}
}

func (r *Router) showProfile(ctx context.Context, msg tg.Message) {
	profile, err := r.profiles.Execute(ctx, msg.From.ID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			_, _ = r.api.SendMessage(ctx, msg.Chat.ID, "Профиль не найден. Начните с /start.", nil)
		case errors.Is(err, domain.ErrProfileNotFound):
			_, _ = r.api.SendMessage(ctx, msg.Chat.ID, "Профиль не завершён. Отправьте /start.", nil)
		default:
			r.log.Error("profile lookup failed", "error", err)
		}
		return
	}
	_, _ = r.api.SendMessage(ctx, msg.Chat.ID, render.Profile(profile), withMenu(htmlOptions()))
}

func registrantInfo(u *tg.User) service.RegistrantInfo {
	return service.RegistrantInfo{
		TelegramUserID: u.ID,
		Username:       u.Username,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
	}
}
