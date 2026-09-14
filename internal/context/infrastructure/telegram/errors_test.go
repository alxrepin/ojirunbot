package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorFromResponse(t *testing.T) {
	cases := map[string]struct {
		body       string
		retryAfter time.Duration
		permanent  bool
	}{
		"flood control": {
			body:       `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 7","parameters":{"retry_after":7}}`,
			retryAfter: 7 * time.Second,
		},
		"blocked by user": {
			body:      `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`,
			permanent: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "TOKEN", "")
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SendMessage(context.Background(), 1, "hi", nil)

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %v", err)
			}
			if apiErr.RetryAfter() != tc.retryAfter || apiErr.Permanent() != tc.permanent {
				t.Fatalf("retryAfter=%v permanent=%v, want %v %v", apiErr.RetryAfter(), apiErr.Permanent(), tc.retryAfter, tc.permanent)
			}
			if !strings.HasPrefix(err.Error(), "telegram api error") {
				t.Fatalf("error text changed: %q", err.Error())
			}
		})
	}
}
