package telegram

import (
	"context"

	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type callbackAnswer struct {
	text  string
	alert bool
}

func (a *callbackAnswer) Alert(text string) {
	a.text, a.alert = text, true
}

func (r *Router) handleCallback(ctx context.Context, cb tg.CallbackQuery) {
	var answer callbackAnswer
	defer func() {
		_ = r.api.AnswerCallbackQuery(ctx, cb.ID, answer.text, answer.alert)
	}()
	if cb.Message == nil {
		return
	}
	if cb.Data == tg.SubscriptionCheckCallback {
		r.handleSubscriptionCheck(ctx, cb, &answer)
		return
	}
	subscribed, err := r.subscription.IsSubscribed(ctx, cb.From.ID)
	switch {
	case err != nil:
		answer.Alert(render.SubscriptionCheckFailed())
		return
	case !subscribed:
		answer.Alert(render.SubscribeAlert(r.channel.Label))
		return
	}

	if cb.Data == tg.ProfileSettingsCallback || cb.Data == tg.ProfileRestartCallback {
		r.handleProfileCallback(ctx, cb)
		return
	}
	if reg, ok := tg.DecodeRegCallback(cb.Data); ok {
		r.handleRegistrationCallback(ctx, cb, reg)
		return
	}
	if settings, ok := tg.DecodeSettingsCallback(cb.Data); ok {
		r.handleSettingsCallback(ctx, cb, settings)
		return
	}
	if sessionID, ok := tg.DecodeMealInputCancel(cb.Data); ok {
		r.handleMealInputCancel(ctx, cb, sessionID, &answer)
		return
	}
	action, ok := tg.DecodeMealCallback(cb.Data)
	if !ok {
		return
	}

	entry, revision, err := r.actions.Load(ctx, action.MealID)
	if err != nil {
		r.log.Error("get meal for callback failed", "error", err)
		return
	}
	if entry.TelegramUserID != cb.From.ID {
		answer.Alert("Это не ваша запись.")
		return
	}

	switch action.Action {
	case tg.ActionAccept:
		if entry.Status == domain.StatusAccepted {
			return
		}
		ok, err := r.actions.Accept(ctx, action.MealID, cb.From.ID)
		if err != nil || !ok {
			answer.Alert("Не удалось принять запись.")
			return
		}
		r.pipeline.ShowAccepted(ctx, entry, revision.Analysis())
	case tg.ActionEdit:
		if !revision.CanCorrect() {
			answer.Alert(render.CorrectionLimitAlert(domain.MaxCorrectionsPerEntry))
			return
		}
		ok, err := r.actions.RequestEdit(ctx, action.MealID, cb.From.ID)
		if err != nil || !ok {
			answer.Alert("Запись уже нельзя редактировать.")
			return
		}
		_ = r.messenger.updateMessage(ctx, entry, render.AwaitCorrection(), &tg.SendOptions{
			ReplyMarkup: tg.MealCorrectionKeyboard(entry.ID),
		})
	case tg.ActionCancel:
		ok, err := r.actions.CancelEdit(ctx, action.MealID, cb.From.ID)
		if err != nil || !ok {
			answer.Alert("Режим редактирования уже закрыт.")
			return
		}
		r.pipeline.ShowResult(ctx, entry, revision.Analysis())
	case tg.ActionDelete:
		ok, err := r.actions.Delete(ctx, action.MealID, cb.From.ID)
		if err != nil || !ok {
			answer.Alert("Не удалось удалить запись.")
			return
		}
		r.messenger.Deleted(ctx, entry)
	}
}
