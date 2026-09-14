package openrouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/httpx"
)

type Client struct {
	cfg  Config
	http *http.Client
	log  *slog.Logger
}

func NewClient(cfg Config) (*Client, error) {
	if len(cfg.Models) == 0 {
		return nil, fmt.Errorf("openrouter: at least one model is required")
	}
	httpClient, err := httpx.NewClient(90*time.Second, cfg.ProxyURL)
	if err != nil {
		return nil, err
	}
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Client{
		cfg:  cfg,
		http: httpClient,
		log:  log,
	}, nil
}

func (c *Client) AnalyzeNutrition(ctx context.Context, input domain.AnalyzeRequest) (domain.AIResult[domain.NutritionAnalysis], error) {
	content := []map[string]any{
		{
			"type": "text",
			"text": buildAnalysisText(input),
		},
	}
	if input.ImagePath != "" {
		dataURL, err := imageDataURL(input.ImagePath, input.ImageMime)
		if err != nil {
			return domain.AIResult[domain.NutritionAnalysis]{}, err
		}
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url": dataURL,
			},
		})
	}

	payload := map[string]any{
		"messages": []map[string]any{
			{
				"role":    "system",
				"content": input.Prompt,
			},
			{
				"role":    "user",
				"content": content,
			},
		},
		"response_format": jsonSchemaResponseFormat("nutrition_analysis", nutritionAnalysisSchema()),
		"stream":          false,
	}

	return doStructured[domain.NutritionAnalysis](ctx, c, payload)
}

func (c *Client) DailyRecommendation(ctx context.Context, input domain.RecommendationRequest) (domain.AIResult[domain.DailyRecommendation], error) {
	text := strings.Builder{}
	text.WriteString("Пользователь: ")
	text.WriteString(input.UserName)
	text.WriteString("\n\nЦели и нормы JSON:\n")
	text.Write(input.TargetsJSON)
	text.WriteString("\n\nОтчет за вчера JSON:\n")
	text.Write(input.ReportJSON)

	payload := map[string]any{
		"messages": []map[string]any{
			{"role": "system", "content": input.Prompt},
			{"role": "user", "content": text.String()},
		},
		"response_format": jsonSchemaResponseFormat("daily_recommendation", dailyRecommendationSchema()),
		"stream":          false,
	}

	return doStructured[domain.DailyRecommendation](ctx, c, payload)
}

const maxAttempts = 3

type attemptOutcome int

const (
	outcomeModelFailure attemptOutcome = iota
	outcomeContentFailure
	outcomeFatal
)

func doStructured[T any](ctx context.Context, c *Client, payload map[string]any) (domain.AIResult[T], error) {
	models := c.cfg.Models
	totalAttempts := max(maxAttempts, len(models))

	modelIdx := 0
	payload["model"] = models[modelIdx]
	requestJSON, err := json.Marshal(payload)
	if err != nil {
		return domain.AIResult[T]{}, err
	}

	var attemptErrs []error
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		if attempt > 1 {
			if err := sleepBackoff(ctx, attempt); err != nil {
				return domain.AIResult[T]{RequestJSON: requestJSON}, err
			}
		}
		result, outcome, err := doAttempt[T](ctx, c, requestJSON)
		if err == nil {
			return result, nil
		}
		attemptErr := fmt.Errorf("attempt %d, model %s: %w", attempt, models[modelIdx], err)
		attemptErrs = append(attemptErrs, attemptErr)
		if outcome == outcomeFatal {
			return result, attemptErr
		}
		if outcome == outcomeModelFailure && len(models) > 1 {
			next := (modelIdx + 1) % len(models)
			c.log.WarnContext(ctx, "openrouter attempt failed, switching model",
				"attempt", attempt, "model", models[modelIdx], "next_model", models[next], "error", err)
			payload["model"] = models[next]
			requestJSON, err = json.Marshal(payload)
			if err != nil {
				return domain.AIResult[T]{}, err
			}
			modelIdx = next
		} else {
			c.log.WarnContext(ctx, "openrouter attempt failed, retrying same model",
				"attempt", attempt, "model", models[modelIdx], "error", err)
		}
	}
	return domain.AIResult[T]{RequestJSON: requestJSON}, fmt.Errorf("openrouter failed after %d attempts: %w", totalAttempts, errors.Join(attemptErrs...))
}

