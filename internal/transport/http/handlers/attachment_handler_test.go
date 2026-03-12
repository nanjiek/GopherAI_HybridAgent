package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gophermind/internal/config"
	"gophermind/internal/core/service"
	httpcontracts "gophermind/pkg/contracts/http"
)

func TestAttachmentHandlerUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewAttachmentService(config.UploadConfig{
		Dir:              t.TempDir(),
		MaxFileSizeBytes: 1024 * 1024,
		AllowedExts:      []string{".txt"},
	}, nil)
	h := NewAttachmentHandler(svc, nil)

	r := gin.New()
	r.POST("/attachments", func(c *gin.Context) {
		c.Set("user_id", "u1")
		h.Upload(c)
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "a.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello upload"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/attachments", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp httpcontracts.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
}

func TestAttachmentHandlerDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	svc := service.NewAttachmentService(config.UploadConfig{
		Dir:              tmpDir,
		MaxFileSizeBytes: 1024 * 1024,
		AllowedExts:      []string{".txt"},
	}, nil)
	h := NewAttachmentHandler(svc, nil)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "u1", "2026", "03", "12"), 0o755))
	fullPath := filepath.Join(tmpDir, "u1", "2026", "03", "12", "f.txt")
	require.NoError(t, os.WriteFile(fullPath, []byte("download me"), 0o644))

	r := gin.New()
	r.GET("/attachments/file", func(c *gin.Context) {
		c.Set("user_id", "u1")
		h.Download(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/attachments/file?key=u1/2026/03/12/f.txt", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "download me", w.Body.String())
}
