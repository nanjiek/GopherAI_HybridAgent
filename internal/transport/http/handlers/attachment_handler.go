package handlers

import (
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gophermind/internal/core/service"
	httpcontracts "gophermind/pkg/contracts/http"
)

// AttachmentHandler handles attachment upload and download.
type AttachmentHandler struct {
	svc    *service.AttachmentService
	logger *zap.Logger
}

// NewAttachmentHandler builds AttachmentHandler.
func NewAttachmentHandler(svc *service.AttachmentService, logger *zap.Logger) *AttachmentHandler {
	return &AttachmentHandler{
		svc:    svc,
		logger: logger,
	}
}

// Upload handles multipart attachment upload.
func (h *AttachmentHandler) Upload(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, httpcontracts.Err(50311, "attachment service unavailable"))
		return
	}
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, httpcontracts.Err(40103, "missing user id"))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40021, "missing file field"))
		return
	}
	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40022, "empty file"))
		return
	}
	if fileHeader.Size > h.svc.MaxFileSizeBytes() {
		c.JSON(http.StatusRequestEntityTooLarge, httpcontracts.Err(41301, "file too large"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40023, "open file failed"))
		return
	}
	defer file.Close()

	out, err := h.svc.Upload(c.Request.Context(), userID, fileHeader.Filename, file, fileHeader.Size)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAttachmentEmpty):
			c.JSON(http.StatusBadRequest, httpcontracts.Err(40022, "empty file"))
		case errors.Is(err, service.ErrAttachmentTooLarge):
			c.JSON(http.StatusRequestEntityTooLarge, httpcontracts.Err(41301, "file too large"))
		case errors.Is(err, service.ErrAttachmentTypeNotAllowed):
			c.JSON(http.StatusUnsupportedMediaType, httpcontracts.Err(41501, "file type not allowed"))
		default:
			if h.logger != nil {
				h.logger.Error("upload attachment failed", zap.Error(err))
			}
			c.JSON(http.StatusInternalServerError, httpcontracts.Err(50021, "upload failed"))
		}
		return
	}

	c.JSON(http.StatusOK, httpcontracts.OK(httpcontracts.UploadAttachmentData{
		AttachmentID: out.ID,
		FileKey:      out.FileKey,
		DownloadURL:  "/attachments/file?key=" + out.FileKey,
		OriginalName: out.OriginalName,
		ContentType:  out.ContentType,
		SizeBytes:    out.SizeBytes,
		SHA256:       out.SHA256,
		CreatedAt:    out.CreatedAt,
	}))
}

// Download returns uploaded file content for current user.
func (h *AttachmentHandler) Download(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, httpcontracts.Err(50311, "attachment service unavailable"))
		return
	}
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, httpcontracts.Err(40103, "missing user id"))
		return
	}
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40024, "missing key"))
		return
	}
	fullPath, err := h.svc.ResolveFilePath(userID, key)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40025, "invalid key"))
		return
	}
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, httpcontracts.Err(40421, "file not found"))
			return
		}
		if h.logger != nil {
			h.logger.Error("stat attachment failed", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, httpcontracts.Err(50022, "read file failed"))
		return
	}
	c.File(fullPath)
}
