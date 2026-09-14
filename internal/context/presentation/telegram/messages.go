package telegram

import (
	"context"
	"log/slog"

	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type mealMessageBinder interface {
	SetBotMessage(ctx context.Context, mealID string, botMessageID int64) error
	SetEphemeralMessage(ctx context.Context, mealID string, ephemeralMessageID int64) error
}

type MealMessenger struct {
	api    API
	binder mealMessageBinder
	log    *slog.Logger
}

func NewMealMessenger(api API, binder mealMessageBinder, log *slog.Logger) *MealMessenger {
	return &MealMessenger{api: api, binder: binder, log: log}
}

func (m *MealMessenger) Received(ctx context.Context, entry domain.MealEntry, backlog int) domain.MealEntry {
	if m.drafts(entry) {
		m.sendDraft(ctx, entry, render.MealReceivedDraft(backlog))
		return entry
	}
	text := render.MealStageStatus(domain.StageReceived)
	if backlog > 0 {
		text = render.MealQueued(backlog)
	}
	return m.sendStatusReply(ctx, entry, text)
}

func (m *MealMessenger) Queued(ctx context.Context, entry domain.MealEntry, backlog int) {
	if m.drafts(entry) {
		m.sendDraft(ctx, entry, render.MealReceivedDraft(backlog))
		return
	}
	_ = m.updateMessage(ctx, entry, render.MealQueued(backlog), nil)
}

func (m *MealMessenger) Stage(ctx context.Context, entry domain.MealEntry, stage domain.MealStage) {
	if m.drafts(entry) {
		m.sendDraft(ctx, entry, render.MealStageDraft(stage))
		return
	}
	if stage != domain.StageReceived {
		_ = m.updateMessage(ctx, entry, render.MealStageStatus(stage), nil)
	}
}

func (m *MealMessenger) drafts(entry domain.MealEntry) bool {
	return !inGroup(entry.ChatID) && mealDraftID(entry) != 0
}

func (m *MealMessenger) Result(ctx context.Context, entry domain.MealEntry, view domain.MealResultView) {
	text := render.MealResult(view)
	opts := &tg.SendOptions{ReplyMarkup: tg.MealKeyboard(entry.ID)}
	if !inGroup(entry.ChatID) && m.finalize(ctx, entry, text, opts, true) {
		return
	}
	if err := m.updateRichMessage(ctx, entry, text, opts); err != nil {
		m.log.Warn("update meal rich message failed, falling back to plain text", "error", err)
		if err := m.updateMessage(ctx, entry, text, opts); err != nil {
			m.log.Error("update meal message failed", "error", err)
		}
	}
}

func (m *MealMessenger) finalize(ctx context.Context, entry domain.MealEntry, text string, opts *tg.SendOptions, rich bool) bool {
	previous := m.card(entry)
	if err := m.send(ctx, &entry, text, opts, rich); err != nil {
		m.log.Warn("send final meal message failed, falling back to editing the status", "error", err, "meal_entry_id", entry.ID)
		return false
	}
	if err := deleteBotMessage(ctx, m.api, previous); err != nil {
		m.log.Debug("delete meal status message failed", "error", err, "meal_entry_id", entry.ID)
	}
	return true
}

func (m *MealMessenger) Accepted(ctx context.Context, entry domain.MealEntry, view domain.MealResultView) {
	if !inGroup(entry.ChatID) {
		if err := m.api.EditMessageReplyMarkup(ctx, entry.ChatID, entry.BotMessageID, nil); err != nil {
			m.log.Warn("remove accepted meal keyboard failed", "error", err, "meal_entry_id", entry.ID)
		}
		return
	}

	text := render.MealResult(view)
	opts := sourceReplyOptions(entry, nil)
	published, err := sendBotMessage(ctx, m.api, entry.ChatID, 0, text, opts, true)
	if err != nil {
		m.log.Warn("publish accepted meal as rich message failed, falling back to plain text", "error", err)
		published, err = sendBotMessage(ctx, m.api, entry.ChatID, 0, text, opts, false)
	}
	if err != nil {
		m.log.Error("publish accepted meal failed", "error", err, "meal_entry_id", entry.ID)
		return
	}
	_ = m.binder.SetBotMessage(ctx, entry.ID, published.ID)
	if err := deleteBotMessage(ctx, m.api, m.card(entry)); err != nil {
		m.log.Debug("delete ephemeral meal card failed", "error", err, "meal_entry_id", entry.ID)
	}
	_ = m.binder.SetEphemeralMessage(ctx, entry.ID, 0)
}

func (m *MealMessenger) Deleted(ctx context.Context, entry domain.MealEntry) {
	for _, card := range m.cards(entry) {
		if err := deleteBotMessage(ctx, m.api, card); err != nil {
			_ = editBotMessage(ctx, m.api, card, render.Deleted(), nil, false)
		}
	}
}

func (m *MealMessenger) Fail(ctx context.Context, entry domain.MealEntry, failure domain.MealFailure) {
	m.conclude(ctx, entry, render.MealFailureText(failure))
}

func (m *MealMessenger) Discard(ctx context.Context, entry domain.MealEntry) {
	m.conclude(ctx, entry, render.MealNothingRecognized())
}

func (m *MealMessenger) conclude(ctx context.Context, entry domain.MealEntry, text string) {
	if !inGroup(entry.ChatID) && m.finalize(ctx, entry, text, nil, false) {
		return
	}
	_ = m.updateMessage(ctx, entry, text, nil)
}

func (m *MealMessenger) sendStatusReply(ctx context.Context, entry domain.MealEntry, text string) domain.MealEntry {
	if err := m.send(ctx, &entry, text, nil, false); err != nil {
		m.log.Warn("send meal status failed", "error", err, "meal_entry_id", entry.ID)
	}
	return entry
}

func (m *MealMessenger) card(entry domain.MealEntry) botMessage {
	if inGroup(entry.ChatID) {
		return botMessage{ChatID: entry.ChatID, ID: entry.EphemeralMessageID, Receiver: entry.TelegramUserID}
	}
	return botMessage{ChatID: entry.ChatID, ID: entry.BotMessageID}
}

func (m *MealMessenger) cards(entry domain.MealEntry) []botMessage {
	var cards []botMessage
	if inGroup(entry.ChatID) && entry.EphemeralMessageID != 0 {
		cards = append(cards, botMessage{ChatID: entry.ChatID, ID: entry.EphemeralMessageID, Receiver: entry.TelegramUserID})
	}
	if entry.BotMessageID != 0 {
		cards = append(cards, botMessage{ChatID: entry.ChatID, ID: entry.BotMessageID})
	}
	return cards
}

func (m *MealMessenger) send(ctx context.Context, entry *domain.MealEntry, text string, opts *tg.SendOptions, rich bool) error {
	card := m.card(*entry)
	sent, err := sendBotMessage(ctx, m.api, card.ChatID, card.Receiver, text, sourceReplyOptions(*entry, opts), rich)
	if err != nil {
		return err
	}
	if sent.Receiver != 0 {
		entry.EphemeralMessageID = sent.ID
		_ = m.binder.SetEphemeralMessage(ctx, entry.ID, sent.ID)
	} else {
		entry.BotMessageID = sent.ID
		_ = m.binder.SetBotMessage(ctx, entry.ID, sent.ID)
	}
	return nil
}

func (m *MealMessenger) updateMessage(ctx context.Context, entry domain.MealEntry, text string, opts *tg.SendOptions) error {
	card := m.card(entry)
	if card.ID == 0 {
		return m.send(ctx, &entry, text, opts, false)
	}
	return editBotMessage(ctx, m.api, card, text, opts, false)
}

func (m *MealMessenger) updateRichMessage(ctx context.Context, entry domain.MealEntry, markdown string, opts *tg.SendOptions) error {
	card := m.card(entry)
	if card.ID == 0 {
		return m.send(ctx, &entry, markdown, opts, true)
	}
	return editBotMessage(ctx, m.api, card, markdown, opts, true)
}

func (m *MealMessenger) sendDraft(ctx context.Context, entry domain.MealEntry, html string) {
	draftID := mealDraftID(entry)
	if draftID == 0 {
		return
	}
	if err := m.api.SendRichHTMLDraft(ctx, entry.ChatID, draftID, html); err != nil {
		m.log.Debug("send meal rich draft failed", "error", err, "meal_entry_id", entry.ID)
	}
}

func mealDraftID(entry domain.MealEntry) int64 {
	if entry.SourceMessageID != 0 {
		return entry.SourceMessageID
	}
	return entry.BotMessageID
}

func sourceReplyOptions(entry domain.MealEntry, opts *tg.SendOptions) *tg.SendOptions {
	if entry.SourceMessageID == 0 {
		return opts
	}
	copied := copyOptions(opts)
	copied.ReplyParameters = &tg.ReplyParameters{MessageID: entry.SourceMessageID}
	return copied
}
