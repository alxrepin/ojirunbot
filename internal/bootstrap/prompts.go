package bootstrap

import (
	"os"
	"path/filepath"
)

type Prompts struct {
	NutritionAnalysis   string
	NutritionCorrection string
	DailyRecommendation string
}

func LoadPrompts(dir string) (Prompts, error) {
	read := func(name string) (string, error) {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}

	analysis, err := read("nutrition_analysis.md")
	if err != nil {
		return Prompts{}, err
	}
	correction, err := read("nutrition_correction.md")
	if err != nil {
		return Prompts{}, err
	}
	recommendation, err := read("daily_recommendation.md")
	if err != nil {
		return Prompts{}, err
	}

	return Prompts{
		NutritionAnalysis:   analysis,
		NutritionCorrection: correction,
		DailyRecommendation: recommendation,
	}, nil
}
