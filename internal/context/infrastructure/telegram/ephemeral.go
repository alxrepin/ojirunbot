package telegram

import (
	"context"
	"encoding/json"
)

type EphemeralMessageParameters struct {
	ReceiverUserID int64 `json:"receiver_user_id"`
}

func (c *Client) EditEphemeralMessageText(ctx context.Context, chatID, receiverUserID, ephemeralMessageID int64, text string, opts *SendOptions) error {
	payload := ephemeralPayload(chatID, receiverUserID, ephemeralMessageID)
	payload["text"] = text
	applyEditOptions(payload, opts)
	var result json.RawMessage
	return c.call(ctx, "editEphemeralMessageText", payload, &result)
}

func (c *Client) EditEphemeralRichMarkdown(ctx context.Context, chatID, receiverUserID, ephemeralMessageID int64, markdown string, opts *SendOptions) error {
	payload := ephemeralPayload(chatID, receiverUserID, ephemeralMessageID)
	payload["rich_message"] = map[string]any{
		"markdown":              markdown,
		"skip_entity_detection": true,
	}
	applyEditOptions(payload, opts)
	var result json.RawMessage
	return c.call(ctx, "editEphemeralMessageText", payload, &result)
}

func (c *Client) DeleteEphemeralMessage(ctx context.Context, chatID, receiverUserID, ephemeralMessageID int64) error {
	var result json.RawMessage
	return c.call(ctx, "deleteEphemeralMessage", ephemeralPayload(chatID, receiverUserID, ephemeralMessageID), &result)
}

func ephemeralPayload(chatID, receiverUserID, ephemeralMessageID int64) map[string]any {
	return map[string]any{
		"chat_id":              chatID,
		"receiver_user_id":     receiverUserID,
		"ephemeral_message_id": ephemeralMessageID,
	}
}
