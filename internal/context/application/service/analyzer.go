package service

import (
	"context"

	"ojirun/internal/context/domain"
)

type Analyzer interface {
	AnalyzeNutrition(ctx context.Context, req domain.AnalyzeRequest) (domain.AIResult[domain.NutritionAnalysis], error)
	DailyRecommendation(ctx context.Context, req domain.RecommendationRequest) (domain.AIResult[domain.DailyRecommendation], error)
}
