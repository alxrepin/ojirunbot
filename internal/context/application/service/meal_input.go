package service

import (
	"context"
	"errors"

	"ojirun/internal/context/domain"
)

const mealInputStep = "meal"

type MealInput struct {
	sessions sessionStore
}

func NewMealInput(sessions sessionStore) *MealInput {
	return &MealInput{sessions: sessions}
}

func (m *MealInput) Begin(ctx context.Context, telegramUserID, chatID int64) (sessionID string, stalePromptID int64, err error) {
	previous, err := m.sessions.GetActive(ctx, domain.SessionMealInput, telegramUserID, chatID)
	if err != nil && !errors.Is(err, domain.ErrSessionNotFound) {
		return "", 0, err
	}
	sessionID, err = m.sessions.Start(ctx, domain.SessionMealInput, telegramUserID, chatID, mealInputStep, map[string]any{})
	if err != nil {
		return "", 0, err
	}
	return sessionID, previous.BotMessageID, nil
}

func (m *MealInput) BindPrompt(ctx context.Context, telegramUserID, chatID, promptMessageID int64) error {
	return m.sessions.SetBotMessage(ctx, domain.SessionMealInput, telegramUserID, chatID, promptMessageID)
}

func (m *MealInput) Claim(ctx context.Context, telegramUserID, chatID int64) (promptMessageID int64, ok bool, err error) {
	session, err := m.sessions.GetActive(ctx, domain.SessionMealInput, telegramUserID, chatID)
	if errors.Is(err, domain.ErrSessionNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	closed, err := m.sessions.Close(ctx, session.ID, telegramUserID)
	if err != nil || !closed {
		return 0, false, err
	}
	return session.BotMessageID, true, nil
}

func (m *MealInput) Cancel(ctx context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return m.sessions.Cancel(ctx, sessionID, telegramUserID)
}
