package telegram

import (
	"context"

	tg "ojirun/internal/context/infrastructure/telegram"
)

type API interface {
	SendMessage(ctx context.Context, chatID any, text string, opts *tg.SendOptions) (tg.Message, error)
	SendRichMarkdown(ctx context.Context, chatID any, markdown string, opts *tg.SendOptions) (tg.Message, error)
	EditMessageText(ctx context.Context, chatID, messageID int64, text string, opts *tg.SendOptions) error
	EditMessageCaption(ctx context.Context, chatID, messageID int64, caption string, opts *tg.SendOptions) error
	EditRichMarkdown(ctx context.Context, chatID, messageID int64, markdown string, opts *tg.SendOptions) error
	EditMessageReplyMarkup(ctx context.Context, chatID, messageID int64, replyMarkup any) error
	SendRichHTMLDraft(ctx context.Context, chatID any, draftID int64, html string) error
	DeleteMessage(ctx context.Context, chatID, messageID int64) error
	EditEphemeralMessageText(ctx context.Context, chatID, receiverUserID, ephemeralMessageID int64, text string, opts *tg.SendOptions) error
	EditEphemeralRichMarkdown(ctx context.Context, chatID, receiverUserID, ephemeralMessageID int64, markdown string, opts *tg.SendOptions) error
	DeleteEphemeralMessage(ctx context.Context, chatID, receiverUserID, ephemeralMessageID int64) error
	AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, alert bool) error
}
