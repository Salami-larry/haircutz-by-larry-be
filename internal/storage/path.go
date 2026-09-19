package storage

import (
	"fmt"
	"net/url"
	"strings"
)

const hairstyleObjectPrefix = "hairstyles/"

// objectPathFromPublicURL extracts the storage object path (e.g. hairstyles/uuid.jpg)
// from a Supabase public object URL for the configured bucket.
func objectPathFromPublicURL(publicURL, bucket string) (string, error) {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return "", fmt.Errorf("empty url")
	}

	u, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	marker := "/object/public/" + bucket + "/"
	idx := strings.Index(u.Path, marker)
	if idx < 0 {
		return "", fmt.Errorf("url is not a public object in bucket %q", bucket)
	}

	objectPath := strings.TrimPrefix(u.Path[idx:], marker)
	objectPath = strings.TrimPrefix(objectPath, "/")
	if objectPath == "" {
		return "", fmt.Errorf("empty object path")
	}

	if !strings.HasPrefix(objectPath, hairstyleObjectPrefix) {
		return "", fmt.Errorf("object path %q is outside %q", objectPath, hairstyleObjectPrefix)
	}

	return objectPath, nil
}
