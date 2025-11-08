package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// parseEncryptionSecret parses the encryption secret.
// Supports two formats:
//  1. 64 hex characters (decoded to 32 bytes) - recommended format
//  2. 32 bytes raw string - compatible with legacy format
func parseEncryptionSecret(secret string) ([]byte, error) {
	// Try to decode as hex string (64 chars -> 32 bytes)
	if len(secret) == 64 {
		decoded, err := hex.DecodeString(secret)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}

	// If not valid hex, try as raw string (32 bytes)
	if len(secret) != 32 {
		return nil, fmt.Errorf("encryption secret must be either 64 hex characters or 32 bytes string, got %d bytes", len(secret))
	}
	return []byte(secret), nil
}

// EncryptAPIKey encrypts an API key using AES-256-GCM
// Returns base64(nonce + ciphertext)
func EncryptAPIKey(plaintext string, secret string) (string, error) {
	// Parse and validate secret
	key, err := parseEncryptionSecret(secret)
	if err != nil {
		return "", err
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (12 bytes for GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine nonce + ciphertext and encode to base64
	combined := append(nonce, ciphertext...)
	encoded := base64.StdEncoding.EncodeToString(combined)

	return encoded, nil
}

// DecryptAPIKey decrypts an API key using AES-256-GCM
// Expects base64(nonce + ciphertext)
func DecryptAPIKey(encoded string, secret string) (string, error) {
	// Parse and validate secret (支持 64 字符十六进制或 32 字节原始字符串)
	key, err := parseEncryptionSecret(secret)
	if err != nil {
		return "", err
	}

	// Decode base64
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := combined[:nonceSize]
	ciphertext := combined[nonceSize:]

	// Decrypt ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// MaskAPIKey masks an API key for display
// Format: prefix-***-last6
// Example: sk-proj-abc123...xyz789 -> sk-proj-***-xyz789
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	// If API key is too short, mask entirely
	if len(apiKey) <= 12 {
		return "***"
	}

	// Find prefix (up to first dash or first 8 characters)
	prefixEnd := 8
	for i, ch := range apiKey {
		if ch == '-' && i > 0 {
			prefixEnd = i
			break
		}
		if i >= 8 {
			break
		}
	}

	prefix := apiKey[:prefixEnd]
	suffix := apiKey[len(apiKey)-6:]

	return fmt.Sprintf("%s-***-%s", prefix, suffix)
}

// IsMaskedAPIKey checks if an API key is masked (contains ***)
func IsMaskedAPIKey(apiKey string) bool {
	return apiKey != "" && (apiKey == "***" || len(apiKey) > 3 && apiKey[len(apiKey)-3:] == "***" || len(apiKey) > 5 && apiKey[len(apiKey)-5:len(apiKey)-2] == "***")
}
