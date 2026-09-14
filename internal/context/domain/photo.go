package domain

type PhotoRef struct {
	FileID       string
	FileUniqueID string
}

type Photo struct {
	MealEntryID          string
	TelegramFileID       string
	TelegramFileUniqueID string
	LocalPath            string
	MimeType             string
	FileSize             int64
	SHA256               string
}
