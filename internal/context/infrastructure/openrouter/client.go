package openrouter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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

type chatRequest struct {
	system string
	wire   any
	record any
	format map[string]any
}

func (r chatRequest) encode(model string) (wire, record []byte, err error) {
	wire, err = json.Marshal(r.payload(model, r.wire))
	if err != nil {
		return nil, nil, err
	}
	record, err = json.Marshal(r.payload(model, r.record))
	if err != nil {
		return nil, nil, err
	}
	return wire, record, nil
}

func (r chatRequest) payload(model string, user any) map[string]any {
	return map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "system", "content": r.system},
			{"role": "user", "content": user},
		},
		"response_format": r.format,
		"stream":          false,
	}
}

func (c *Client) AnalyzeNutrition(ctx context.Context, input domain.AnalyzeRequest) (domain.AIResult[domain.NutritionAnalysis], error) {
	text := textPart(buildAnalysisText(input))
	wire := []map[string]any{text}
	record := []map[string]any{text}
	if input.ImagePath != "" {
		image, err := loadImage(input.ImagePath, input.ImageMime)
		if err != nil {
			return domain.AIResult[domain.NutritionAnalysis]{}, err
		}
		wire = append(wire, imagePart(image.dataURL))
		record = append(record, imagePart(image.ref))
	}
	return doStructured[domain.NutritionAnalysis](ctx, c, chatRequest{
		system: input.Prompt,
		wire:   wire,
		record: record,
		format: jsonSchemaResponseFormat("nutrition_analysis", nutritionAnalysisSchema()),
	})
}

func (c *Client) DailyRecommendation(ctx context.Context, input domain.RecommendationRequest) (domain.AIResult[domain.DailyRecommendation], error) {
	text := strings.Builder{}
	text.WriteString("Пользователь: ")
	text.WriteString(input.UserName)
	text.WriteString("\n\nЦели и нормы JSON:\n")
	text.Write(input.TargetsJSON)
	text.WriteString("\n\nОтчет за вчера JSON:\n")
	text.Write(input.ReportJSON)

	return doStructured[domain.DailyRecommendation](ctx, c, chatRequest{
		system: input.Prompt,
		wire:   text.String(),
		record: text.String(),
		format: jsonSchemaResponseFormat("daily_recommendation", dailyRecommendationSchema()),
	})
}

const maxAttempts = 3

type attemptOutcome int

const (
	outcomeModelFailure attemptOutcome = iota
	outcomeContentFailure
	outcomeFatal
)

func doStructured[T any](ctx context.Context, c *Client, req chatRequest) (domain.AIResult[T], error) {
	models := c.cfg.Models
	totalAttempts := max(maxAttempts, len(models))

	modelIdx := 0
	wire, record, err := req.encode(models[modelIdx])
	if err != nil {
		return domain.AIResult[T]{}, err
	}

	var attemptErrs []error
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		if attempt > 1 {
			if err := sleepBackoff(ctx, attempt); err != nil {
				return domain.AIResult[T]{RequestJSON: record}, err
			}
		}
		result, outcome, err := doAttempt[T](ctx, c, wire, record)
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
			wire, record, err = req.encode(models[next])
			if err != nil {
				return domain.AIResult[T]{}, err
			}
			modelIdx = next
		} else {
			c.log.WarnContext(ctx, "openrouter attempt failed, retrying same model",
				"attempt", attempt, "model", models[modelIdx], "error", err)
		}
	}
	return domain.AIResult[T]{RequestJSON: record}, fmt.Errorf("openrouter failed after %d attempts: %w", totalAttempts, errors.Join(attemptErrs...))
}

func doAttempt[T any](ctx context.Context, c *Client, wire, record []byte) (domain.AIResult[T], attemptOutcome, error) {
	result := domain.AIResult[T]{RequestJSON: record}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(wire))
	if err != nil {
		return result, outcomeFatal, err
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
		return result, failureOutcome(ctx, outcomeModelFailure), err
	}
	defer resp.Body.Close()

	result.ResponseJSON, err = io.ReadAll(resp.Body)
	if err != nil {
		return result, failureOutcome(ctx, outcomeModelFailure), err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter status %s: %s", resp.Status, string(result.ResponseJSON))
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
	if err := json.Unmarshal(result.ResponseJSON, &envelope); err != nil {
		return result, failureOutcome(ctx, outcomeModelFailure), err
	}
	if envelope.Error != nil {
		return result, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Choices) == 0 {
		return result, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter returned no choices; body=%s", string(result.ResponseJSON))
	}
	if envelope.Choices[0].Error != nil {
		return result, failureOutcome(ctx, outcomeModelFailure), fmt.Errorf("openrouter choice error %d: %s", envelope.Choices[0].Error.Code, envelope.Choices[0].Error.Message)
	}
	if err := json.Unmarshal([]byte(envelope.Choices[0].Message.Content), &result.Value); err != nil {
		return result, failureOutcome(ctx, outcomeContentFailure), fmt.Errorf("decode structured content: %w; content=%s", err, envelope.Choices[0].Message.Content)
	}
	return result, outcomeModelFailure, nil
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

type inlineImage struct {
	dataURL string
	ref     string
}

func loadImage(path, mimeType string) (inlineImage, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return inlineImage{}, err
	}
	sum := sha256.Sum256(raw)
	return inlineImage{
		dataURL: "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw),
		ref:     "photo:sha256:" + hex.EncodeToString(sum[:]),
	}, nil
}

func textPart(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func imagePart(url string) map[string]any {
	return map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}}
}
