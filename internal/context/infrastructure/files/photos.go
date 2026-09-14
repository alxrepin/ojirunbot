package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/telegram"
)

var errPhotoTooLarge = errors.New("photo exceeds max size")

const (
	downloadAttempts = 3
	attemptTimeout   = 45 * time.Second
	retryBackoff     = time.Second
)

type PhotoSource interface {
	GetFile(ctx context.Context, fileID string) (telegram.File, error)
	DownloadFile(ctx context.Context, filePath string, dst io.Writer) error
}

type PhotoStorage struct {
	dir      string
	maxBytes int64
	backoff  time.Duration
}

func NewPhotoStorage(dir string, maxBytes int64) *PhotoStorage {
	return &PhotoStorage{dir: dir, maxBytes: maxBytes, backoff: retryBackoff}
}

func (s *PhotoStorage) Save(ctx context.Context, source PhotoSource, mealID string, ref domain.PhotoRef) (domain.Photo, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return domain.Photo{}, err
	}
	raw, filePath, err := s.download(ctx, source, ref)
	if err != nil {
		return domain.Photo{}, err
	}

	mimeType := http.DetectContentType(raw)
	ext := extensionFromMime(mimeType)
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(filePath))
	}
	if ext == "" {
		ext = ".jpg"
	}

	sum := sha256.Sum256(raw)
	sha := hex.EncodeToString(sum[:])
	filename := fmt.Sprintf("%s_%s%s", mealID, ref.FileUniqueID, ext)
	path := filepath.Join(s.dir, filename)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return domain.Photo{}, err
	}

	return domain.Photo{
		MealEntryID:          mealID,
		TelegramFileID:       ref.FileID,
		TelegramFileUniqueID: ref.FileUniqueID,
		LocalPath:            path,
		MimeType:             mimeType,
		FileSize:             int64(len(raw)),
		SHA256:               sha,
	}, nil
}

func (s *PhotoStorage) download(ctx context.Context, source PhotoSource, ref domain.PhotoRef) ([]byte, string, error) {
	var lastErr error
	for attempt := 1; attempt <= downloadAttempts; attempt++ {
		if attempt > 1 && s.backoff > 0 {
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(s.backoff):
			}
		}

		raw, filePath, err := s.downloadOnce(ctx, source, ref)
		if err == nil {
			return raw, filePath, nil
		}
		lastErr = err
		if errors.Is(err, errPhotoTooLarge) || ctx.Err() != nil {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("download photo after %d attempts: %w", downloadAttempts, lastErr)
}

func (s *PhotoStorage) downloadOnce(ctx context.Context, source PhotoSource, ref domain.PhotoRef) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	file, err := source.GetFile(ctx, ref.FileID)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	limited := &limitedWriter{w: &buf, max: s.maxBytes}
	if err := source.DownloadFile(ctx, file.FilePath, limited); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), file.FilePath, nil
}

type limitedWriter struct {
	w       *bytes.Buffer
	max     int64
	written int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.written+int64(len(p)) > w.max {
		return 0, errPhotoTooLarge
	}
	n, err := w.w.Write(p)
	w.written += int64(n)
	return n, err
}

func extensionFromMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
