package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"ojirun/internal/context/domain"
)

func newTestClient(t *testing.T, srv *httptest.Server, models ...string) *Client {
	t.Helper()
	prev := backoffBase
	backoffBase = time.Millisecond
	t.Cleanup(func() { backoffBase = prev })

	if len(models) == 0 {
		models = []string{"test-model"}
	}
	c, err := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Models:  models,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func requestModel(t *testing.T, r *http.Request) string {
	t.Helper()
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	return payload.Model
}

const validEnvelope = `{"choices":[{"message":{"content":"{\"summary\":\"гречка\",\"items\":[],\"total\":{\"calories_kcal\":100,\"protein_g\":10,\"fat_g\":5,\"carbs_g\":12},\"confidence\":0.9,\"notes\":\"\"}"}}]}`

func analyze(t *testing.T, c *Client) (domain.AIResult[domain.NutritionAnalysis], error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.AnalyzeNutrition(ctx, domain.AnalyzeRequest{Prompt: "prompt", UserText: "гречка"})
}

func TestRetriesThenSucceeds(t *testing.T) {
	cases := map[string]func(w http.ResponseWriter){
		"http 500": func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusInternalServerError)
		},
		"http 400": func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusBadRequest)
		},
		"error envelope in 200": func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"error":{"code":502,"message":"provider down"}}`))
		},
		"no choices": func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"choices":[]}`))
		},
		"choice error": func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"choices":[{"error":{"code":403,"message":"refused"},"message":{"content":""}}]}`))
		},
		"malformed structured content": func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
		},
		"malformed envelope": func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{{{`))
		},
	}
	for name, failOnce := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					failOnce(w)
					return
				}
				_, _ = w.Write([]byte(validEnvelope))
			}))
			defer srv.Close()

			result, err := analyze(t, newTestClient(t, srv))
			if err != nil {
				t.Fatalf("expected success after retry, got: %v", err)
			}
			if calls != 2 {
				t.Errorf("calls = %d, want 2", calls)
			}
			if result.Value.Total.CaloriesKCal != 100 {
				t.Errorf("total kcal = %v, want 100", result.Value.Total.CaloriesKCal)
			}
		})
	}
}

func TestFailsAfterMaxAttempts(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer srv.Close()

	_, err := analyze(t, newTestClient(t, srv))
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != maxAttempts {
		t.Errorf("calls = %d, want %d", calls, maxAttempts)
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Errorf("error should mention attempts, got: %v", err)
	}
}

func TestFailsOverToNextModelWhenAPIDoesNotAnswer(t *testing.T) {
	var served []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		model := requestModel(t, r)
		served = append(served, model)
		if model == "primary" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(validEnvelope))
	}))
	defer srv.Close()

	result, err := analyze(t, newTestClient(t, srv, "primary", "backup-1", "backup-2"))
	if err != nil {
		t.Fatalf("expected success via backup model, got: %v", err)
	}
	if want := []string{"primary", "backup-1"}; !reflect.DeepEqual(served, want) {
		t.Errorf("models served = %v, want %v", served, want)
	}
	if result.Value.Total.CaloriesKCal != 100 {
		t.Errorf("total kcal = %v, want 100", result.Value.Total.CaloriesKCal)
	}
}

func TestContentErrorRetriesSameModel(t *testing.T) {
	var served []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = append(served, requestModel(t, r))
		if len(served) == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
			return
		}
		_, _ = w.Write([]byte(validEnvelope))
	}))
	defer srv.Close()

	if _, err := analyze(t, newTestClient(t, srv, "primary", "backup")); err != nil {
		t.Fatalf("expected success after re-ask, got: %v", err)
	}
	if want := []string{"primary", "primary"}; !reflect.DeepEqual(served, want) {
		t.Errorf("models served = %v, want %v", served, want)
	}
}

func TestFailsAfterAllModelsDown(t *testing.T) {
	var served []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = append(served, requestModel(t, r))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := analyze(t, newTestClient(t, srv, "m1", "m2", "m3", "m4"))
	if err == nil {
		t.Fatal("expected error after exhausting all models")
	}
	if want := []string{"m1", "m2", "m3", "m4"}; !reflect.DeepEqual(served, want) {
		t.Errorf("models served = %v, want %v", served, want)
	}
	for _, model := range []string{"m1", "m2", "m3", "m4"} {
		if !strings.Contains(err.Error(), "model "+model+":") {
			t.Errorf("error should mention %s, got: %v", model, err)
		}
	}
}

func TestNoRetryAfterContextCancelled(t *testing.T) {
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cancel()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.AnalyzeNutrition(ctx, domain.AnalyzeRequest{Prompt: "prompt", UserText: "гречка"})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retries once context is done)", calls)
	}
}
