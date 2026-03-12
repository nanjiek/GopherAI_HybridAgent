package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gophermind/internal/config"
)

func TestAttachmentServiceUploadSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAttachmentService(config.UploadConfig{
		Dir:              tmpDir,
		MaxFileSizeBytes: 1024,
		AllowedExts:      []string{".txt"},
	}, nil)

	out, err := svc.Upload(context.Background(), "u1", "note.txt", strings.NewReader("hello"), 5)
	require.NoError(t, err)
	require.Equal(t, "u1", out.UserID)
	require.NotEmpty(t, out.FileKey)
	require.Equal(t, int64(5), out.SizeBytes)

	absPath, err := svc.ResolveFilePath("u1", out.FileKey)
	require.NoError(t, err)
	raw, err := os.ReadFile(absPath)
	require.NoError(t, err)
	require.Equal(t, "hello", string(raw))
}

func TestAttachmentServiceUploadTypeBlocked(t *testing.T) {
	svc := NewAttachmentService(config.UploadConfig{
		Dir:              t.TempDir(),
		MaxFileSizeBytes: 1024,
		AllowedExts:      []string{".txt"},
	}, nil)
	_, err := svc.Upload(context.Background(), "u1", "a.exe", strings.NewReader("x"), 1)
	require.ErrorIs(t, err, ErrAttachmentTypeNotAllowed)
}

func TestAttachmentServiceUploadTooLarge(t *testing.T) {
	svc := NewAttachmentService(config.UploadConfig{
		Dir:              t.TempDir(),
		MaxFileSizeBytes: 3,
		AllowedExts:      []string{".txt"},
	}, nil)
	_, err := svc.Upload(context.Background(), "u1", "a.txt", strings.NewReader("hello"), 5)
	require.ErrorIs(t, err, ErrAttachmentTooLarge)
}

func TestAttachmentServiceResolveFilePathRejectCrossUser(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAttachmentService(config.UploadConfig{
		Dir:              tmpDir,
		MaxFileSizeBytes: 1024,
		AllowedExts:      []string{".txt"},
	}, nil)

	// Ensure target file exists to avoid os.IsNotExist branch confusion in callers.
	p := filepath.Join(tmpDir, "u2", "2026", "01", "01")
	require.NoError(t, os.MkdirAll(p, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(p, "x.txt"), []byte("x"), 0o644))

	_, err := svc.ResolveFilePath("u1", "u2/2026/01/01/x.txt")
	require.ErrorIs(t, err, ErrInvalidAttachmentKey)
}
