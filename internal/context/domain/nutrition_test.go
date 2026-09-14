package domain

import "testing"

func TestCalculateTargets(t *testing.T) {
	targets, err := CalculateTargets(ProfileInput{
		Sex:           SexMale,
		Age:           30,
		HeightCM:      180,
		WeightKG:      80,
		ActivityLevel: ActivityModerate,
		Goal:          GoalMaintain,
	})
	if err != nil {
		t.Fatal(err)
	}
	if targets.CaloriesKCal <= 0 || targets.ProteinG <= 0 || targets.FatG <= 0 || targets.CarbsG <= 0 {
		t.Fatalf("expected positive targets, got %+v", targets)
	}
}

func TestParseRussianValues(t *testing.T) {
	if sex, ok := ParseSex("мужской"); !ok || sex != SexMale {
		t.Fatalf("unexpected sex parse result: %q %v", sex, ok)
	}
	if activity, ok := ParseActivity("3"); !ok || activity != ActivityModerate {
		t.Fatalf("unexpected activity parse result: %q %v", activity, ok)
	}
	if goal, ok := ParseGoal("похудение"); !ok || goal != GoalLose {
		t.Fatalf("unexpected goal parse result: %q %v", goal, ok)
	}
}
