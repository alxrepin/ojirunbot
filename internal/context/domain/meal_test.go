package domain

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusReceived, StatusAnalyzing, true},
		{StatusReceived, StatusDownloadingPhoto, true},
		{StatusAnalyzing, StatusPending, true},
		{StatusPending, StatusAccepted, true},
		{StatusPending, StatusAwaitingFix, true},
		{StatusAwaitingFix, StatusReanalyzing, true},
		{StatusAwaitingFix, StatusPending, true},
		{StatusAccepted, StatusDeleted, true},
		{StatusReceived, StatusAccepted, false},
		{StatusDeleted, StatusAccepted, false},
		{StatusAccepted, StatusPending, false},
		{StatusFailed, StatusAnalyzing, false},
		{StatusPending, StatusReanalyzing, false},
	}
	for _, tt := range cases {
		if got := CanTransition(tt.from, tt.to); got != tt.want {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	if !StatusDeleted.IsTerminal() {
		t.Error("deleted should be terminal")
	}
	if StatusAccepted.IsTerminal() {
		t.Error("accepted should not be terminal (can still be deleted with audit)")
	}
}
