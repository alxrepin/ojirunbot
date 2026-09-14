package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(c *postgres.Client) *Repository {
	return &Repository{pool: c.Pool()}
}

func (r *Repository) Start(ctx context.Context, kind domain.SessionKind, telegramUserID, chatID int64, step string, payload any) (string, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal session payload: %w", err)
	}
	var id string
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, cancelActiveQuery, telegramUserID, chatID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, insertQuery, string(kind), telegramUserID, chatID, step, payloadJSON).Scan(&id)
	})
	if err != nil {
		return "", fmt.Errorf("start %s session: %w", kind, err)
	}
	return id, nil
}

func (r *Repository) GetActive(ctx context.Context, kind domain.SessionKind, telegramUserID, chatID int64) (domain.Session, error) {
	var (
		s        domain.Session
		kindText string
	)
	err := r.pool.QueryRow(ctx, selectActiveQuery, string(kind), telegramUserID, chatID).
		Scan(&s.ID, &kindText, &s.UserID, &s.TelegramUserID, &s.ChatID, &s.Step, &s.PayloadJSON, &s.BotMessageID, &s.Status, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("get active %s session: %w", kind, err)
	}
	s.Kind = domain.SessionKind(kindText)
	return s, nil
}

func (r *Repository) Advance(ctx context.Context, sessionID, fromStep, step string, payload any) (bool, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("marshal session payload: %w", err)
	}
	tag, err := r.pool.Exec(ctx, advanceQuery, sessionID, fromStep, step, payloadJSON)
	if err != nil {
		return false, fmt.Errorf("advance session: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) SetBotMessage(ctx context.Context, kind domain.SessionKind, telegramUserID, chatID, botMessageID int64) error {
	if _, err := r.pool.Exec(ctx, setBotMessageQuery, string(kind), telegramUserID, chatID, botMessageID); err != nil {
		return fmt.Errorf("set %s session bot message: %w", kind, err)
	}
	return nil
}

func (r *Repository) Complete(ctx context.Context, sessionID, userID string) error {
	if _, err := r.pool.Exec(ctx, completeQuery, sessionID, userID); err != nil {
		return fmt.Errorf("complete registration session: %w", err)
	}
	return nil
}

func (r *Repository) Close(ctx context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return r.finish(ctx, closeQuery, "close", sessionID, telegramUserID)
}

func (r *Repository) Cancel(ctx context.Context, sessionID string, telegramUserID int64) (bool, error) {
	return r.finish(ctx, cancelQuery, "cancel", sessionID, telegramUserID)
}

func (r *Repository) finish(ctx context.Context, query, action, sessionID string, telegramUserID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, query, sessionID, telegramUserID)
	if err != nil {
		return false, fmt.Errorf("%s session: %w", action, err)
	}
	return tag.RowsAffected() > 0, nil
}
