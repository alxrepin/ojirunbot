package openrouter

import (
	"log/slog"
)

type Config struct {
	APIKey      string
	BaseURL     string
	Models      []string
	HTTPReferer string
	AppTitle    string
	ProxyURL    string
	Logger      *slog.Logger
}
