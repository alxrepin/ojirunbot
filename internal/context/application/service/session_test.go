package service

import (
	"context"
	"encoding/json"
	"fmt"

	"ojirun/internal/context/domain"
)

type memSessions struct {
	rows []*domain.Session
}

func (m *memSessions) Start(_ context.Context, kind domain.SessionKind, telegramUserID, chatID int64, step string, payload any) (string, error) {
	for _, s := range m.rows {
		if s.TelegramUserID == telegramUserID && s.ChatID == chatID && s.Status == "active" {
			s.Status = "cancelled"
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	s := &domain.Session{
		ID: fmt.Sprintf("s%d", len(m.rows)+1), Kind: kind, TelegramUserID: telegramUserID,
		ChatID: chatID, Step: step, PayloadJSON: raw, Status: "active",
	}
	m.rows = append(m.rows, s)
	return s.ID, nil
}

func (m *memSessions) GetActive(_ context.Context, kind domain.SessionKind, telegramUserID, chatID int64) (domain.Session, error) {
	if s := m.active(kind, telegramUserID, chatID); s != nil {
		return *s, nil
	}
	return domain.Session{}, domain.ErrSessionNotFound
}

func (m *memSessions) SetBotMessage(_ context.Context, kind domain.SessionKind, telegramUserID, chatID, botMessageID int64) error {
	if s := m.active(kind, telegramUserID, chatID); s != nil {
		s.BotMessageID = botMessageID
	}
	return nil
}

func (m *memSessions) Advance(_ context.Context, sessionID, fromStep, step string, payload any) (bool, error) {
	s := m.byID(sessionID)
	if s == nil || s.Status != "active" || s.Step != fromStep {
		return false, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	s.Step, s.PayloadJSON = step, raw
	return true, nil
}

func (m *memSessions) Close(_ context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return m.finish(sessionID, telegramUserID, "completed"), nil
}

func (m *memSessions) Cancel(_ context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return m.finish(sessionID, telegramUserID, "cancelled"), nil
}

func (m *memSessions) Complete(_ context.Context, sessionID, _ string) error {
	if s := m.byID(sessionID); s != nil {
		s.Status = "completed"
	}
	return nil
}

func (m *memSessions) finish(sessionID string, telegramUserID int64, status string) bool {
	s := m.byID(sessionID)
	if s == nil || s.Status != "active" || s.TelegramUserID != telegramUserID {
		return false
	}
	s.Status = status
	return true
}

func (m *memSessions) active(kind domain.SessionKind, telegramUserID, chatID int64) *domain.Session {
	for i := len(m.rows) - 1; i >= 0; i-- {
		s := m.rows[i]
		if s.Kind == kind && s.TelegramUserID == telegramUserID && s.ChatID == chatID && s.Status == "active" {
			return s
		}
	}
	return nil
}

func (m *memSessions) byID(id string) *domain.Session {
	for _, s := range m.rows {
		if s.ID == id {
			return s
		}
	}
	return nil
}
