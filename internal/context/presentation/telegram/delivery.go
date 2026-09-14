package telegram

import (
	"context"

	tg "ojirun/internal/context/infrastructure/telegram"
)

type botMessage struct {
	ChatID   int64
	ID       int64
	Receiver int64
}

func inGroup(chatID int64) bool {
	return chatID < 0
}

func sendBotMessage(ctx context.Context, api API, chatID, receiver int64, text string, opts *tg.SendOptions, rich bool) (botMessage, error) {
	if receiver != 0 {
		opts = withEphemeral(opts, receiver)
	}
	var (
		sent tg.Message
		err  error
	)
	if rich {
		sent, err = api.SendRichMarkdown(ctx, chatID, text, opts)
	} else {
		sent, err = api.SendMessage(ctx, chatID, text, opts)
	}
	if err != nil {
		return botMessage{}, err
	}
	if receiver != 0 {
		return botMessage{ChatID: chatID, ID: sent.EphemeralMessageID, Receiver: receiver}, nil
	}
	return botMessage{ChatID: chatID, ID: sent.MessageID}, nil
}

func editBotMessage(ctx context.Context, api API, msg botMessage, text string, opts *tg.SendOptions, rich bool) error {
	switch {
	case msg.Receiver != 0 && rich:
		return api.EditEphemeralRichMarkdown(ctx, msg.ChatID, msg.Receiver, msg.ID, text, opts)
	case msg.Receiver != 0:
		return api.EditEphemeralMessageText(ctx, msg.ChatID, msg.Receiver, msg.ID, text, opts)
	case rich:
		return api.EditRichMarkdown(ctx, msg.ChatID, msg.ID, text, opts)
	}
	if err := api.EditMessageText(ctx, msg.ChatID, msg.ID, text, opts); err == nil {
		return nil
	}
	return api.EditMessageCaption(ctx, msg.ChatID, msg.ID, text, opts)
}

func deleteBotMessage(ctx context.Context, api API, msg botMessage) error {
	if msg.ID == 0 {
		return nil
	}
	if msg.Receiver != 0 {
		return api.DeleteEphemeralMessage(ctx, msg.ChatID, msg.Receiver, msg.ID)
	}
	return api.DeleteMessage(ctx, msg.ChatID, msg.ID)
}

func callbackMessage(cb tg.CallbackQuery) botMessage {
	if cb.Message.EphemeralMessageID != 0 {
		return botMessage{ChatID: cb.Message.Chat.ID, ID: cb.Message.EphemeralMessageID, Receiver: cb.From.ID}
	}
	return botMessage{ChatID: cb.Message.Chat.ID, ID: cb.Message.MessageID}
}

func withEphemeral(opts *tg.SendOptions, receiver int64) *tg.SendOptions {
	copied := copyOptions(opts)
	copied.Ephemeral = &tg.EphemeralMessageParameters{ReceiverUserID: receiver}
	return copied
}

func copyOptions(opts *tg.SendOptions) *tg.SendOptions {
	if opts == nil {
		return &tg.SendOptions{}
	}
	copied := *opts
	return &copied
}
