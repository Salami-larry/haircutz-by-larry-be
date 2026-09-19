package controller

import (
	"fmt"
	"io"
	"strings"
)

var (
	ErrUploadNotConfigured = fmt.Errorf("image upload is not configured")
	ErrInvalidImageType    = fmt.Errorf("invalid image type")
	ErrInvalidVideoType    = fmt.Errorf("invalid video type")
)

type MediaUploader interface {
	UploadHairstyleMedia(filename string, r io.Reader, contentType string) (string, error)
}

type UploadController struct {
	uploader MediaUploader
}

func NewUploadController(uploader MediaUploader) *UploadController {
	return &UploadController{uploader: uploader}
}

func (c *UploadController) UploadHairstyleImage(filename, contentType string, r io.Reader) (string, error) {
	if c.uploader == nil {
		return "", ErrUploadNotConfigured
	}

	contentType = resolveImageContentType(filename, contentType)
	if !isAllowedImageType(contentType) {
		return "", ErrInvalidImageType
	}

	url, err := c.uploader.UploadHairstyleMedia(filename, r, contentType)
	if err != nil {
		return "", fmt.Errorf("upload hairstyle image: %w", err)
	}
	return url, nil
}

func (c *UploadController) UploadHairstyleVideo(filename, contentType string, r io.Reader) (string, error) {
	if c.uploader == nil {
		return "", ErrUploadNotConfigured
	}

	contentType = resolveVideoContentType(filename, contentType)
	if !isAllowedVideoType(contentType) {
		return "", ErrInvalidVideoType
	}

	if !hasVideoExt(filename) {
		filename = filename + extForVideoContentType(contentType)
	}

	url, err := c.uploader.UploadHairstyleMedia(filename, r, contentType)
	if err != nil {
		return "", fmt.Errorf("upload hairstyle video: %w", err)
	}
	return url, nil
}

func resolveImageContentType(filename, contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if isAllowedImageType(contentType) {
		return contentType
	}
	if fromName := imageContentTypeFromFilename(filename); fromName != "" {
		return fromName
	}
	return contentType
}

func resolveVideoContentType(filename, contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	if isAllowedVideoType(contentType) {
		return contentType
	}
	if fromName := videoContentTypeFromFilename(filename); fromName != "" {
		return fromName
	}
	return contentType
}

func isAllowedImageType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func isAllowedVideoType(ct string) bool {
	switch ct {
	case "video/mp4", "video/quicktime", "video/x-matroska":
		return true
	default:
		return false
	}
}

func imageContentTypeFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	default:
		return ""
	}
}

func videoContentTypeFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".m4v"):
		return "video/mp4"
	case strings.HasSuffix(lower, ".mov"):
		return "video/quicktime"
	case strings.HasSuffix(lower, ".mkv"):
		return "video/x-matroska"
	default:
		return ""
	}
}

func hasVideoExt(filename string) bool {
	return videoContentTypeFromFilename(filename) != ""
}

func extForVideoContentType(ct string) string {
	switch ct {
	case "video/quicktime":
		return ".mov"
	case "video/x-matroska":
		return ".mkv"
	default:
		return ".mp4"
	}
}
