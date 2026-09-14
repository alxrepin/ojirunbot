package service

import (
	"context"
	"errors"
	"slices"
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

func settingsAnswer(t *testing.T, settings *Settings, apply func(domain.Session) (SettingsOutcome, error)) SettingsOutcome {
	t.Helper()
	session, err := settings.ActiveSession(context.Background(), 7, 7)
	if err != nil {
		t.Fatalf("active session: %v", err)
	}
	out, err := apply(session)
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	return out
}

func beginSettings(t *testing.T, settings *Settings, choice string) SettingsOutcome {
	t.Helper()
	ctx := context.Background()
	out, err := settings.Begin(ctx, 7, 7)
	if err != nil || out.Step != SettingsMenuStep {
		t.Fatalf("begin = %+v, %v; want the menu", out, err)
	}
	return settingsAnswer(t, settings, func(s domain.Session) (SettingsOutcome, error) { return settings.Select(ctx, s, choice) })
}

func TestSettingsFlowSavesCoachTargets(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()

	if out := beginSettings(t, settings, SettingsChoiceAll); out.Step != "sex" || len(out.Fields) != 6 {
		t.Fatalf("select all = %+v; want step sex over all fields", out)
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
		out := settingsAnswer(t, settings, step.apply)
		if out.Step != step.wantStep || out.Invalid != step.invalid {
			t.Fatalf("step %d = %+v, want step %q invalid %v", i, out, step.wantStep, step.invalid)
		}
	}
	if profiles.saved != nil {
		t.Fatal("profile must not be saved before the last field")
	}

	out := settingsAnswer(t, settings, func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "250г") })
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

func TestSettingsSingleFieldSavesRightAway(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()

	if out := beginSettings(t, settings, "weight"); out.Step != "weight" || !slices.Equal(out.Fields, []string{"weight"}) {
		t.Fatalf("select weight = %+v", out)
	}
	out := settingsAnswer(t, settings, func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "79") })
	if !out.Saved || !slices.Equal(out.Fields, []string{"weight"}) {
		t.Fatalf("single field should save after one answer, got %+v", out)
	}
	saved := profiles.saved
	if saved == nil || saved.WeightKG != 79 || saved.DailyCaloriesKCal != 2600 || saved.FormulaVersion != domain.FormulaVersion {
		t.Fatalf("unexpected saved profile: %+v", saved)
	}
}

func TestSettingsTargetsChoiceSkipsBodyFields(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()

	if out := beginSettings(t, settings, SettingsChoiceTargets); out.Step != "calories" {
		t.Fatalf("select targets = %+v; want step calories", out)
	}
	answers := []func(domain.Session) (SettingsOutcome, error){
		func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "2000") },
		func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "150") },
		func(s domain.Session) (SettingsOutcome, error) { return settings.Keep(ctx, s, "fat") },
		func(s domain.Session) (SettingsOutcome, error) { return settings.SubmitText(ctx, s, "200") },
	}
	var out SettingsOutcome
	for _, apply := range answers {
		out = settingsAnswer(t, settings, apply)
	}
	if !out.Saved || profiles.saved == nil {
		t.Fatalf("targets flow should save after carbs, got %+v", out)
	}
	if p := *profiles.saved; p.Sex != "male" || p.WeightKG != 80 || p.DailyCaloriesKCal != 2000 ||
		p.DailyProteinG != 150 || p.DailyFatG != 81 || p.DailyCarbsG != 200 || p.FormulaVersion != domain.ManualFormulaVersion {
		t.Fatalf("unexpected saved profile: %+v", p)
	}
}

func TestSettingsMenuIgnoresTextAndUnknownChoices(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()
	if _, err := settings.Begin(ctx, 7, 7); err != nil {
		t.Fatal(err)
	}
	session, _ := settings.ActiveSession(ctx, 7, 7)

	out, err := settings.SubmitText(ctx, session, "80")
	if err != nil || !out.Invalid || out.Step != SettingsMenuStep {
		t.Fatalf("text on the menu = %+v, %v; want invalid menu", out, err)
	}
	if out, err := settings.Keep(ctx, session, SettingsMenuStep); err != nil || out.Step != "" || out.Saved {
		t.Fatalf("keep on the menu = %+v, %v; want no-op", out, err)
	}
	if out, err := settings.Select(ctx, session, "age"); err != nil || out.Step != "" {
		t.Fatalf("unknown choice = %+v, %v; want no-op", out, err)
	}
	if profiles.saved != nil {
		t.Fatal("the menu must not save the profile")
	}
}

func TestSettingsLegacySessionWalksAllFields(t *testing.T) {
	ctx := context.Background()
	settings, sessions, _ := newSettingsFixture()
	current := domain.ProfileSettings{Sex: "male", WeightKG: 80, CaloriesKCal: 2600, ProteinG: 128, FatG: 81, CarbsG: 350}
	if _, err := sessions.Start(ctx, domain.SessionSettings, 7, 7, "sex", settingsPayload{Current: current, Draft: current}); err != nil {
		t.Fatal(err)
	}
	out := settingsAnswer(t, settings, func(s domain.Session) (SettingsOutcome, error) { return settings.Keep(ctx, s, "sex") })
	if out.Step != "weight" || len(out.Fields) != 6 {
		t.Fatalf("legacy session = %+v; want step weight over all fields", out)
	}
}

func TestSettingsCancelLeavesProfile(t *testing.T) {
	ctx := context.Background()
	settings, _, profiles := newSettingsFixture()
	beginSettings(t, settings, SettingsChoiceAll)

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
	beginSettings(t, settings, SettingsChoiceAll)

	session, _ := settings.ActiveSession(ctx, 7, 7)
	if out, err := settings.Keep(ctx, session, "weight"); err != nil || out.Step != "" || out.Saved {
		t.Fatalf("stale keep = %+v, %v; want no-op", out, err)
	}
	if out, err := settings.Select(ctx, session, "weight"); err != nil || out.Step != "" {
		t.Fatalf("stale menu choice = %+v, %v; want no-op", out, err)
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
