package telegram

import (
	"context"
	"errors"
	"strings"

	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

func (r *Router) handleStats(ctx context.Context, msg tg.Message, args string) {
	if !r.requireSubscriber(ctx, msg) {
		return
	}
	inGroup := msg.Chat.Type == "group" || msg.Chat.Type == "supergroup"
	username := ""
	if inGroup {
		username = parseUsernameArg(args)
	}

	var (
		result usecase.StatsResult
		err    error
	)
	if username != "" {
		result, err = r.stats.ForUsername(ctx, username, msg.Chat.ID)
	} else {
		result, err = r.stats.ForSelf(ctx, msg.From.ID)
	}
	if err != nil {
		r.replyStatsError(ctx, msg, inGroup, username, err)
		return
	}

	text := render.StatsCard(result.DisplayName, result.MealCount, result.Totals, result.Profile)
	_, _ = r.api.SendMessage(ctx, msg.Chat.ID, text, statsReplyOptions(msg))
}

func (r *Router) replyStatsError(ctx context.Context, msg tg.Message, inGroup bool, username string, err error) {
	switch {
	case username != "" && errors.Is(err, domain.ErrUserNotFound):
		_, _ = r.notify(ctx, msg, "Не нашёл @"+username+" среди тех, кто ведёт дневник в этом чате.", nil)
	case username != "" && errors.Is(err, domain.ErrProfileNotFound):
		_, _ = r.notify(ctx, msg, "У @"+username+" не завершена регистрация.", nil)
	case errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrProfileNotFound):
		if inGroup {
			_, _ = r.notify(ctx, msg, render.NeedRegister(r.botUsername), htmlOptions())
		} else {
			_, _ = r.notify(ctx, msg, render.NeedRegisterPrivate(), nil)
		}
	default:
		r.log.Error("stats lookup failed", "error", err)
	}
}

func parseUsernameArg(args string) string {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimPrefix(fields[0], "@")
}

func statsReplyOptions(msg tg.Message) *tg.SendOptions {
	opts := replyToMessage(msg, htmlOptions())
	if msg.Chat.Type == "private" {
		opts = withMenu(opts)
	}
	return opts
}
