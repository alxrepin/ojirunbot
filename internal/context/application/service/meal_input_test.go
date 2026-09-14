package service

import (
	"context"
	"testing"
)

func TestMealInputLifecycle(t *testing.T) {
	ctx := context.Background()
	input := NewMealInput(&memSessions{})

	_, stale, err := input.Begin(ctx, 7, 100)
	if err != nil || stale != 0 {
		t.Fatalf("first begin: stale=%d err=%v", stale, err)
	}
	if err := input.BindPrompt(ctx, 7, 100, 55); err != nil {
		t.Fatal(err)
	}

	sessionID, stale, err := input.Begin(ctx, 7, 100)
	if err != nil || stale != 55 {
		t.Fatalf("repeated /add should report the stale prompt 55, got %d (%v)", stale, err)
	}

	if ok, _ := input.Cancel(ctx, sessionID, 8); ok {
		t.Fatal("another user must not cancel the input mode")
	}
	if _, ok, _ := input.Claim(ctx, 7, 100); !ok {
		t.Fatal("the author's next message should claim the input mode")
	}
	if ok, _ := input.Cancel(ctx, sessionID, 7); ok {
		t.Fatal("a claimed input mode can no longer be cancelled")
	}
	if _, ok, _ := input.Claim(ctx, 7, 100); ok {
		t.Fatal("input mode must be claimed only once")
	}
}
