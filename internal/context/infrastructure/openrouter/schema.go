package openrouter

func nutritionAnalysisSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"recognized",
			"summary",
			"items",
			"total",
			"confidence",
			"notes",
		},
		"properties": map[string]any{
			"recognized": map[string]any{"type": "boolean"},
			"summary":    map[string]any{"type": "string"},
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required": []string{
						"name",
						"estimated_weight_g",
						"calories_kcal",
						"protein_g",
						"fat_g",
						"carbs_g",
						"confidence",
					},
					"properties": map[string]any{
						"name":               map[string]any{"type": "string"},
						"estimated_weight_g": map[string]any{"type": "number"},
						"calories_kcal":      map[string]any{"type": "number"},
						"protein_g":          map[string]any{"type": "number"},
						"fat_g":              map[string]any{"type": "number"},
						"carbs_g":            map[string]any{"type": "number"},
						"confidence":         map[string]any{"type": "number"},
					},
				},
			},
			"total": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"calories_kcal",
					"protein_g",
					"fat_g",
					"carbs_g",
				},
				"properties": map[string]any{
					"calories_kcal": map[string]any{"type": "number"},
					"protein_g":     map[string]any{"type": "number"},
					"fat_g":         map[string]any{"type": "number"},
					"carbs_g":       map[string]any{"type": "number"},
				},
			},
			"confidence": map[string]any{"type": "number"},
			"notes":      map[string]any{"type": "string"},
		},
	}
}

func dailyRecommendationSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"summary",
			"keep_doing",
			"improve",
			"macro_advice",
			"food_ideas",
			"warning_notes",
		},
		"properties": map[string]any{
			"summary":       map[string]any{"type": "string"},
			"keep_doing":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"improve":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"macro_advice":  map[string]any{"type": "string"},
			"food_ideas":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"warning_notes": map[string]any{"type": "string"},
		},
	}
}
