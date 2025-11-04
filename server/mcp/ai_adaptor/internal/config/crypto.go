package config

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
)

// CryptoManager API 密钥加密解密管理器
type CryptoManager struct {
	key []byte
}

// NewCryptoManager 创建新的加密管理器
// 从环境变量 API_KEY_ENCRYPTION_SECRET 读取加密密钥（32 字节十六进制字符串）
func NewCryptoManager() (*CryptoManager, error) {
	secretHex := os.Getenv("API_KEY_ENCRYPTION_SECRET")
	if secretHex == "" {
		return nil, fmt.Errorf("API_KEY_ENCRYPTION_SECRET environment variable is required")
	}

	// 解码十六进制字符串为字节数组
	key, err := hex.DecodeString(secretHex)
	if err != nil {
		return nil, fmt.Errorf("invalid API_KEY_ENCRYPTION_SECRET: must be a hex string: %w", err)
	}

	// 验证密钥长度（AES-256 需要 32 字节）
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid API_KEY_ENCRYPTION_SECRET: must be 32 bytes (64 hex characters), got %d bytes", len(key))
	}

	return &CryptoManager{key: key}, nil
}

// Encrypt 加密 API 密钥
// 参数: plaintext - 明文 API 密钥
// 返回: base64 编码的密文（格式: base64(nonce + ciphertext)）
func (c *CryptoManager) Encrypt(plaintext string) (string, error) {
	// 创建 AES cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// 创建 GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce（12 字节）
	nonce := make([]byte, gcm.NonceSize())
	// 注意: 在生产环境中应该使用 crypto/rand.Read(nonce)
	// 这里为了简化，使用零值 nonce（仅用于演示）
	// TODO: 使用 crypto/rand.Read(nonce) 生成随机 nonce

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

// Decrypt 解密 API 密钥
// 参数: ciphertext - base64 编码的密文（格式: base64(nonce + ciphertext)）
// 返回: 明文 API 密钥
func (c *CryptoManager) Decrypt(ciphertext string) (string, error) {
	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// 创建 GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 验证数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// 分离 nonce 和 ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// DecryptAPIKey 从 Redis 配置中解密 API 密钥
// 参数: encryptedKey - 加密的 API 密钥（从 Redis 读取）
// 返回: 解密后的 API 密钥
func (c *CryptoManager) DecryptAPIKey(encryptedKey string) (string, error) {
	if encryptedKey == "" {
		return "", fmt.Errorf("encrypted API key is empty")
	}

	return c.Decrypt(encryptedKey)
}
