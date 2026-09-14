package domain

import "testing"

func TestCanCorrect(t *testing.T) {
	cases := map[int]bool{0: true, 1: true, MaxCorrectionsPerEntry: true, MaxCorrectionsPerEntry + 1: false}
	for revision, want := range cases {
		if got := (MealRevision{Revision: revision}).CanCorrect(); got != want {
			t.Errorf("revision %d CanCorrect = %v, want %v", revision, got, want)
		}
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
