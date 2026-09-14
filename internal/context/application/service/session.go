package service

import (
	"context"

	"ojirun/internal/context/domain"
)

type sessionStore interface {
	Start(ctx context.Context, kind domain.SessionKind, telegramUserID, chatID int64, step string, payload any) (string, error)
	GetActive(ctx context.Context, kind domain.SessionKind, telegramUserID, chatID int64) (domain.Session, error)
	SetBotMessage(ctx context.Context, kind domain.SessionKind, telegramUserID, chatID, botMessageID int64) error
	Advance(ctx context.Context, sessionID, fromStep, step string, payload any) (bool, error)
	Close(ctx context.Context, sessionID string, telegramUserID int64) (bool, error)
	Cancel(ctx context.Context, sessionID string, telegramUserID int64) (bool, error)
	Complete(ctx context.Context, sessionID, userID string) error
}
