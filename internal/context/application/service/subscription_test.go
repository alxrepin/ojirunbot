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
	subscribed := func(check func(context.Context, int64) (bool, error)) bool {
		t.Helper()
		ok, err := check(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return ok
	}

	first, second := subscribed(subscription.IsSubscribed), subscribed(subscription.IsSubscribed)
	if first || second || calls != 2 {
		t.Fatalf("a refusal must not be cached (calls=%d)", calls)
	}

	status = "member"
	first, second = subscribed(subscription.IsSubscribed), subscribed(subscription.IsSubscribed)
	if !first || !second || calls != 3 {
		t.Fatalf("a confirmation should be cached (calls=%d)", calls)
	}

	status = "left"
	now = now.Add(subscriptionCacheTTL + time.Second)
	if subscribed(subscription.IsSubscribed) || calls != 4 {
		t.Fatalf("an expired confirmation must be checked again (calls=%d)", calls)
	}

	status = "member"
	if !subscribed(subscription.Recheck) || calls != 5 {
		t.Fatalf("recheck must always ask Telegram (calls=%d)", calls)
	}
}

func TestSubscriptionClosedGateOnFailure(t *testing.T) {
	ctx := context.Background()
	failing := NewSubscription(ChatMemberStatusFunc(func(context.Context, any, int64) (string, error) {
		return "", errors.New("telegram is down")
	}), "@channel", discardLogger())
	if ok, err := failing.IsSubscribed(ctx, 1); ok || err == nil {
		t.Fatalf("a Telegram failure must refuse the user with an error, got %v %v", ok, err)
	}
	if ok, err := failing.Recheck(ctx, 1); ok || err == nil {
		t.Fatalf("recheck must fail the same way, got %v %v", ok, err)
	}

	disabled := NewSubscription(nil, nil, discardLogger())
	if disabled.Enabled() {
		t.Fatal("no channel configured means no requirement")
	}
	if ok, err := disabled.IsSubscribed(ctx, 1); !ok || err != nil {
		t.Fatalf("disabled gate = %v %v, want open", ok, err)
	}
	var missing *Subscription
	if ok, err := missing.IsSubscribed(ctx, 1); !ok || err != nil {
		t.Fatalf("nil gate = %v %v, want open", ok, err)
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
