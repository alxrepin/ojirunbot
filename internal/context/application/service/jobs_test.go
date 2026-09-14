package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"ojirun/internal/context/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type memQueue struct {
	enqueued  []string
	completed []string
	failed    []string
	retried   map[string]time.Duration
	backlog   int
}

func (q *memQueue) Enqueue(_ context.Context, _, kind string, _ any, _ string, _ int64) error {
	q.enqueued = append(q.enqueued, kind)
	return nil
}

func (q *memQueue) Claim(context.Context, string, time.Duration) (domain.Job, bool, error) {
	return domain.Job{}, false, nil
}

func (q *memQueue) Complete(_ context.Context, jobID string) error {
	q.completed = append(q.completed, jobID)
	return nil
}

func (q *memQueue) Retry(_ context.Context, jobID string, delay time.Duration, _ string) error {
	if q.retried == nil {
		q.retried = map[string]time.Duration{}
	}
	q.retried[jobID] = delay
	return nil
}

func (q *memQueue) Fail(_ context.Context, jobID string, _ string) error {
	q.failed = append(q.failed, jobID)
	return nil
}

func (q *memQueue) Backlog(context.Context, string, int64) (int, error) { return q.backlog, nil }

type throttledError struct{}

func (throttledError) Error() string             { return "too many requests" }
func (throttledError) RetryAfter() time.Duration { return 7 * time.Second }

func TestRetryDelay(t *testing.T) {
	cases := []struct {
		name      string
		attempts  int
		err       error
		wantDelay time.Duration
		wantRetry bool
	}{
		{"first failure backs off", 1, errors.New("boom"), 30 * time.Second, true},
		{"backoff grows with attempts", 2, errors.New("boom"), 60 * time.Second, true},
		{"attempts exhausted", 3, errors.New("boom"), 0, false},
		{"permanent error", 1, permanentError{errors.New("bad payload")}, 0, false},
		{"rate limit waits as asked even past attempts", 5, throttledError{}, 7 * time.Second, true},
		{"rate limit gives up eventually", maxJobClaims, throttledError{}, 7 * time.Second, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := domain.Job{Attempts: tc.attempts, MaxAttempts: 3}
			delay, retry := retryDelay(job, tc.err)
			if delay != tc.wantDelay || retry != tc.wantRetry {
				t.Fatalf("retryDelay = (%v, %v), want (%v, %v)", delay, retry, tc.wantDelay, tc.wantRetry)
			}
		})
	}
}

func TestJobWorkersSettleJobs(t *testing.T) {
	cases := map[string]struct {
		handler JobHandler
		check   func(*memQueue) bool
	}{
		"success completes": {
			func(context.Context, domain.Job) error { return nil },
			func(q *memQueue) bool { return len(q.completed) == 1 },
		},
		"error retries": {
			func(context.Context, domain.Job) error { return errors.New("ai timeout") },
			func(q *memQueue) bool { return q.retried["j1"] == jobRetryBackoff },
		},
		"permanent error fails": {
			func(context.Context, domain.Job) error { return permanentError{errors.New("bad")} },
			func(q *memQueue) bool { return len(q.failed) == 1 },
		},
		"panic is retried, not fatal": {
			func(context.Context, domain.Job) error { panic("nil map") },
			func(q *memQueue) bool { return q.retried["j1"] == jobRetryBackoff },
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			queue := &memQueue{}
			workers := NewJobWorkers(queue, domain.QueueMeal, 1, time.Minute, tc.handler, discardLogger())
			workers.process(context.Background(), domain.Job{ID: "j1", Attempts: 1, MaxAttempts: 3})
			if !tc.check(queue) {
				t.Fatalf("unexpected settlement: %+v", queue)
			}
		})
	}
}

func TestJobWorkersGiveUpOnRunawayJob(t *testing.T) {
	queue := &memQueue{}
	ran := false
	workers := NewJobWorkers(queue, domain.QueueMeal, 1, time.Minute, func(context.Context, domain.Job) error {
		ran = true
		return nil
	}, discardLogger())
	workers.process(context.Background(), domain.Job{ID: "j1", Attempts: maxJobClaims + 1, MaxAttempts: 3})
	if ran || len(queue.failed) != 1 {
		t.Fatalf("a job claimed too often must fail without running (ran=%v, queue=%+v)", ran, queue)
	}
}
