package telegram

import (
	"fmt"
	"net/http"
	"time"
)

type APIError struct {
	Code              int
	Description       string
	RetryAfterSeconds int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram api error %d: %s", e.Code, e.Description)
}

func (e *APIError) RetryAfter() time.Duration {
	return time.Duration(e.RetryAfterSeconds) * time.Second
}

func (e *APIError) Permanent() bool {
	return e.Code == http.StatusBadRequest || e.Code == http.StatusForbidden
}
