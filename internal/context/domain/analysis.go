package domain

type NutritionAnalysis struct {
	Recognized bool            `json:"recognized"`
	Summary    string          `json:"summary"`
	Items      []NutritionItem `json:"items"`
	Total      NutritionTotal  `json:"total"`
	Confidence float64         `json:"confidence"`
	Notes      string          `json:"notes"`
}

type NutritionItem struct {
	Name             string  `json:"name"`
	EstimatedWeightG float64 `json:"estimated_weight_g"`
	CaloriesKCal     float64 `json:"calories_kcal"`
	ProteinG         float64 `json:"protein_g"`
	FatG             float64 `json:"fat_g"`
	CarbsG           float64 `json:"carbs_g"`
	Confidence       float64 `json:"confidence"`
}

type NutritionTotal struct {
	CaloriesKCal float64 `json:"calories_kcal"`
	ProteinG     float64 `json:"protein_g"`
	FatG         float64 `json:"fat_g"`
	CarbsG       float64 `json:"carbs_g"`
}

type DailyRecommendation struct {
	Summary      string   `json:"summary"`
	KeepDoing    []string `json:"keep_doing"`
	Improve      []string `json:"improve"`
	MacroAdvice  string   `json:"macro_advice"`
	FoodIdeas    []string `json:"food_ideas"`
	WarningNotes string   `json:"warning_notes"`
}

const (
	maxCaloriesKCal = 20000
	maxMacroG       = 5000
	maxWeightG      = 20000
)

func (a NutritionAnalysis) Normalize() NutritionAnalysis {
	a.Confidence = clamp01(a.Confidence)
	a.Total = a.Total.normalize()

	items := make([]NutritionItem, len(a.Items))
	for i, item := range a.Items {
		item.EstimatedWeightG = clamp(item.EstimatedWeightG, 0, maxWeightG)
		item.CaloriesKCal = clamp(item.CaloriesKCal, 0, maxCaloriesKCal)
		item.ProteinG = clamp(item.ProteinG, 0, maxMacroG)
		item.FatG = clamp(item.FatG, 0, maxMacroG)
		item.CarbsG = clamp(item.CarbsG, 0, maxMacroG)
		item.Confidence = clamp01(item.Confidence)
		items[i] = item
	}
	a.Items = items
	return a
}

func (t NutritionTotal) normalize() NutritionTotal {
	t.CaloriesKCal = clamp(t.CaloriesKCal, 0, maxCaloriesKCal)
	t.ProteinG = clamp(t.ProteinG, 0, maxMacroG)
	t.FatG = clamp(t.FatG, 0, maxMacroG)
	t.CarbsG = clamp(t.CarbsG, 0, maxMacroG)
	return t
}

func (a NutritionAnalysis) HasFood() bool {
	return a.Recognized && len(a.Items) > 0
}

func (a NutritionAnalysis) TotalsConsistent(tolerance float64) bool {
	var sum float64
	for _, item := range a.Items {
		sum += item.CaloriesKCal
	}
	diff := sum - a.Total.CaloriesKCal
	if diff < 0 {
		diff = -diff
	}
	allowed := tolerance * a.Total.CaloriesKCal
	if allowed < tolerance*sum {
		allowed = tolerance * sum
	}
	return diff <= allowed
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clamp01(v float64) float64 {
	return clamp(v, 0, 1)
}
