package domain

type AnalyzeRequest struct {
	Prompt       string
	UserText     string
	ImagePath    string
	ImageMime    string
	Correction   string
	PreviousJSON []byte
}

type RecommendationRequest struct {
	Prompt      string
	UserName    string
	ReportJSON  []byte
	TargetsJSON []byte
}

type AIResult[T any] struct {
	Value        T
	RequestJSON  []byte
	ResponseJSON []byte
}
