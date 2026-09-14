package domain

import (
	"testing"
	"time"
)

func TestCanCorrect(t *testing.T) {
	cases := map[int]bool{0: true, 1: true, MaxCorrectionsPerEntry: true, MaxCorrectionsPerEntry + 1: false}
	for revision, want := range cases {
		if got := (MealRevision{Revision: revision}).CanCorrect(); got != want {
			t.Errorf("revision %d CanCorrect = %v, want %v", revision, got, want)
		}
	}
}

func TestEntryProcessing(t *testing.T) {
	now := time.Now()
	fresh := MealEntry{Status: StatusAnalyzing, CreatedAt: now.Add(-time.Minute)}
	stale := MealEntry{Status: StatusAnalyzing, CreatedAt: now.Add(-InFlightWindow - time.Second)}
	done := MealEntry{Status: StatusPending, CreatedAt: now}
	if !fresh.Processing(now) || stale.Processing(now) || done.Processing(now) {
		t.Fatalf("processing: fresh=%v stale=%v done=%v, want true false false", fresh.Processing(now), stale.Processing(now), done.Processing(now))
	}
}

func TestStatusInFlight(t *testing.T) {
	cases := map[Status]bool{
		StatusReceived: true, StatusDownloadingPhoto: true, StatusAnalyzing: true, StatusReanalyzing: true,
		StatusPending: false, StatusAwaitingFix: false, StatusAccepted: false, StatusDeleted: false, StatusFailed: false,
	}
	for status, want := range cases {
		if got := status.InFlight(); got != want {
			t.Errorf("%s InFlight = %v, want %v", status, got, want)
		}
	}
}
