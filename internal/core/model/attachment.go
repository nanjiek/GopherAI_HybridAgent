package model

import "time"

// Attachment stores metadata of an uploaded user file.
type Attachment struct {
	ID           string
	UserID       string
	FileKey      string
	OriginalName string
	ContentType  string
	SizeBytes    int64
	SHA256       string
	CreatedAt    time.Time
}
