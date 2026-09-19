package storage

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	storagego "github.com/supabase-community/storage-go"
)

type Config struct {
	SupabaseURL    string
	ServiceRoleKey string
	Bucket         string
}

type Client struct {
	inner  *storagego.Client
	bucket string
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.SupabaseURL == "" || cfg.ServiceRoleKey == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("supabase storage is not configured")
	}

	baseURL := strings.TrimSuffix(cfg.SupabaseURL, "/") + "/storage/v1"
	return &Client{
		inner:  storagego.NewClient(baseURL, cfg.ServiceRoleKey, nil),
		bucket: cfg.Bucket,
	}, nil
}

func (c *Client) UploadHairstyleMedia(filename string, r io.Reader, contentType string) (string, error) {
	return c.upload("hairstyles", filename, r, contentType)
}

func (c *Client) upload(folder, filename string, r io.Reader, contentType string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = defaultExtForContentType(contentType)
	}

	objectPath := fmt.Sprintf("%s/%s%s", folder, uuid.NewString(), ext)

	upsert := false
	_, err := c.inner.UploadFile(c.bucket, objectPath, r, storagego.FileOptions{
		ContentType: &contentType,
		Upsert:      &upsert,
	})
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}

	public := c.inner.GetPublicUrl(c.bucket, objectPath)
	if public.SignedURL == "" {
		return "", fmt.Errorf("empty public url")
	}

	return public.SignedURL, nil
}

func defaultExtForContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/x-matroska":
		return ".mkv"
	default:
		return ".jpg"
	}
}

// DeleteByPublicURL removes a hairstyle media object previously uploaded to this bucket.
func (c *Client) DeleteByPublicURL(publicURL string) error {
	objectPath, err := objectPathFromPublicURL(publicURL, c.bucket)
	if err != nil {
		return err
	}
	_, err = c.inner.RemoveFile(c.bucket, []string{objectPath})
	if err != nil {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}
