package domain

import "testing"

func TestApplySettings(t *testing.T) {
	profile := Profile{
		UserID: "u1", Sex: "male", Age: 30, HeightCM: 180, WeightKG: 80,
		ActivityLevel: "moderate", Goal: "maintain",
		DailyCaloriesKCal: 2600, DailyProteinG: 128, DailyFatG: 81, DailyCarbsG: 350,
		FormulaVersion: FormulaVersion,
	}

	body := profile.Settings()
	body.Sex = "female"
	body.WeightKG = 70.5
	updated, err := profile.ApplySettings(body)
	if err != nil {
		t.Fatalf("apply body settings: %v", err)
	}
	if updated.Sex != "female" || updated.WeightKG != 70.5 || updated.Age != 30 {
		t.Fatalf("body fields not applied or others lost: %+v", updated)
	}
	if updated.FormulaVersion != FormulaVersion {
		t.Fatalf("unchanged targets must keep formula version, got %q", updated.FormulaVersion)
	}

	coach := profile.Settings()
	coach.CaloriesKCal = 2200
	coach.ProteinG = 160
	updated, err = profile.ApplySettings(coach)
	if err != nil {
		t.Fatalf("apply target settings: %v", err)
	}
	if updated.DailyCaloriesKCal != 2200 || updated.DailyProteinG != 160 || updated.FormulaVersion != ManualFormulaVersion {
		t.Fatalf("manual targets not applied: %+v", updated)
	}

	typo := profile.Settings()
	typo.CaloriesKCal = 26000
	if _, err := profile.ApplySettings(typo); err == nil {
		t.Fatal("out-of-range calories must be rejected")
	}
}
