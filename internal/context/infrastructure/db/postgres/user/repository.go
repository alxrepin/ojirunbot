package user

import (
	"context"
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

func (r *Repository) Upsert(ctx context.Context, telegramUserID int64, username, displayName string) (domain.User, error) {
	var row user
	err := r.pool.QueryRow(ctx, upsertQuery, telegramUserID, username, displayName).
		Scan(&row.ID, &row.TelegramUserID, &row.Username, &row.DisplayName)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return row.toDomain(), nil
}

func (r *Repository) GetByTelegramID(ctx context.Context, telegramUserID int64) (domain.User, error) {
	var row user
	err := r.pool.QueryRow(ctx, selectByTelegramIDQuery, telegramUserID).
		Scan(&row.ID, &row.TelegramUserID, &row.Username, &row.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by telegram id: %w", err)
	}
	return row.toDomain(), nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	var row user
	err := r.pool.QueryRow(ctx, selectByUsernameQuery, username).
		Scan(&row.ID, &row.TelegramUserID, &row.Username, &row.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by username: %w", err)
	}
	return row.toDomain(), nil
}
