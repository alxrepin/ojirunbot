package telegram

import (
	"context"
	"errors"

	"ojirun/internal/context/application/service"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) startSettings(ctx context.Context, msg tg.Message) {
	outcome, err := r.settings.Begin(ctx, msg.From.ID, msg.Chat.ID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrProfileNotFound) {
			_, _ = r.api.SendMessage(ctx, msg.Chat.ID, render.NeedRegisterPrivate(), nil)
			return
		}
		r.log.Error("start settings failed", "error", err)
		_, _ = r.api.SendMessage(ctx, msg.Chat.ID, render.SettingsError(), htmlOptions())
		return
	}
	text := render.SettingsCard(outcome.Step, outcome.Current, outcome.Draft, false)
	sent, err := r.api.SendMessage(ctx, msg.Chat.ID, text, settingsOptions(outcome))
	if err != nil {
		r.log.Error("send settings card failed", "error", err)
		return
	}
	r.bindSettingsCard(ctx, msg.From.ID, msg.Chat.ID, sent.MessageID)
}

func (r *Router) advanceSettings(ctx context.Context, msg tg.Message, session domain.Session, text string) {
	outcome, err := r.settings.SubmitText(ctx, session, text)
	r.deleteUserMessage(ctx, msg)
	r.renderSettings(ctx, session.TelegramUserID, session.ChatID, session.BotMessageID, outcome, err)
}

func (r *Router) handleSettingsCallback(ctx context.Context, cb tg.CallbackQuery, action tg.SettingsCallback) {
	session, err := r.settings.ActiveSession(ctx, cb.From.ID, cb.Message.Chat.ID)
	if err != nil {
		return
	}
	if session.BotMessageID != 0 && session.BotMessageID != cb.Message.MessageID {
		return
	}

	var outcome service.SettingsOutcome
	switch action.Value {
	case tg.SettingsCancel:
		outcome, err = r.settings.Cancel(ctx, session)
	case tg.SettingsKeep:
		outcome, err = r.settings.Keep(ctx, session, action.Step)
	default:
		outcome, err = r.settings.Choose(ctx, session, action.Step, action.Value)
	}
	r.renderSettings(ctx, session.TelegramUserID, session.ChatID, cb.Message.MessageID, outcome, err)
}

func (r *Router) renderSettings(ctx context.Context, telegramUserID, chatID, messageID int64, outcome service.SettingsOutcome, err error) {
	switch {
	case err != nil:
		r.log.Error("settings step failed", "error", err)
		r.editCard(ctx, chatID, messageID, render.SettingsError(), htmlOptions(), nil)
	case outcome.Saved:
		r.editCard(ctx, chatID, messageID, render.SettingsSaved(outcome.Current, outcome.Draft), htmlOptions(), nil)
	case outcome.Cancelled:
		r.editCard(ctx, chatID, messageID, render.SettingsCancelled(), htmlOptions(), nil)
	case outcome.Step != "":
		text := render.SettingsCard(outcome.Step, outcome.Current, outcome.Draft, outcome.Invalid)
		r.editCard(ctx, chatID, messageID, text, settingsOptions(outcome), func(newID int64) {
			r.bindSettingsCard(ctx, telegramUserID, chatID, newID)
		})
	}
}

func (r *Router) bindSettingsCard(ctx context.Context, telegramUserID, chatID, messageID int64) {
	if err := r.settings.BindMessage(ctx, telegramUserID, chatID, messageID); err != nil {
		r.log.Warn("bind settings card failed", "error", err)
	}
}

func settingsOptions(outcome service.SettingsOutcome) *tg.SendOptions {
	return &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: tg.SettingsKeyboard(outcome.Step, render.SettingsKeepLabel(outcome.Step, outcome.Current)),
	}
}
