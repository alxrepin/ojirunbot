package telegram

import (
	"context"

	tg "ojirun/internal/context/infrastructure/telegram"
)

func (r *Router) notify(ctx context.Context, msg tg.Message, text string, opts *tg.SendOptions) (botMessage, error) {
	return sendBotMessage(ctx, r.api, msg.Chat.ID, receiverOf(msg), text, replyToMessage(msg, opts), false)
}

func receiverOf(msg tg.Message) int64 {
	if msg.Chat.Type == "private" {
		return 0
	}
	return msg.From.ID
}

func senderMessage(msg tg.Message, id int64) botMessage {
	return botMessage{ChatID: msg.Chat.ID, ID: id, Receiver: receiverOf(msg)}
}

func (r *Router) discard(ctx context.Context, msg botMessage) {
	if err := deleteBotMessage(ctx, r.api, msg); err != nil {
		r.log.Debug("delete bot message failed", "error", err, "chat_id", msg.ChatID, "message_id", msg.ID)
	}
}

func replyToMessage(msg tg.Message, opts *tg.SendOptions) *tg.SendOptions {
	if msg.MessageID == 0 || msg.EphemeralMessageID != 0 {
		return opts
	}
	copied := copyOptions(opts)
	copied.ReplyParameters = &tg.ReplyParameters{MessageID: msg.MessageID}
	return copied
}

func htmlOptions() *tg.SendOptions {
	return &tg.SendOptions{ParseMode: "HTML"}
}
