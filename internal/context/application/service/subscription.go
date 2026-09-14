package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type ChatMemberStatusGetter interface {
	ChatMemberStatus(ctx context.Context, chat any, telegramUserID int64) (string, error)
}

type ChatMemberStatusFunc func(ctx context.Context, chat any, telegramUserID int64) (string, error)

func (f ChatMemberStatusFunc) ChatMemberStatus(ctx context.Context, chat any, telegramUserID int64) (string, error) {
	return f(ctx, chat, telegramUserID)
}

const (
	subscriptionCacheTTL       = 10 * time.Minute
	subscriptionCacheSweepSize = 10000
)

type Subscription struct {
	members ChatMemberStatusGetter
	channel any
	now     func() time.Time
	log     *slog.Logger

	mu        sync.Mutex
	confirmed map[int64]time.Time
}

func NewSubscription(members ChatMemberStatusGetter, channel any, log *slog.Logger) *Subscription {
	return &Subscription{
		members:   members,
		channel:   channel,
		now:       time.Now,
		log:       log,
		confirmed: make(map[int64]time.Time),
	}
}

func (s *Subscription) Enabled() bool {
	return s != nil && s.channel != nil
}

func (s *Subscription) IsSubscribed(ctx context.Context, telegramUserID int64) (bool, error) {
	if !s.Enabled() {
		return true, nil
	}
	if s.isConfirmed(telegramUserID) {
		return true, nil
	}
	return s.check(ctx, telegramUserID)
}

func (s *Subscription) Recheck(ctx context.Context, telegramUserID int64) (bool, error) {
	if !s.Enabled() {
		return true, nil
	}
	return s.check(ctx, telegramUserID)
}

func (s *Subscription) Verify(ctx context.Context, botUserID int64) error {
	if !s.Enabled() {
		return nil
	}
	status, err := s.members.ChatMemberStatus(ctx, s.channel, botUserID)
	if err != nil {
		return fmt.Errorf("read the bot's own channel membership: %w", err)
	}
	if status != "administrator" && status != "creator" {
		return fmt.Errorf("bot is %q in the channel, it must be an administrator", status)
	}
	return nil
}

func (s *Subscription) check(ctx context.Context, telegramUserID int64) (bool, error) {
	status, err := s.members.ChatMemberStatus(ctx, s.channel, telegramUserID)
	if err != nil {
		s.log.Error("check channel subscription failed, refusing the user", "error", err, "telegram_user_id", telegramUserID)
		return false, fmt.Errorf("check channel subscription: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !isChannelMember(status) {
		delete(s.confirmed, telegramUserID)
		return false, nil
	}
	now := s.now()
	if len(s.confirmed) >= subscriptionCacheSweepSize {
		for id, expires := range s.confirmed {
			if !expires.After(now) {
				delete(s.confirmed, id)
			}
		}
	}
	s.confirmed[telegramUserID] = now.Add(subscriptionCacheTTL)
	return true, nil
}

func (s *Subscription) isConfirmed(telegramUserID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expires, ok := s.confirmed[telegramUserID]
	return ok && expires.After(s.now())
}

func isChannelMember(status string) bool {
	switch status {
	case "creator", "administrator", "member", "restricted":
		return true
	default:
		return false
	}
}
