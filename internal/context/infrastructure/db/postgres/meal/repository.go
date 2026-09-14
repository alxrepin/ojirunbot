package meal

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) CreateEntryWithinLimit(ctx context.Context, userID string, chatID, sourceMessageID int64, mealDate, since time.Time, limit int) (domain.MealEntry, error) {
	var entry domain.MealEntry
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if limit > 0 {
			if _, err := tx.Exec(ctx, lockUserMealsQuery, userID); err != nil {
				return err
			}
			var created int
			if err := tx.QueryRow(ctx, countCreatedSinceQuery, userID, since).Scan(&created); err != nil {
				return err
			}
			if created >= limit {
				return domain.ErrDailyMealLimit
			}
		}
		var err error
		entry, err = scanCreatedEntry(tx.QueryRow(ctx, createEntryQuery, userID, chatID, sourceMessageID, mealDate))
		return err
	})
	if errors.Is(err, domain.ErrDailyMealLimit) {
		return domain.MealEntry{}, err
	}
	if err != nil {
		return domain.MealEntry{}, fmt.Errorf("create meal entry: %w", err)
	}
	return entry, nil
}

func (r *Repository) CountCreatedSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var created int
	if err := r.pool.QueryRow(ctx, countCreatedSinceQuery, userID, since).Scan(&created); err != nil {
		return 0, fmt.Errorf("count created meals: %w", err)
	}
	return created, nil
}

func scanCreatedEntry(row pgx.Row) (domain.MealEntry, error) {
	var entry domain.MealEntry
	var status string
	err := row.Scan(&entry.ID, &entry.UserID, &entry.ChatID, &entry.SourceMessageID,
		&entry.BotMessageID, &entry.EphemeralMessageID, &status, &entry.MealDate, &entry.CreatedAt)
	if err != nil {
		return domain.MealEntry{}, err
	}
	entry.Status = domain.Status(status)
	return entry, nil
}

func (r *Repository) SetBotMessage(ctx context.Context, mealID string, botMessageID int64) error {
	if _, err := r.pool.Exec(ctx, setBotMessageQuery, mealID, botMessageID); err != nil {
		return fmt.Errorf("set meal bot message: %w", err)
	}
	return nil
}

func (r *Repository) SetEphemeralMessage(ctx context.Context, mealID string, ephemeralMessageID int64) error {
	if _, err := r.pool.Exec(ctx, setEphemeralMessageQuery, mealID, ephemeralMessageID); err != nil {
		return fmt.Errorf("set meal ephemeral message: %w", err)
	}
	return nil
}

func (r *Repository) SetStatus(ctx context.Context, mealID string, status domain.Status) error {
	if _, err := r.pool.Exec(ctx, setStatusQuery, mealID, string(status)); err != nil {
		return fmt.Errorf("set meal status: %w", err)
	}
	return nil
}

func (r *Repository) SaveRevision(ctx context.Context, revision domain.MealRevision) (domain.MealRevision, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, insertRevisionQuery,
			revision.MealEntryID, revision.UserText, revision.CorrectionText, revision.AIRequestJSON, revision.AIResponseJSON,
			revision.Summary, revision.TotalCaloriesKCal, revision.TotalProteinG, revision.TotalFatG, revision.TotalCarbsG, revision.Confidence).
			Scan(&revision.ID, &revision.Revision, &revision.CreatedAt)
		if err != nil {
			return err
		}
		for _, item := range revision.Items {
			if _, err := tx.Exec(ctx, insertItemQuery,
				revision.ID, item.Name, item.EstimatedWeightG, item.CaloriesKCal, item.ProteinG, item.FatG, item.CarbsG, item.Confidence); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, markPendingQuery, revision.MealEntryID)
		return err
	})
	if err != nil {
		return domain.MealRevision{}, fmt.Errorf("save meal revision: %w", err)
	}
	return revision, nil
}

func (r *Repository) GetForAction(ctx context.Context, mealID string) (domain.MealEntry, domain.MealRevision, error) {
	var entry domain.MealEntry
	var revision domain.MealRevision
	var status string
	err := r.pool.QueryRow(ctx, selectForActionQuery, mealID).Scan(
		&entry.ID, &entry.UserID, &entry.TelegramUserID, &entry.ChatID, &entry.SourceMessageID,
		&entry.BotMessageID, &entry.EphemeralMessageID, &status, &entry.MealDate, &entry.CreatedAt,
		&revision.ID, &revision.Revision, &revision.UserText, &revision.CorrectionText, &revision.AIResponseJSON,
		&revision.Summary, &revision.TotalCaloriesKCal, &revision.TotalProteinG, &revision.TotalFatG, &revision.TotalCarbsG, &revision.Confidence, &revision.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MealEntry{}, domain.MealRevision{}, domain.ErrMealNotFound
	}
	if err != nil {
		return domain.MealEntry{}, domain.MealRevision{}, fmt.Errorf("get meal for action: %w", err)
	}
	entry.Status = domain.Status(status)
	revision.MealEntryID = entry.ID
	items, err := r.revisionItems(ctx, revision.ID)
	if err != nil {
		return domain.MealEntry{}, domain.MealRevision{}, err
	}
	revision.Items = items
	return entry, revision, nil
}

