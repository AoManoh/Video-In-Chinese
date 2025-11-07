package config

import (
"crypto/aes"
"crypto/cipher"
"crypto/rand"
"encoding/base64"
"encoding/hex"
"fmt"
"os"
)

// CryptoManager manages API key encryption/decryption logic
type CryptoManager struct {
key []byte
}

// NewCryptoManager creates a new crypto manager.
// Reads encryption key from API_KEY_ENCRYPTION_SECRET environment variable.
// Supports two formats:
//   1. 64 hex characters (decoded to 32 bytes) - recommended format
//   2. 32 bytes raw string - compatible with Gateway format
func NewCryptoManager() (*CryptoManager, error) {
secret := os.Getenv("API_KEY_ENCRYPTION_SECRET")
if secret == "" {
return nil, fmt.Errorf("API_KEY_ENCRYPTION_SECRET environment variable is required")
}

var key []byte

// Try to decode as hex string (64 chars -> 32 bytes)
if len(secret) == 64 {
decoded, err := hex.DecodeString(secret)
if err == nil && len(decoded) == 32 {
key = decoded
}
}

// If not valid hex, try as raw string (32 bytes)
if key == nil {
if len(secret) != 32 {
return nil, fmt.Errorf("invalid API_KEY_ENCRYPTION_SECRET: must be either 64 hex characters or 32 bytes string, got %d bytes", len(secret))
}
key = []byte(secret)
}

return &CryptoManager{key: key}, nil
}

// Encrypt encrypts API key and returns base64 encoded ciphertext
func (c *CryptoManager) Encrypt(plaintext string) (string, error) {
// Create AES cipher
block, err := aes.NewCipher(c.key)
if err != nil {
return "", fmt.Errorf("failed to create AES cipher: %w", err)
}

// Create GCM mode
gcm, err := cipher.NewGCM(block)
if err != nil {
return "", fmt.Errorf("failed to create GCM: %w", err)
}

// Generate random nonce (12 bytes)
nonce := make([]byte, gcm.NonceSize())
if _, err := rand.Read(nonce); err != nil {
return "", fmt.Errorf("failed to generate nonce: %w", err)
}

// Encrypt
ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

// Base64 encode
encoded := base64.StdEncoding.EncodeToString(ciphertext)
return encoded, nil
}

// Decrypt decrypts base64 encoded ciphertext and returns plaintext API key
func (c *CryptoManager) Decrypt(ciphertext string) (string, error) {
// Base64 decode
data, err := base64.StdEncoding.DecodeString(ciphertext)
if err != nil {
return "", fmt.Errorf("failed to decode base64: %w", err)
}

// Create AES cipher
block, err := aes.NewCipher(c.key)
if err != nil {
return "", fmt.Errorf("failed to create AES cipher: %w", err)
}

// Create GCM mode
gcm, err := cipher.NewGCM(block)
if err != nil {
return "", fmt.Errorf("failed to create GCM: %w", err)
}

// Verify data length
nonceSize := gcm.NonceSize()
if len(data) < nonceSize {
return "", fmt.Errorf("ciphertext too short")
}

// Separate nonce and ciphertext
nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

// Decrypt
plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
if err != nil {
return "", fmt.Errorf("failed to decrypt: %w", err)
}

return string(plaintext), nil
}

// DecryptAPIKey decrypts API key from Redis configuration
func (c *CryptoManager) DecryptAPIKey(encryptedKey string) (string, error) {
if encryptedKey == "" {
return "", fmt.Errorf("encrypted API key is empty")
}

return c.Decrypt(encryptedKey)
}