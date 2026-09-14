package telegram

import (
	"context"
	"log/slog"
	"time"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/presentation/telegram/render"
)

type ReportNotifier struct {
	api API
	log *slog.Logger
}

func NewReportNotifier(api API, log *slog.Logger) *ReportNotifier {
	return &ReportNotifier{api: api, log: log}
}

func (n *ReportNotifier) Send(ctx context.Context, user domain.ReportUser, reportDate time.Time, meals []domain.DailyMeal, totals domain.DailyTotals, rec *domain.DailyRecommendation) (int64, error) {
	chatID := user.TelegramUserID
	markdown := render.ReportMarkdown(user, reportDate, meals, totals, rec)
	sent, err := n.api.SendRichMarkdown(ctx, chatID, markdown, nil)
	if err != nil {
		n.log.Warn("send rich report failed, falling back", "error", err, "telegram_user_id", chatID)
		sent, err = n.api.SendMessage(ctx, chatID, render.ReportFallback(markdown), nil)
	}
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}