func doAttempt[T any](ctx context.Context, c *Client, requestJSON []byte) (domain.AIResult[T], attemptOutcome, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(requestJSON))
	if err != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON}, outcomeFatal, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.HTTPReferer != "" {
		req.Header.Set("HTTP-Referer", c.cfg.HTTPReferer)
	}
	if c.cfg.AppTitle != "" {
		req.Header.Set("X-Title", c.cfg.AppTitle)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON}, failureOutcome(ctx, outcomeModelFailure), err
	}
	defer resp.Body.Close()

	responseJSON, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON}, failureOutcome(ctx, outcomeModelFailure), err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.AIResult[T]{RequestJSON: requestJSON, ResponseJSON: responseJSON}, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter status %s: %s", resp.Status, string(responseJSON))
	}

	var envelope struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
			Error        *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseJSON, &envelope); err != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON, ResponseJSON: responseJSON}, failureOutcome(ctx, outcomeModelFailure), err
	}
	if envelope.Error != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON, ResponseJSON: responseJSON}, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Choices) == 0 {
		return domain.AIResult[T]{RequestJSON: requestJSON, ResponseJSON: responseJSON}, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter returned no choices; body=%s", string(responseJSON))
	}
	if envelope.Choices[0].Error != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON, ResponseJSON: responseJSON}, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter choice error %d: %s", envelope.Choices[0].Error.Code, envelope.Choices[0].Error.Message)
	}
	var value T
	if err := json.Unmarshal([]byte(envelope.Choices[0].Message.Content), &value); err != nil {
		return domain.AIResult[T]{RequestJSON: requestJSON, ResponseJSON: responseJSON}, failureOutcome(ctx, outcomeContentFailure), fmt.Errorf("decode structured content: %w; content=%s", err, envelope.Choices[0].Message.Content)
	}
	return domain.AIResult[T]{Value: value, RequestJSON: requestJSON, ResponseJSON: responseJSON}, outcomeModelFailure, nil
}

func failureOutcome(ctx context.Context, kind attemptOutcome) attemptOutcome {
	if ctx.Err() != nil {
		return outcomeFatal
	}
	return kind
}

var backoffBase = 500 * time.Millisecond

func sleepBackoff(ctx context.Context, attempt int) error {
	base := time.Duration(1<<(attempt-1)) * backoffBase
	jitter := time.Duration(rand.Int64N(int64(backoffBase/2) + 1))
	timer := time.NewTimer(base + jitter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func jsonSchemaResponseFormat(name string, schema map[string]any) map[string]any {
	return map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   name,
			"strict": true,
			"schema": schema,
		},
	}
}

func buildAnalysisText(input domain.AnalyzeRequest) string {
	var b strings.Builder
	b.WriteString("Текст пользователя:\n")
	if strings.TrimSpace(input.UserText) == "" {
		b.WriteString("(пользователь не добавил текст)\n")
	} else {
		b.WriteString(input.UserText)
		b.WriteString("\n")
	}
	if len(input.PreviousJSON) > 0 {
		b.WriteString("\nПредыдущий structured output JSON:\n")
		b.Write(input.PreviousJSON)
		b.WriteString("\n")
	}
	if strings.TrimSpace(input.Correction) != "" {
		b.WriteString("\nКорректировка пользователя:\n")
		b.WriteString(input.Correction)
		b.WriteString("\n")
	}
	return b.String()
}

func imageDataURL(path, mimeType string) (string, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}
