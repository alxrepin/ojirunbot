package report

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

func (r *Repository) EnqueueDailyReports(ctx context.Context, reportDate time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, enqueueDailyReportsQuery, reportDate, domain.QueueReport, domain.JobDailyReport)
	if err != nil {
		return 0, fmt.Errorf("enqueue daily reports: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) ReportUser(ctx context.Context, userID string) (domain.ReportUser, error) {
	var item domain.ReportUser
	err := r.pool.QueryRow(ctx, reportUserQuery, userID).Scan(
		&item.ID, &item.TelegramUserID, &item.Username, &item.DisplayName,
		&item.Profile.UserID, &item.Profile.Sex, &item.Profile.Age, &item.Profile.HeightCM, &item.Profile.WeightKG,
		&item.Profile.ActivityLevel, &item.Profile.Goal, &item.Profile.DailyCaloriesKCal, &item.Profile.DailyProteinG,
		&item.Profile.DailyFatG, &item.Profile.DailyCarbsG, &item.Profile.FormulaVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReportUser{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.ReportUser{}, fmt.Errorf("get report user: %w", err)
	}
	return item, nil
}

func (r *Repository) MealsForReport(ctx context.Context, userID string, date time.Time) ([]domain.DailyMeal, domain.DailyTotals, error) {
	rows, err := r.pool.Query(ctx, mealsForReportQuery, userID, date)
	if err != nil {
		return nil, domain.DailyTotals{}, fmt.Errorf("query meals for report: %w", err)
	}
	defer rows.Close()

	var meals []domain.DailyMeal
	var totals domain.DailyTotals
	for rows.Next() {
		var meal domain.DailyMeal
		if err := rows.Scan(&meal.CreatedAt, &meal.Summary, &meal.CaloriesKCal, &meal.ProteinG, &meal.FatG, &meal.CarbsG); err != nil {
			return nil, domain.DailyTotals{}, fmt.Errorf("scan daily meal: %w", err)
		}
		totals.CaloriesKCal += meal.CaloriesKCal
		totals.ProteinG += meal.ProteinG
		totals.FatG += meal.FatG
		totals.CarbsG += meal.CarbsG
		meals = append(meals, meal)
	}
	return meals, totals, rows.Err()
}

func (r *Repository) HasMealsInChat(ctx context.Context, userID string, chatID int64) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, hasMealsInChatQuery, userID, chatID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check meals in chat: %w", err)
	}
	return exists, nil
}

func (r *Repository) TryCreate(ctx context.Context, userID string, chatID int64, reportDate time.Time) (created bool, reportID string, err error) {
	err = r.pool.QueryRow(ctx, tryCreateQuery, userID, chatID, reportDate).Scan(&reportID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("try create daily report: %w", err)
	}
	return true, reportID, nil
}

func (r *Repository) SetStatus(ctx context.Context, reportID, status string) error {
	if _, err := r.pool.Exec(ctx, setStatusQuery, reportID, status); err != nil {
		return fmt.Errorf("set daily report status: %w", err)
	}
	return nil
}

func (r *Repository) MarkSent(ctx context.Context, reportID string, botMessageID int64, requestJSON, responseJSON []byte) error {
	if _, err := r.pool.Exec(ctx, markSentQuery, reportID, botMessageID, requestJSON, responseJSON); err != nil {
		return fmt.Errorf("save daily report sent: %w", err)
	}
	return nil
}
