package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ojirun/internal/context/infrastructure/httpx"
)

type Client struct {
	token       string
	baseURL     string
	fileBaseURL string
	http        *http.Client
	download    *http.Client
}

func NewClient(apiBaseURL, token, proxyURL string) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if base == "" {
		base = "https://api.telegram.org"
	}
	rpc, err := httpx.NewClient(90*time.Second, proxyURL)
	if err != nil {
		return nil, fmt.Errorf("build telegram rpc client: %w", err)
	}
	download, err := httpx.NewClient(0, proxyURL)
	if err != nil {
		return nil, fmt.Errorf("build telegram download client: %w", err)
	}
	return &Client{
		token:       token,
		baseURL:     base + "/bot" + token,
		fileBaseURL: base + "/file/bot" + token,
		http:        rpc,
		download:    download,
	}, nil
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	var result []Update
	payload := map[string]any{
		"offset":  offset,
		"limit":   50,
		"timeout": timeoutSeconds,
		"allowed_updates": []string{
			"message",
			"callback_query",
		},
	}
	if err := c.call(ctx, "getUpdates", payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetMe(ctx context.Context) (User, error) {
	var result User
	err := c.call(ctx, "getMe", map[string]any{}, &result)
	return result, err
}

func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand, scope *BotCommandScope) error {
	payload := map[string]any{
		"commands": commands,
	}
	if scope != nil {
		payload["scope"] = scope
	}
	var result bool
	return c.call(ctx, "setMyCommands", payload, &result)
}

func (c *Client) SendMessage(ctx context.Context, chatID any, text string, opts *SendOptions) (Message, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	applySendOptions(payload, opts)
	var result Message
	err := c.call(ctx, "sendMessage", payload, &result)
	return result, err
}

func (c *Client) SendRichMarkdown(ctx context.Context, chatID any, markdown string, opts *SendOptions) (Message, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"rich_message": map[string]any{
			"markdown":              markdown,
			"skip_entity_detection": true,
		},
	}
	applySendOptions(payload, opts)
	var result Message
	err := c.call(ctx, "sendRichMessage", payload, &result)
	return result, err
}

func (c *Client) EditMessageText(ctx context.Context, chatID, messageID int64, text string, opts *SendOptions) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	applyEditOptions(payload, opts)
	var result json.RawMessage
	return c.call(ctx, "editMessageText", payload, &result)
}

func (c *Client) EditMessageCaption(ctx context.Context, chatID, messageID int64, caption string, opts *SendOptions) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"caption":    caption,
	}
	applyEditOptions(payload, opts)
	var result json.RawMessage
	return c.call(ctx, "editMessageCaption", payload, &result)
}

func (c *Client) EditRichMarkdown(ctx context.Context, chatID, messageID int64, markdown string, opts *SendOptions) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"rich_message": map[string]any{
			"markdown":              markdown,
			"skip_entity_detection": true,
		},
	}
	applyEditOptions(payload, opts)
	var result json.RawMessage
	return c.call(ctx, "editMessageText", payload, &result)
}

func (c *Client) EditMessageReplyMarkup(ctx context.Context, chatID, messageID int64, replyMarkup any) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
	}
	var result json.RawMessage
	return c.call(ctx, "editMessageReplyMarkup", payload, &result)
}

func (c *Client) SendRichHTMLDraft(ctx context.Context, chatID any, draftID int64, html string) error {
	if draftID == 0 {
		return fmt.Errorf("telegram rich draft id must be non-zero")
	}
	payload := map[string]any{
		"chat_id":  chatID,
		"draft_id": draftID,
		"rich_message": map[string]any{
			"html": html,
		},
	}
	var result bool
	return c.call(ctx, "sendRichMessageDraft", payload, &result)
}

func (c *Client) DeleteMessage(ctx context.Context, chatID, messageID int64) error {
	payload := map[string]any{"chat_id": chatID, "message_id": messageID}
	var result bool
	return c.call(ctx, "deleteMessage", payload, &result)
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, alert bool) error {
	payload := map[string]any{
		"callback_query_id": callbackQueryID,
		"show_alert":        alert,
	}
	if text != "" {
		payload["text"] = text
	}
	var result bool
	return c.call(ctx, "answerCallbackQuery", payload, &result)
}

func (c *Client) GetChatMember(ctx context.Context, chatID any, userID int64) (ChatMember, error) {
	payload := map[string]any{"chat_id": chatID, "user_id": userID}
	var result ChatMember
	err := c.call(ctx, "getChatMember", payload, &result)
	return result, err
}

func (c *Client) GetFile(ctx context.Context, fileID string) (File, error) {
	payload := map[string]any{"file_id": fileID}
	var result File
	err := c.call(ctx, "getFile", payload, &result)
	return result, err
}

func (c *Client) DownloadFile(ctx context.Context, filePath string, dst io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.fileBaseURL+"/"+strings.TrimPrefix(filePath, "/"), nil)
	if err != nil {
		return err
	}
	resp, err := c.download.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram file download status %s", resp.Status)
	}
	_, err = io.Copy(dst, resp.Body)
	return err
}

func (c *Client) call(ctx context.Context, method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, result)
}

func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
		ErrorCode   int             `json:"error_code"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode telegram response status=%s: %w", resp.Status, err)
	}
	if !envelope.OK {
		return &APIError{Code: envelope.ErrorCode, Description: envelope.Description, RetryAfterSeconds: envelope.Parameters.RetryAfter}
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode telegram result: %w", err)
	}
	return nil
}

func applySendOptions(payload map[string]any, opts *SendOptions) {
	applyEditOptions(payload, opts)
	if opts == nil {
		return
	}
	if opts.ReplyParameters != nil {
		payload["reply_parameters"] = opts.ReplyParameters
	}
	if opts.Ephemeral != nil {
		payload["ephemeral_message_parameters"] = opts.Ephemeral
	}
}

func applyEditOptions(payload map[string]any, opts *SendOptions) {
	if opts == nil {
		return
	}
	if opts.ParseMode != "" {
		payload["parse_mode"] = opts.ParseMode
	}
	if opts.ReplyMarkup != nil {
		payload["reply_markup"] = opts.ReplyMarkup
	}
	if opts.DisableNotification {
		payload["disable_notification"] = true
	}
}
