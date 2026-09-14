package domain

import "testing"

func TestNormalizeClampsBounds(t *testing.T) {
	in := NutritionAnalysis{
		Confidence: 2.5,
		Items: []NutritionItem{
			{Name: "x", CaloriesKCal: -100, ProteinG: 99999, FatG: -5, CarbsG: 10, EstimatedWeightG: -1, Confidence: -0.3},
		},
		Total: NutritionTotal{CaloriesKCal: 999999, ProteinG: -10, FatG: 20, CarbsG: 30},
	}

	out := in.Normalize()

	if out.Confidence != 1 {
		t.Errorf("confidence not clamped to 1: got %v", out.Confidence)
	}
	item := out.Items[0]
	if item.CaloriesKCal != 0 {
		t.Errorf("negative calories not clamped: got %v", item.CaloriesKCal)
	}
	if item.ProteinG != maxMacroG {
		t.Errorf("protein not capped: got %v", item.ProteinG)
	}
	if item.FatG != 0 || item.EstimatedWeightG != 0 {
		t.Errorf("negatives not clamped: fat=%v weight=%v", item.FatG, item.EstimatedWeightG)
	}
	if item.Confidence != 0 {
		t.Errorf("negative confidence not clamped: got %v", item.Confidence)
	}
	if out.Total.CaloriesKCal != maxCaloriesKCal {
		t.Errorf("total calories not capped: got %v", out.Total.CaloriesKCal)
	}
	if out.Total.ProteinG != 0 {
		t.Errorf("negative total protein not clamped: got %v", out.Total.ProteinG)
	}
}

func TestNormalizeDoesNotMutateInput(t *testing.T) {
	in := NutritionAnalysis{
		Items: []NutritionItem{{Name: "x", CaloriesKCal: -100}},
	}
	_ = in.Normalize()
	if in.Items[0].CaloriesKCal != -100 {
		t.Errorf("input mutated: got %v", in.Items[0].CaloriesKCal)
	}
}

func TestHasFood(t *testing.T) {
	cases := []struct {
		name string
		in   NutritionAnalysis
		want bool
	}{
		{"recognized with items", NutritionAnalysis{Recognized: true, Items: []NutritionItem{{Name: "x"}}}, true},
		{"recognized but no items", NutritionAnalysis{Recognized: true}, false},
		{"not recognized with items", NutritionAnalysis{Items: []NutritionItem{{Name: "x"}}}, false},
		{"empty", NutritionAnalysis{}, false},
	}
	for _, tt := range cases {
		if got := tt.in.HasFood(); got != tt.want {
			t.Errorf("%s: HasFood = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTotalsConsistent(t *testing.T) {
	a := NutritionAnalysis{
		Items: []NutritionItem{{CaloriesKCal: 300}, {CaloriesKCal: 200}},
		Total: NutritionTotal{CaloriesKCal: 500},
	}
	if !a.TotalsConsistent(0.2) {
		t.Error("matching totals should be consistent")
	}

	a.Total.CaloriesKCal = 1000
	if a.TotalsConsistent(0.2) {
		t.Error("wildly off totals should be inconsistent")
	}
}

func TestRevisionRoundTripFallback(t *testing.T) {
	analysis := MealRevision{
		Summary:           "Омлет",
		TotalCaloriesKCal: 430,
		TotalProteinG:     24,
		TotalFatG:         21,
		TotalCarbsG:       32,
		Confidence:        0.8,
		Items: []MealItem{
			{Name: "Омлет", EstimatedWeightG: 180, CaloriesKCal: 430, ProteinG: 24, FatG: 21, CarbsG: 32, Confidence: 0.8},
		},
	}.Analysis()

	if analysis.Summary != "Омлет" {
		t.Fatalf("summary = %q, want %q", analysis.Summary, "Омлет")
	}
	if analysis.Total.CaloriesKCal != 430 || len(analysis.Items) != 1 || analysis.Items[0].Name != "Омлет" {
		t.Fatalf("unexpected reconstruction: %#v", analysis)
	}
}

func TestNewRevisionNormalizesViaCaller(t *testing.T) {
	rev := NewRevision("meal-1", "текст", "", NutritionAnalysis{
		Summary:    "Тест",
		Confidence: 1,
		Items:      []NutritionItem{{Name: "позиция", CaloriesKCal: 0, ProteinG: 10, Confidence: 1}},
		Total:      NutritionTotal{ProteinG: 10},
	}.Normalize(), nil)

	if rev.Confidence != 1 || rev.TotalCaloriesKCal != 0 || rev.MealEntryID != "meal-1" || rev.UserText != "текст" {
		t.Errorf("unexpected revision: %+v", rev)
	}
}
