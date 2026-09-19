package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/controller"
)

const (
	maxImageUploadBytes = 500 * 1024 // 500KB
	maxVideoUploadBytes = 5 << 20    // 5MB
)

type UploadHandler struct {
	ctrl *controller.UploadController
	log  *slog.Logger
}

func NewUploadHandler(ctrl *controller.UploadController, log *slog.Logger) *UploadHandler {
	return &UploadHandler{ctrl: ctrl, log: log}
}

func (h *UploadHandler) UploadHairstyleImage(c *gin.Context) {
	h.upload(c, maxImageUploadBytes, "image", func(filename, contentType string, r io.Reader) (string, error) {
		return h.ctrl.UploadHairstyleImage(filename, contentType, r)
	})
}

func (h *UploadHandler) UploadHairstyleVideo(c *gin.Context) {
	h.upload(c, maxVideoUploadBytes, "video", func(filename, contentType string, r io.Reader) (string, error) {
		return h.ctrl.UploadHairstyleVideo(filename, contentType, r)
	})
}

func (h *UploadHandler) upload(
	c *gin.Context,
	maxBytes int64,
	kind string,
	doUpload func(filename, contentType string, r io.Reader) (string, error),
) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+1024)

	if err := c.Request.ParseMultipartForm(maxBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fileTooLargeMessage(kind, maxBytes)})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	defer file.Close()

	if header.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": fileTooLargeMessage(kind, maxBytes)})
		return
	}

	contentType := header.Header.Get("Content-Type")
	limited := io.LimitReader(file, maxBytes+1)
	url, err := doUpload(header.Filename, contentType, limited)
	if err != nil {
		switch {
		case errors.Is(err, controller.ErrUploadNotConfigured):
			h.log.Warn("upload not configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upload service not configured"})
		case errors.Is(err, controller.ErrInvalidImageType):
			h.log.Warn("upload invalid image type", "contentType", contentType, "filename", header.Filename)
			c.JSON(http.StatusBadRequest, gin.H{"error": "allowed types: jpeg, png, webp, gif"})
		case errors.Is(err, controller.ErrInvalidVideoType):
			h.log.Warn("upload invalid video type", "contentType", contentType, "filename", header.Filename)
			c.JSON(http.StatusBadRequest, gin.H{"error": "allowed types: mp4, mov, mkv"})
		default:
			h.log.Error("upload failed",
				"err", err,
				"kind", kind,
				"filename", header.Filename,
				"contentType", contentType,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

func fileTooLargeMessage(kind string, maxBytes int64) string {
	if kind == "video" {
		return fmt.Sprintf("video exceeds %dMB limit", maxBytes>>20)
	}
	kb := maxBytes / 1024
	return fmt.Sprintf("image exceeds %dKB limit", kb)
}
