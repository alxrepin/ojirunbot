package files

import (
	"context"

	"ojirun/internal/context/domain"
)

type Archiver struct {
	storage *PhotoStorage
	source  PhotoSource
}

func NewArchiver(storage *PhotoStorage, source PhotoSource) *Archiver {
	return &Archiver{storage: storage, source: source}
}

func (a *Archiver) Archive(ctx context.Context, mealID string, ref domain.PhotoRef) (domain.Photo, error) {
	return a.storage.Save(ctx, a.source, mealID, ref)
}
