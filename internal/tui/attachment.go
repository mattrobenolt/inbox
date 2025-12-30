package tui

import (
	"encoding/base64"
	"path/filepath"
	"strings"
)

func isImageMimeType(mime string) bool {
	switch strings.ToLower(mime) {
	case "image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func isHTMLAttachment(mime, filename string) bool {
	mime = strings.ToLower(mime)
	if mime == "text/html" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".html" || ext == ".htm"
}

func isTextAttachment(mime, filename string) bool {
	mime = strings.ToLower(mime)
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	if mime == "application/json" || mime == "application/xml" || mime == "application/yaml" {
		return true
	}
	if strings.HasSuffix(mime, "+xml") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md", ".markdown", ".txt", ".log", ".json", ".xml", ".yml", ".yaml", ".csv":
		return true
	default:
		return false
	}
}

func decodeAttachmentData(data string) ([]byte, error) {
	if data == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(data)
	if err == nil {
		return decoded, nil
	}
	return base64.URLEncoding.DecodeString(data)
}
