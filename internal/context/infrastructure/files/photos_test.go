package files

import (
	"context"
	"errors"
	"io"
	"testing"

	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/telegram"
)

type fakeSource struct {
	failDownloads int
	payload       []byte
	downloadErr   error
	attempts      int
}

func (f *fakeSource) GetFile(context.Context, string) (telegram.File, error) {
	return telegram.File{FilePath: "photos/file.jpg"}, nil
}

func (f *fakeSource) DownloadFile(_ context.Context, _ string, dst io.Writer) error {
	f.attempts++
	if f.attempts <= f.failDownloads {
		if f.downloadErr != nil {
			return f.downloadErr
		}
		return errors.New("proxy tunnel dropped")
	}
	_, err := dst.Write(f.payload)
	return err
}

var pngPayload = append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 16)...)

func TestSaveRetriesTransientDownload(t *testing.T) {
	src := &fakeSource{failDownloads: 2, payload: pngPayload}
	storage := NewPhotoStorage(t.TempDir(), 1<<20)
	storage.backoff = 0

	photo, err := storage.Save(context.Background(), src, "meal-1", domain.PhotoRef{FileID: "f", FileUniqueID: "u"})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if src.attempts != 3 {
		t.Fatalf("download attempts = %d, want 3 (2 failures + success)", src.attempts)
	}
	if photo.FileSize != int64(len(pngPayload)) || photo.MimeType != "image/png" {
		t.Fatalf("unexpected photo metadata: %+v", photo)
	}
}

func TestSaveStopsAfterMaxAttempts(t *testing.T) {
	src := &fakeSource{failDownloads: downloadAttempts + 5, payload: pngPayload}
	storage := NewPhotoStorage(t.TempDir(), 1<<20)
	storage.backoff = 0

	_, err := storage.Save(context.Background(), src, "meal-1", domain.PhotoRef{FileID: "f", FileUniqueID: "u"})
	if err == nil {
		t.Fatal("expected failure after exhausting attempts")
	}
	if src.attempts != downloadAttempts {
		t.Fatalf("download attempts = %d, want %d", src.attempts, downloadAttempts)
	}
}

func TestSaveDoesNotRetryOversizePhoto(t *testing.T) {
	src := &fakeSource{failDownloads: 0, payload: make([]byte, 64)}
	storage := NewPhotoStorage(t.TempDir(), 16)

	_, err := storage.Save(context.Background(), src, "meal-1", domain.PhotoRef{FileID: "f", FileUniqueID: "u"})
	if !errors.Is(err, errPhotoTooLarge) {
		t.Fatalf("error = %v, want errPhotoTooLarge", err)
	}
	if src.attempts != 1 {
		t.Fatalf("download attempts = %d, want 1 (no retry on size limit)", src.attempts)
	}
}
