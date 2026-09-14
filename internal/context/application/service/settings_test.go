package service

import (
	"context"
	"errors"
	"testing"

	"ojirun/internal/context/domain"
)

type settingsUsersStub struct{ user domain.User }

func (s settingsUsersStub) GetByTelegramID(context.Context, int64) (domain.User, error) {
	return s.user, nil
}

type settingsProfilesStub struct {
	profile domain.Profile
	saved   *domain.Profile
}

func (s *settingsProfilesStub) GetByUserID(context.Context, string) (domain.Profile, error) {
	return s.profile, nil
}

func (s *settingsProfilesStub) Save(_ context.Context, p domain.Profile) error {
	s.saved = &p
	return nil
}

func newSettingsFixture() (*Settings, *memSessions, *settingsProfilesStub) {
	sessions := &memSessions{}
	profiles := &settingsProfilesStub{profile: domain.Profile{
		UserID: "u1", Sex: "male", Age: 30, HeightCM: 180, WeightKG: 80,
		ActivityLevel: "moderate", Goal: "maintain",
		DailyCaloriesKCal: 2600, DailyProteinG: 128, DailyFatG: 81, DailyCarbsG: 350,
		FormulaVersion: domain.FormulaVersion,
	}}
	users := settingsUsersStub{user: domain.User{ID: "u1", TelegramUserID: 7}}
	return NewSettings(sessions, users, profiles), sessions, profiles
}

func TestSettingsFlowSavesCoachTargets(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()

	out, err := settings.Begin(ctx, 7, 7)
	if err != nil || out.Step != "sex" {
		t.Fatalf("begin = %+v, %v; want step sex", out, err)
	}

	answer := func(apply func(domain.Session) (SettingsOutcome, error)) SettingsOutcome {
		t.Helper()
		session, err := settings.ActiveSession(ctx, 7, 7)
		if err != nil {
			t.Fatalf("active session: %v", err)
		}
		out, err := apply(session)
		if err != nil {
			t.Fatalf("answer: %v", err)
		}
		return out
	}

	steps := []struct {
		apply    func(domain.Session) (SettingsOutcome, error)
		wantStep string
		invalid  bool
	}{
		{func(s domain.Session) (SettingsOutcome, error) { return settings.Keep(ctx, s, "sex") }, "weight", false},
		{func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "78,5 кг") }, "calories", false},
		{func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "много") }, "calories", true},
		{func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "2 200 ккал") }, "protein", false},
		{func(s domain.Session) (SettingsOutcome, error) { return settings.Keep(ctx, s, "protein") }, "fat", false},
		{func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "70") }, "carbs", false},
	}
	for i, step := range steps {
		out := answer(step.apply)
		if out.Step != step.wantStep || out.Invalid != step.invalid {
			t.Fatalf("step %d = %+v, want step %q invalid %v", i, out, step.wantStep, step.invalid)
		}
	}
	if profiles.saved != nil {
		t.Fatal("profile must not be saved before the last field")
	}

	out = answer(func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "250г") })
	if !out.Saved {
		t.Fatalf("last answer should save, got %+v", out)
	}
	saved := profiles.saved
	if saved == nil {
		t.Fatal("profile was not saved")
	}
	if saved.Sex != "male" || saved.WeightKG != 78.5 || saved.DailyCaloriesKCal != 2200 ||
		saved.DailyProteinG != 128 || saved.DailyFatG != 70 || saved.DailyCarbsG != 250 || saved.Age != 30 {
		t.Fatalf("unexpected saved profile: %+v", *saved)
	}
	if saved.FormulaVersion != domain.ManualFormulaVersion {
		t.Fatalf("formula version = %q, want manual", saved.FormulaVersion)
	}
	if _, err := settings.ActiveSession(ctx, 7, 7); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("session should be closed after save, got %v", err)
	}
}

func TestSettingsCancelLeavesProfile(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()
	if _, err := settings.Begin(ctx, 7, 7); err != nil {
		t.Fatal(err)
	}
	session, _ := settings.ActiveSession(ctx, 7, 7)
	out, err := settings.Choose(ctx, session, "sex", "female")
	if err != nil || out.Step != "weight" {
		t.Fatalf("choose = %+v, %v", out, err)
	}

	session, _ = settings.ActiveSession(ctx, 7, 7)
	out, err = settings.Cancel(ctx, session)
	if err != nil || !out.Cancelled {
		t.Fatalf("cancel = %+v, %v", out, err)
	}
	if profiles.saved != nil {
		t.Fatal("cancel must not save the profile")
	}
}

func TestSettingsStaleButtonIgnored(t *testing.T) {
	ctx := context.Background()
	settings, _, _ := newSettingsFixture()
	if _, err := settings.Begin(ctx, 7, 7); err != nil {
		t.Fatal(err)
	}
	session, _ := settings.ActiveSession(ctx, 7, 7)
	if out, err := settings.Keep(ctx, session, "weight"); err != nil || out != (SettingsOutcome{}) {
		t.Fatalf("stale keep = %+v, %v; want no-op", out, err)
	}
}

func TestParseAmount(t *testing.T) {
	cases := map[string]float64{"72": 72, "72,5 кг": 72.5, "2 100 ккал": 2100, "150г.": 150}
	for in, want := range cases {
		if got, ok := parseAmount(in); !ok || got != want {
			t.Errorf("parseAmount(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "много", "кг"} {
		if _, ok := parseAmount(in); ok {
			t.Errorf("parseAmount(%q) should fail", in)
		}
	}
}
