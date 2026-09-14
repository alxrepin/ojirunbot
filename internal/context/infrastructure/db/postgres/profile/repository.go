package profile

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

func (r *Repository) Save(ctx context.Context, p domain.Profile) error {
	_, err := r.pool.Exec(ctx, upsertQuery,
		p.UserID, p.Sex, p.Age, p.HeightCM, p.WeightKG, p.ActivityLevel, p.Goal,
		p.DailyCaloriesKCal, p.DailyProteinG, p.DailyFatG, p.DailyCarbsG, p.FormulaVersion)
	if err != nil {
		return fmt.Errorf("save profile: %w", err)
	}
	return nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID string) (domain.Profile, error) {
	var p domain.Profile
	err := r.pool.QueryRow(ctx, selectByUserIDQuery, userID).Scan(
		&p.UserID, &p.Sex, &p.Age, &p.HeightCM, &p.WeightKG, &p.ActivityLevel, &p.Goal,
		&p.DailyCaloriesKCal, &p.DailyProteinG, &p.DailyFatG, &p.DailyCarbsG, &p.FormulaVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	if err != nil {
		return domain.Profile{}, fmt.Errorf("get profile: %w", err)
	}
	return p, nil
}
