package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/internal/core/model"
)

var (
	// ErrAttachmentEmpty means the uploaded file has no content.
	ErrAttachmentEmpty = errors.New("attachment is empty")
	// ErrAttachmentTooLarge means file size exceeds configured limit.
	ErrAttachmentTooLarge = errors.New("attachment too large")
	// ErrAttachmentTypeNotAllowed means file extension is blocked.
	ErrAttachmentTypeNotAllowed = errors.New("attachment type not allowed")
	// ErrInvalidAttachmentKey means file key cannot be trusted.
	ErrInvalidAttachmentKey = errors.New("invalid attachment key")
)

// AttachmentService handles attachment upload and file resolution.
type AttachmentService struct {
	cfg         config.UploadConfig
	logger      *zap.Logger
	allowedExts map[string]struct{}
}

// NewAttachmentService builds AttachmentService.
func NewAttachmentService(cfg config.UploadConfig, logger *zap.Logger) *AttachmentService {
	allowed := make(map[string]struct{}, len(cfg.AllowedExts))
	for _, ext := range cfg.AllowedExts {
		e := strings.TrimSpace(strings.ToLower(ext))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		allowed[e] = struct{}{}
	}
	return &AttachmentService{
		cfg:         cfg,
		logger:      logger,
		allowedExts: allowed,
	}
}

// MaxFileSizeBytes returns the configured upload limit.
func (s *AttachmentService) MaxFileSizeBytes() int64 {
	return s.cfg.MaxFileSizeBytes
}

// Upload saves attachment content to local storage and returns metadata.
func (s *AttachmentService) Upload(ctx context.Context, userID string, originalName string, file io.Reader, declaredSize int64) (model.Attachment, error) {
	if declaredSize <= 0 {
		return model.Attachment{}, ErrAttachmentEmpty
	}
	if s.cfg.MaxFileSizeBytes > 0 && declaredSize > s.cfg.MaxFileSizeBytes {
		return model.Attachment{}, ErrAttachmentTooLarge
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	if _, ok := s.allowedExts[ext]; !ok {
		return model.Attachment{}, ErrAttachmentTypeNotAllowed
	}

	limited := io.LimitReader(file, s.cfg.MaxFileSizeBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return model.Attachment{}, err
	}
	if len(content) == 0 {
		return model.Attachment{}, ErrAttachmentEmpty
	}
	if s.cfg.MaxFileSizeBytes > 0 && int64(len(content)) > s.cfg.MaxFileSizeBytes {
		return model.Attachment{}, ErrAttachmentTooLarge
	}

	select {
	case <-ctx.Done():
		return model.Attachment{}, ctx.Err()
	default:
	}

	safeUser := sanitizeUserID(userID)
	if safeUser == "" {
		return model.Attachment{}, ErrInvalidAttachmentKey
	}
	dayDir := time.Now().Format("2006/01/02")
	fileID := uuid.NewString()
	storedName := fileID + ext
	relDir := path.Join(safeUser, dayDir)
	fileKey := path.Join(relDir, storedName)

	absDir := filepath.Join(s.cfg.Dir, filepath.FromSlash(relDir))
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return model.Attachment{}, err
	}
	absPath := filepath.Join(absDir, storedName)
	tmpPath := absPath + ".tmp"

	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return model.Attachment{}, err
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		return model.Attachment{}, err
	}

	sum := sha256.Sum256(content)
	contentType := http.DetectContentType(content)
	createdAt := time.Now()

	if s.logger != nil {
		s.logger.Info("attachment uploaded",
			zap.String("user_id", safeUser),
			zap.String("file_key", fileKey),
			zap.String("content_type", contentType),
			zap.Int64("size_bytes", int64(len(content))),
		)
	}

	return model.Attachment{
		ID:           fileID,
		UserID:       safeUser,
		FileKey:      fileKey,
		OriginalName: originalName,
		ContentType:  contentType,
		SizeBytes:    int64(len(content)),
		SHA256:       hex.EncodeToString(sum[:]),
		CreatedAt:    createdAt,
	}, nil
}

// ResolveFilePath validates user/fileKey relation and returns absolute path.
func (s *AttachmentService) ResolveFilePath(userID string, fileKey string) (string, error) {
	clean := path.Clean(strings.TrimSpace(fileKey))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" || strings.Contains(clean, "..") {
		return "", ErrInvalidAttachmentKey
	}
	safeUser := sanitizeUserID(userID)
	if safeUser == "" {
		return "", ErrInvalidAttachmentKey
	}
	userPrefix := safeUser + "/"
	if !strings.HasPrefix(clean, userPrefix) {
		return "", ErrInvalidAttachmentKey
	}

	baseAbs, err := filepath.Abs(s.cfg.Dir)
	if err != nil {
		return "", err
	}
	targetAbs := filepath.Clean(filepath.Join(baseAbs, filepath.FromSlash(clean)))
	if !strings.HasPrefix(targetAbs, baseAbs+string(filepath.Separator)) && targetAbs != baseAbs {
		return "", ErrInvalidAttachmentKey
	}
	return targetAbs, nil
}

func sanitizeUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(userID))
	for _, r := range userID {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}
