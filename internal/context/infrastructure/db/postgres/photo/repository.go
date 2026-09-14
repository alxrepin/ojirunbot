package photo

import (
	"context"
	"errors"
	"fmt"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const insertQuery = `
	INSERT INTO photos (
		meal_entry_id, telegram_file_id, telegram_file_unique_id,
		local_path, mime_type, file_size, sha256
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

const selectLatestQuery = `
	SELECT meal_entry_id::text, telegram_file_id, telegram_file_unique_id, local_path, mime_type, file_size, sha256
	FROM photos
	WHERE meal_entry_id=$1
	ORDER BY created_at DESC
	LIMIT 1`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(c *postgres.Client) *Repository {
	return &Repository{pool: c.Pool()}
}

func (r *Repository) Save(ctx context.Context, p domain.Photo) error {
	_, err := r.pool.Exec(ctx, insertQuery,
		p.MealEntryID, p.TelegramFileID, p.TelegramFileUniqueID, p.LocalPath, p.MimeType, p.FileSize, p.SHA256)
	if err != nil {
		return fmt.Errorf("save photo: %w", err)
	}
	return nil
}

func (r *Repository) GetByMeal(ctx context.Context, mealID string) (domain.Photo, error) {
	var p domain.Photo
	err := r.pool.QueryRow(ctx, selectLatestQuery, mealID).
		Scan(&p.MealEntryID, &p.TelegramFileID, &p.TelegramFileUniqueID, &p.LocalPath, &p.MimeType, &p.FileSize, &p.SHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Photo{}, domain.ErrPhotoNotFound
	}
	if err != nil {
		return domain.Photo{}, fmt.Errorf("get meal photo: %w", err)
	}
	return p, nil
}
