package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxErrorRunes = 1000

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(c *postgres.Client) *Repository {
	return &Repository{pool: c.Pool()}
}

func (r *Repository) Enqueue(ctx context.Context, queue, kind string, payload any, dedupeKey string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s job payload: %w", kind, err)
	}
	if _, err := r.pool.Exec(ctx, enqueueQuery, queue, kind, dedupeKey, payloadJSON); err != nil {
		return fmt.Errorf("enqueue %s job: %w", kind, err)
	}
	return nil
}

func (r *Repository) Claim(ctx context.Context, queue string, lease time.Duration) (domain.Job, bool, error) {
	var job domain.Job
	err := r.pool.QueryRow(ctx, claimQuery, queue, lease.Seconds()).
		Scan(&job.ID, &job.Queue, &job.Kind, &job.Payload, &job.Attempts, &job.MaxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, false, nil
	}
	if err != nil {
		return domain.Job{}, false, fmt.Errorf("claim %s job: %w", queue, err)
	}
	return job, true, nil
}

func (r *Repository) Complete(ctx context.Context, jobID string) error {
	if _, err := r.pool.Exec(ctx, completeQuery, jobID); err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

func (r *Repository) Retry(ctx context.Context, jobID string, delay time.Duration, reason string) error {
	if _, err := r.pool.Exec(ctx, retryQuery, jobID, delay.Seconds(), truncateRunes(reason)); err != nil {
		return fmt.Errorf("retry job: %w", err)
	}
	return nil
}

func (r *Repository) Fail(ctx context.Context, jobID, reason string) error {
	if _, err := r.pool.Exec(ctx, failQuery, jobID, truncateRunes(reason)); err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

func (r *Repository) Backlog(ctx context.Context, queue string) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, backlogQuery, queue).Scan(&count); err != nil {
		return 0, fmt.Errorf("count %s backlog: %w", queue, err)
	}
	return count, nil
}

func truncateRunes(value string) string {
	runes := []rune(value)
	if len(runes) <= maxErrorRunes {
		return value
	}
	return string(runes[:maxErrorRunes])
}
