package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ojirun/internal/context/domain"
)

type fakeReportRepo struct {
	created     bool
	meals       []domain.DailyMeal
	slotChatID  int64
	statuses    []string
	sentMessage int64
}

func (f *fakeReportRepo) EnqueueDailyReports(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (f *fakeReportRepo) ReportUser(context.Context, string) (domain.ReportUser, error) {
	return domain.ReportUser{User: domain.User{ID: "u1", TelegramUserID: 555, DisplayName: "Оля"}}, nil
}

func (f *fakeReportRepo) MealsForReport(context.Context, string, time.Time) ([]domain.DailyMeal, domain.DailyTotals, error) {
	return f.meals, domain.DailyTotals{CaloriesKCal: 500}, nil
}

func (f *fakeReportRepo) TryCreate(_ context.Context, _ string, chatID int64, _ time.Time) (bool, string, error) {
	f.slotChatID = chatID
	return f.created, "r1", nil
}

func (f *fakeReportRepo) SetStatus(_ context.Context, _ string, status string) error {
	f.statuses = append(f.statuses, status)
	return nil
}

func (f *fakeReportRepo) MarkSent(_ context.Context, _ string, botMessageID int64, _, _ []byte) error {
	f.sentMessage = botMessageID
	return nil
}

type fakeReportNotifier struct {
	sends  int
	chatID int64
	err    error
}

func (f *fakeReportNotifier) Send(_ context.Context, user domain.ReportUser, _ time.Time, _ []domain.DailyMeal, _ domain.DailyTotals, _ *domain.DailyRecommendation) (int64, error) {
	f.sends++
	f.chatID = user.TelegramUserID
	return 77, f.err
}

type fixedSubscribers bool

func (f fixedSubscribers) IsSubscribed(context.Context, int64) bool { return bool(f) }

type recommendingAnalyzer struct{}

func (recommendingAnalyzer) AnalyzeNutrition(context.Context, domain.AnalyzeRequest) (domain.AIResult[domain.NutritionAnalysis], error) {
	return domain.AIResult[domain.NutritionAnalysis]{}, nil
}

func (recommendingAnalyzer) DailyRecommendation(context.Context, domain.RecommendationRequest) (domain.AIResult[domain.DailyRecommendation], error) {
	return domain.AIResult[domain.DailyRecommendation]{Value: domain.DailyRecommendation{Summary: "хороший день"}}, nil
}

type blockedError struct{}

func (blockedError) Error() string   { return "bot was blocked by the user" }
func (blockedError) Permanent() bool { return true }

func reportJob(t *testing.T) domain.Job {
	t.Helper()
	payload, err := json.Marshal(reportJobPayload{UserID: "u1", ReportDate: "2026-09-13"})
	if err != nil {
		t.Fatal(err)
	}
	return domain.Job{Kind: domain.JobDailyReport, Payload: payload, Attempts: 1, MaxAttempts: 3}
}

func newTestReport(repo *fakeReportRepo, notifier *fakeReportNotifier, subscribed bool) *Report {
	return NewReport(repo, recommendingAnalyzer{}, notifier, fixedSubscribers(subscribed), "prompt", time.UTC, discardLogger())
}

func TestReportSendsToPrivateChat(t *testing.T) {
	repo := &fakeReportRepo{created: true, meals: []domain.DailyMeal{{Summary: "омлет", CaloriesKCal: 500}}}
	notifier := &fakeReportNotifier{}
	if err := newTestReport(repo, notifier, true).Handle(context.Background(), reportJob(t)); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if notifier.sends != 1 || notifier.chatID != 555 || repo.slotChatID != 555 || repo.sentMessage != 77 {
		t.Fatalf("report should go to the user's private chat: notifier=%+v repo=%+v", notifier, repo)
	}
}

func TestReportSkips(t *testing.T) {
	cases := map[string]struct {
		repo       *fakeReportRepo
		subscribed bool
		status     []string
	}{
		"already delivered": {&fakeReportRepo{created: false}, true, nil},
		"unsubscribed":      {&fakeReportRepo{created: true, meals: []domain.DailyMeal{{}}}, false, []string{domain.ReportSkippedUnsubscribed}},
		"meals removed":     {&fakeReportRepo{created: true}, true, []string{domain.ReportSkippedEmpty}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			notifier := &fakeReportNotifier{}
			if err := newTestReport(tc.repo, notifier, tc.subscribed).Handle(context.Background(), reportJob(t)); err != nil {
				t.Fatalf("Handle: %v", err)
			}
			if notifier.sends != 0 || len(tc.repo.statuses) != len(tc.status) || (len(tc.status) > 0 && tc.repo.statuses[0] != tc.status[0]) {
				t.Fatalf("sends=%d statuses=%v, want no send and %v", notifier.sends, tc.repo.statuses, tc.status)
			}
		})
	}
}

func TestReportSendFailureReleasesSlot(t *testing.T) {
	repo := &fakeReportRepo{created: true, meals: []domain.DailyMeal{{}}}
	job := reportJob(t)
	err := newTestReport(repo, &fakeReportNotifier{err: blockedError{}}, true).Handle(context.Background(), job)
	if err == nil || len(repo.statuses) != 1 || repo.statuses[0] != domain.ReportFailed {
		t.Fatalf("err=%v statuses=%v, want error and a released slot", err, repo.statuses)
	}
	if _, retry := retryDelay(job, err); retry {
		t.Fatal("a user who blocked the bot should not be retried")
	}
	if !errors.As(err, new(blockedError)) {
		t.Fatalf("send error should stay inspectable, got %v", err)
	}
}