func (r *Repository) Accept(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, acceptQuery, mealID, telegramUserID)
	if err != nil {
		return false, fmt.Errorf("accept meal: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) AwaitCorrection(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, awaitCorrectionQuery, mealID, telegramUserID)
	if err != nil {
		return false, fmt.Errorf("await correction: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) CancelCorrection(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, cancelCorrectionQuery, mealID, telegramUserID)
	if err != nil {
		return false, fmt.Errorf("cancel correction: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) Delete(ctx context.Context, mealID string, telegramUserID int64) (bool, error) {
	var ok bool
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var status string
		err := tx.QueryRow(ctx, lockForDeleteQuery, mealID, telegramUserID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if status == "deleted" {
			ok = true
			return nil
		}
		if status == "accepted" {
			if _, err := tx.Exec(ctx, insertDeleteAuditQuery, mealID, status, telegramUserID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, markDeletedQuery, mealID); err != nil {
			return err
		}
		ok = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("delete meal: %w", err)
	}
	return ok, nil
}

func (r *Repository) FindByMessage(ctx context.Context, chatID, messageID int64) (domain.MealEntry, error) {
	var entry domain.MealEntry
	var status string
	err := r.pool.QueryRow(ctx, findByMessageQuery, chatID, messageID).Scan(
		&entry.ID, &entry.UserID, &entry.TelegramUserID, &entry.ChatID, &entry.SourceMessageID,
		&entry.BotMessageID, &entry.EphemeralMessageID, &status, &entry.MealDate, &entry.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MealEntry{}, domain.ErrMealNotFound
	}
	if err != nil {
		return domain.MealEntry{}, fmt.Errorf("find meal by message: %w", err)
	}
	entry.Status = domain.Status(status)
	return entry, nil
}

func (r *Repository) SetMealDate(ctx context.Context, mealID string, telegramUserID int64, date time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, setMealDateQuery, mealID, telegramUserID, date)
	if err != nil {
		return false, fmt.Errorf("set meal date: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) FindAwaitingCorrection(ctx context.Context, telegramUserID, chatID int64) (domain.MealEntry, domain.MealRevision, error) {
	var mealID string
	err := r.pool.QueryRow(ctx, findAwaitingCorrectionQuery, telegramUserID, chatID).Scan(&mealID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MealEntry{}, domain.MealRevision{}, domain.ErrMealNotFound
	}
	if err != nil {
		return domain.MealEntry{}, domain.MealRevision{}, fmt.Errorf("find awaiting correction: %w", err)
	}
	return r.GetForAction(ctx, mealID)
}

func (r *Repository) revisionItems(ctx context.Context, revisionID string) ([]domain.MealItem, error) {
	rows, err := r.pool.Query(ctx, selectItemsQuery, revisionID)
	if err != nil {
		return nil, fmt.Errorf("query revision items: %w", err)
	}
	defer rows.Close()

	var items []domain.MealItem
	for rows.Next() {
		var item domain.MealItem
		if err := rows.Scan(&item.Name, &item.EstimatedWeightG, &item.CaloriesKCal, &item.ProteinG, &item.FatG, &item.CarbsG, &item.Confidence); err != nil {
			return nil, fmt.Errorf("scan revision item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetEntry(ctx context.Context, mealID string) (domain.MealEntry, error) {
	var entry domain.MealEntry
	var status string
	err := r.pool.QueryRow(ctx, selectEntryQuery, mealID).Scan(
		&entry.ID, &entry.UserID, &entry.TelegramUserID, &entry.ChatID, &entry.SourceMessageID,
		&entry.BotMessageID, &entry.EphemeralMessageID, &status, &entry.MealDate, &entry.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MealEntry{}, domain.ErrMealNotFound
	}
	if err != nil {
		return domain.MealEntry{}, fmt.Errorf("get meal entry: %w", err)
	}
	entry.Status = domain.Status(status)
	return entry, nil
}
