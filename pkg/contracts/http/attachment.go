package httpcontracts

import "time"

// UploadAttachmentData is the upload attachment response payload.
type UploadAttachmentData struct {
	AttachmentID string    `json:"attachment_id"`
	FileKey      string    `json:"file_key"`
	DownloadURL  string    `json:"download_url"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	SHA256       string    `json:"sha256"`
	CreatedAt    time.Time `json:"created_at"`
}
