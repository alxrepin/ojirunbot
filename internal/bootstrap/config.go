package bootstrap

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ojirun/internal/context/infrastructure/openrouter"
)

type Config struct {
	AppEnv                string
	LogLevel              string
	Telegram              TelegramConfig
	OpenRouter            OpenRouterConfig
	DatabaseURL           string
	ProxyURL              string
	Timezone              string
	DailyReportTime       string
	PhotoStorageDir       string
	MaxPhotoBytes         int64
	MaxConcurrentUpdates  int
	MaxConcurrentAI       int
	ReportWorkers         int
	MaxMealsPerDay        int
	MaxMealsPerChatPerDay int
	MaxMealsInFlight      int
}

type TelegramConfig struct {
	BotToken        string
	APIBaseURL      string
	RequiredChannel string
	ChannelURL      string
}

type OpenRouterConfig struct {
	APIKey      string
	BaseURL     string
	Models      []string
	HTTPReferer string
	AppTitle    string
}

func Load() (Config, error) {
	_ = LoadDotEnv(".env")

	cfg := Config{
		AppEnv:                env("APP_ENV", "local"),
		LogLevel:              env("LOG_LEVEL", "info"),
		DatabaseURL:           env("DATABASE_URL", ""),
		ProxyURL:              env("PROXY_URL", ""),
		Timezone:              env("APP_TIMEZONE", "Europe/Moscow"),
		DailyReportTime:       env("DAILY_REPORT_TIME", "09:00"),
		PhotoStorageDir:       env("PHOTO_STORAGE_DIR", "./storage/photos"),
		MaxPhotoBytes:         envInt64("MAX_PHOTO_BYTES", 10*1024*1024),
		MaxConcurrentUpdates:  max(envInt("MAX_CONCURRENT_UPDATES", 32), 1),
		MaxConcurrentAI:       max(envInt("MAX_CONCURRENT_AI", 4), 1),
		ReportWorkers:         max(envInt("REPORT_WORKERS", 2), 1),
		MaxMealsPerDay:        max(envInt("MAX_MEALS_PER_DAY", 10), 0),
		MaxMealsPerChatPerDay: max(envInt("MAX_MEALS_PER_CHAT_PER_DAY", 100), 0),
		MaxMealsInFlight:      max(envInt("MAX_MEALS_IN_FLIGHT", 3), 0),
		Telegram: TelegramConfig{
			BotToken:        env("TELEGRAM_BOT_TOKEN", ""),
			APIBaseURL:      env("TELEGRAM_API_BASE_URL", "https://api.telegram.org"),
			RequiredChannel: normalizeChannel(env("TELEGRAM_REQUIRED_CHANNEL", "")),
			ChannelURL:      env("TELEGRAM_CHANNEL_URL", ""),
		},
		OpenRouter: OpenRouterConfig{
			APIKey:      env("OPENROUTER_API_KEY", ""),
			BaseURL:     strings.TrimRight(env("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"), "/"),
			Models:      envList("OPENROUTER_MODEL", "google/gemini-3-flash-preview", "google/gemini-2.5-flash", "openai/gpt-5-mini"),
			HTTPReferer: env("OPENROUTER_HTTP_REFERER", ""),
			AppTitle:    env("OPENROUTER_APP_TITLE", "Ojirun"),
		},
	}
	if cfg.Telegram.ChannelURL == "" && strings.HasPrefix(cfg.Telegram.RequiredChannel, "@") {
		cfg.Telegram.ChannelURL = "https://t.me/" + strings.TrimPrefix(cfg.Telegram.RequiredChannel, "@")
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	if c.Telegram.BotToken == "" {
		missing = append(missing, "TELEGRAM_BOT_TOKEN")
	}
	if c.Telegram.RequiredChannel != "" && c.Telegram.ChannelURL == "" {
		missing = append(missing, "TELEGRAM_CHANNEL_URL (required when TELEGRAM_REQUIRED_CHANNEL is a numeric id)")
	}
	if c.OpenRouter.APIKey == "" {
		missing = append(missing, "OPENROUTER_API_KEY")
	}
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("load timezone %q: %w", c.Timezone, err)
	}
	if _, _, err := ParseClock(c.DailyReportTime); err != nil {
		return fmt.Errorf("parse DAILY_REPORT_TIME: %w", err)
	}
	if c.MaxPhotoBytes <= 0 {
		return errors.New("MAX_PHOTO_BYTES must be positive")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (t TelegramConfig) ChannelTarget() any {
	if t.RequiredChannel == "" {
		return nil
	}
	if id, err := strconv.ParseInt(t.RequiredChannel, 10, 64); err == nil {
		return id
	}
	return t.RequiredChannel
}

func (t TelegramConfig) ChannelLabel() string {
	if strings.HasPrefix(t.RequiredChannel, "@") {
		return t.RequiredChannel
	}
	return t.ChannelURL
}

func normalizeChannel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return value
	}
	for _, prefix := range []string{"https://t.me/", "http://t.me/", "t.me/"} {
		value = strings.TrimPrefix(value, prefix)
	}
	return "@" + strings.TrimPrefix(strings.Trim(value, "/"), "@")
}

func (c Config) OpenRouterClientConfig() openrouter.Config {
	return openrouter.Config{
		APIKey:      c.OpenRouter.APIKey,
		BaseURL:     c.OpenRouter.BaseURL,
		Models:      c.OpenRouter.Models,
		HTTPReferer: c.OpenRouter.HTTPReferer,
		AppTitle:    c.OpenRouter.AppTitle,
		ProxyURL:    c.ProxyURL,
	}
}

func ParseClock(value string) (hour, minute int, err error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected HH:MM, got %q", value)
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("clock out of range: %q", value)
	}
	return hour, minute, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envList(key string, fallback ...string) []string {
	var items []string
	for _, part := range strings.Split(env(key, ""), ",") {
		if part = strings.TrimSpace(part); part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}

func envInt(key string, fallback int) int {
	return int(envInt64(key, int64(fallback)))
}

func envInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}
