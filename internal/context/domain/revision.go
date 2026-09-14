package domain

import (
	"encoding/json"
	"time"
)

const MaxCorrectionsPerEntry = 5

type MealRevision struct {
	ID                string
	MealEntryID       string
	Revision          int
	UserText          string
	CorrectionText    string
	AIRequestJSON     []byte
	AIResponseJSON    []byte
	Summary           string
	TotalCaloriesKCal float64
	TotalProteinG     float64
	TotalFatG         float64
	TotalCarbsG       float64
	Confidence        float64
	Items             []MealItem
	CreatedAt         time.Time
}

type MealItem struct {
	Name             string
	EstimatedWeightG float64
	CaloriesKCal     float64
	ProteinG         float64
	FatG             float64
	CarbsG           float64
	Confidence       float64
}

func NewRevision(mealID, userText, correction string, analysis NutritionAnalysis, requestJSON []byte) MealRevision {
	items := make([]MealItem, 0, len(analysis.Items))
	for _, item := range analysis.Items {
		items = append(items, MealItem(item))
	}

	responseJSON, _ := json.Marshal(analysis)
	return MealRevision{
		MealEntryID:       mealID,
		UserText:          userText,
		CorrectionText:    correction,
		AIRequestJSON:     requestJSON,
		AIResponseJSON:    responseJSON,
		Summary:           analysis.Summary,
		TotalCaloriesKCal: analysis.Total.CaloriesKCal,
		TotalProteinG:     analysis.Total.ProteinG,
		TotalFatG:         analysis.Total.FatG,
		TotalCarbsG:       analysis.Total.CarbsG,
		Confidence:        analysis.Confidence,
		Items:             items,
	}
}

func (r MealRevision) CanCorrect() bool {
	return r.Revision <= MaxCorrectionsPerEntry
}

func (r MealRevision) Analysis() NutritionAnalysis {
	var analysis NutritionAnalysis
	if len(r.AIResponseJSON) > 0 && json.Unmarshal(r.AIResponseJSON, &analysis) == nil && analysis.Summary != "" {
		return analysis
	}

	items := make([]NutritionItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, NutritionItem(item))
	}

	return NutritionAnalysis{
		Summary: r.Summary,
		Items:   items,
		Total: NutritionTotal{
			CaloriesKCal: r.TotalCaloriesKCal,
			ProteinG:     r.TotalProteinG,
			FatG:         r.TotalFatG,
			CarbsG:       r.TotalCarbsG,
		},
		Confidence: r.Confidence,
	}
}
