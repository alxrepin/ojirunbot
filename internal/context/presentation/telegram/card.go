package telegram

import (
	"context"
	"strings"

	tg "ojirun/internal/context/infrastructure/telegram"
)

func (r *Router) editCard(ctx context.Context, chatID, messageID int64, text string, opts *tg.SendOptions, rebind func(messageID int64)) {
	if messageID != 0 {
		err := r.api.EditMessageText(ctx, chatID, messageID, text, opts)
		if err == nil || strings.Contains(err.Error(), "message is not modified") {
			return
		}
		_ = r.api.DeleteMessage(ctx, chatID, messageID)
	}
	sent, err := r.api.SendMessage(ctx, chatID, text, opts)
	if err != nil {
		r.log.Error("resend interactive card failed", "error", err)
		return
	}
	if rebind != nil {
		rebind(sent.MessageID)
	}
}
