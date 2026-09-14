package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSubscriptionCachesOnlyConfirmations(t *testing.T) {
	ctx := context.Background()
	calls, status := 0, "left"
	subscription := NewSubscription(ChatMemberStatusFunc(func(context.Context, any, int64) (string, error) {
		calls++
		return status, nil
	}), "@channel", discardLogger())
	now := time.Now()
	subscription.now = func() time.Time { return now }

	first, second := subscription.IsSubscribed(ctx, 1), subscription.IsSubscribed(ctx, 1)
	if first || second || calls != 2 {
		t.Fatalf("a refusal must not be cached (calls=%d)", calls)
	}

	status = "member"
	first, second = subscription.IsSubscribed(ctx, 1), subscription.IsSubscribed(ctx, 1)
	if !first || !second || calls != 3 {
		t.Fatalf("a confirmation should be cached (calls=%d)", calls)
	}

	status = "left"
	now = now.Add(subscriptionCacheTTL + time.Second)
	if subscription.IsSubscribed(ctx, 1) || calls != 4 {
		t.Fatalf("an expired confirmation must be checked again (calls=%d)", calls)
	}

	status = "member"
	if !subscription.Recheck(ctx, 1) || calls != 5 {
		t.Fatalf("recheck must always ask Telegram (calls=%d)", calls)
	}
}

func TestSubscriptionOpenGate(t *testing.T) {
	ctx := context.Background()
	failing := NewSubscription(ChatMemberStatusFunc(func(context.Context, any, int64) (string, error) {
		return "", errors.New("telegram is down")
	}), "@channel", discardLogger())
	if !failing.IsSubscribed(ctx, 1) {
		t.Fatal("a Telegram failure should let the user through")
	}

	if disabled := NewSubscription(nil, nil, discardLogger()); disabled.Enabled() || !disabled.IsSubscribed(ctx, 1) {
		t.Fatal("no channel configured means no requirement")
	}
	var missing *Subscription
	if !missing.IsSubscribed(ctx, 1) {
		t.Fatal("a nil subscription is an open gate")
	}
}

func TestSubscriptionVerify(t *testing.T) {
	ctx := context.Background()
	for status, wantErr := range map[string]bool{"administrator": false, "creator": false, "member": true, "left": true} {
		subscription := NewSubscription(ChatMemberStatusFunc(func(context.Context, any, int64) (string, error) {
			return status, nil
		}), "@channel", discardLogger())
		if err := subscription.Verify(ctx, 99); (err != nil) != wantErr {
			t.Errorf("Verify with bot status %q: err=%v, want error %v", status, err, wantErr)
		}
	}
}
