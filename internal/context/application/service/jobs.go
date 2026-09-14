package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ojirun/internal/context/domain"
)

type jobQueue interface {
	Enqueue(ctx context.Context, queue, kind string, payload any, dedupeKey string) error
	Claim(ctx context.Context, queue string, lease time.Duration) (domain.Job, bool, error)
	Complete(ctx context.Context, jobID string) error
	Retry(ctx context.Context, jobID string, delay time.Duration, reason string) error
	Fail(ctx context.Context, jobID string, reason string) error
	Backlog(ctx context.Context, queue string) (int, error)
}

type JobHandler func(ctx context.Context, job domain.Job) error

const (
	jobPollInterval = time.Second
	jobRetryBackoff = 30 * time.Second
	jobSettleTimout = 10 * time.Second
	maxJobClaims    = 20
)

type JobWorkers struct {
	queue   jobQueue
	name    string
	workers int
	lease   time.Duration
	handler JobHandler
	wake    chan struct{}
	log     *slog.Logger
}

func NewJobWorkers(queue jobQueue, name string, workers int, lease time.Duration, handler JobHandler, log *slog.Logger) *JobWorkers {
	if workers < 1 {
		workers = 1
	}
	return &JobWorkers{
		queue:   queue,
		name:    name,
		workers: workers,
		lease:   lease,
		handler: handler,
		wake:    make(chan struct{}, 1),
		log:     log.With("queue", name),
	}
}

func (w *JobWorkers) Wake() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *JobWorkers) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.loop(ctx)
		}()
	}
	wg.Wait()
}

func (w *JobWorkers) loop(ctx context.Context) {
	for ctx.Err() == nil {
		job, ok, err := w.queue.Claim(ctx, w.name, w.lease)
		if err != nil && ctx.Err() == nil {
			w.log.Error("claim job failed", "error", err)
		}
		if err != nil || !ok {
			w.idle(ctx)
			continue
		}
		w.process(ctx, job)
	}
}

func (w *JobWorkers) idle(ctx context.Context) {
	timer := time.NewTimer(jobPollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-w.wake:
	case <-timer.C:
	}
}

func (w *JobWorkers) process(ctx context.Context, job domain.Job) {
	var err error
	if job.Attempts > maxJobClaims {
		err = permanentError{fmt.Errorf("job claimed %d times without finishing", job.Attempts)}
	} else {
		jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.lease)
		err = w.handle(jobCtx, job)
		cancel()
	}

	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jobSettleTimout)
	defer cancel()
	log := w.log.With("kind", job.Kind, "job_id", job.ID, "attempts", job.Attempts)

	if err == nil {
		if err := w.queue.Complete(settleCtx, job.ID); err != nil {
			log.Error("complete job failed", "error", err)
		}
		return
	}
	delay, retry := retryDelay(job, err)
	if !retry {
		log.Error("job failed permanently", "error", err)
		if err := w.queue.Fail(settleCtx, job.ID, err.Error()); err != nil {
			log.Error("mark job failed failed", "error", err)
		}
		return
	}
	log.Warn("job failed, retrying", "error", err, "retry_in", delay)
	if err := w.queue.Retry(settleCtx, job.ID, delay, err.Error()); err != nil {
		log.Error("reschedule job failed", "error", err)
	}
}

func (w *JobWorkers) handle(ctx context.Context, job domain.Job) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("job panicked: %v", rec)
		}
	}()
	return w.handler(ctx, job)
}

func retryDelay(job domain.Job, err error) (time.Duration, bool) {
	var throttled interface{ RetryAfter() time.Duration }
	if errors.As(err, &throttled) && throttled.RetryAfter() > 0 {
		return throttled.RetryAfter(), job.Attempts < maxJobClaims
	}
	var permanent interface{ Permanent() bool }
	if errors.As(err, &permanent) && permanent.Permanent() {
		return 0, false
	}
	if job.Attempts >= job.MaxAttempts {
		return 0, false
	}
	return time.Duration(job.Attempts) * jobRetryBackoff, true
}

type permanentError struct{ err error }

func (e permanentError) Error() string   { return e.err.Error() }
func (e permanentError) Unwrap() error   { return e.err }
func (e permanentError) Permanent() bool { return true }

func decodeJob(job domain.Job, into any) error {
	if err := json.Unmarshal(job.Payload, into); err != nil {
		return permanentError{fmt.Errorf("decode %s job payload: %w", job.Kind, err)}
	}
	return nil
}
