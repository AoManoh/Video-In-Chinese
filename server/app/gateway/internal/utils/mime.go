package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// DetectMimeType detects the MIME type of a file by reading its first 512 bytes
func DetectMimeType(filePath string) (string, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type
	mimeType := http.DetectContentType(buffer[:n])
	return mimeType, nil
}

// ValidateMimeType checks if the given MIME type is in the whitelist
func ValidateMimeType(mimeType string, whitelist []string) error {
	for _, allowed := range whitelist {
		if mimeType == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported MIME type: %s (allowed: %v)", mimeType, whitelist)
}

// DetectAndValidateMimeType detects and validates the MIME type of a file
func DetectAndValidateMimeType(filePath string, whitelist []string) (string, error) {
	mimeType, err := DetectMimeType(filePath)
	if err != nil {
		return "", err
	}

	if err := ValidateMimeType(mimeType, whitelist); err != nil {
		return "", err
	}

	return mimeType, nil
}

