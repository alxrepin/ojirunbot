package service

import (
	"context"
	"encoding/json"
	"testing"

	"ojirun/internal/context/domain"
)

type stubMealLoader struct{ entry domain.MealEntry }

func (s stubMealLoader) GetEntry(context.Context, string) (domain.MealEntry, error) {
	return s.entry, nil
}

func (s stubMealLoader) GetForAction(context.Context, string) (domain.MealEntry, domain.MealRevision, error) {
	return s.entry, domain.MealRevision{}, nil
}

func TestMealJobsSkipSettledEntries(t *testing.T) {
	cases := map[string]struct {
		kind   string
		status domain.Status
	}{
		"analysis of an accepted meal": {domain.JobMealAnalysis, domain.StatusAccepted},
		"analysis of a deleted meal":   {domain.JobMealAnalysis, domain.StatusDeleted},
		"correction after cancel":      {domain.JobMealCorrection, domain.StatusPending},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			jobs := NewMealJobs(&memQueue{}, stubMealLoader{entry: domain.MealEntry{ID: "m1", Status: tc.status}}, nil, nil)
			payload, _ := json.Marshal(map[string]string{"meal_id": "m1"})
			if err := jobs.Handle(context.Background(), domain.Job{Kind: tc.kind, Payload: payload}); err != nil {
				t.Fatalf("Handle: %v", err)
			}
		})
	}
}

func TestMealJobsRejectMalformedJobs(t *testing.T) {
	jobs := NewMealJobs(&memQueue{}, stubMealLoader{}, nil, nil)
	for _, job := range []domain.Job{
		{Kind: "unknown"},
		{Kind: domain.JobMealAnalysis, Payload: []byte("{")},
	} {
		if _, retry := retryDelay(job, jobs.Handle(context.Background(), job)); retry {
			t.Errorf("job %+v should fail permanently", job)
		}
	}
}

func TestMealJobsEnqueueWakesWorkers(t *testing.T) {
	queue := &memQueue{}
	jobs := NewMealJobs(queue, nil, nil, nil)
	woken := 0
	jobs.OnEnqueue(func() { woken++ })

	ctx := context.Background()
	if err := jobs.EnqueueAnalysis(ctx, "m1", domain.PhotoRef{}, "омлет"); err != nil {
		t.Fatal(err)
	}
	if err := jobs.EnqueueCorrection(ctx, "m1", "без масла"); err != nil {
		t.Fatal(err)
	}
	if woken != 2 || len(queue.enqueued) != 2 || queue.enqueued[1] != domain.JobMealCorrection {
		t.Fatalf("woken=%d enqueued=%v", woken, queue.enqueued)
	}
}
