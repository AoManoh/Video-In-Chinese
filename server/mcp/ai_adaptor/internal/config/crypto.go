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

// CryptoManager 管理 API 密钥的加解密逻辑，统一封装 AES-GCM 实现和密钥加载，
// 使配置管理器在处理敏感字段时具备一致的安全策略与错误处理。
type CryptoManager struct {
	key []byte
}

// NewCryptoManager 创建新的加密管理器。
// 从环境变量 API_KEY_ENCRYPTION_SECRET 读取 32 字节（64 个十六进制字符）的密钥，
// 并验证长度与格式。
//
// 设计说明:
//   - 采用 AES-256-GCM，满足 Phase 7 对称加密安全要求。
//   - 在初始化阶段快速失败（fast-fail），方便部署流程在密钥缺失时立即报警。
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

// Encrypt 加密 API 密钥并返回 base64 编码的密文（格式: base64(nonce + ciphertext)）。
//
// 参数:
//   - plaintext: 明文 API 密钥。
//
// 返回:
//   - 经过 base64 编码的密文。
//   - error: 加密失败时返回。
//
// 设计说明:
//   - 使用 AES-GCM 提供认证加密能力，避免密文被篡改。
//   - 使用 crypto/rand 生成随机 nonce，确保每次加密的唯一性。
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
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

// Decrypt 解密 base64 编码的密文，返回明文 API 密钥。
//
// 参数:
//   - ciphertext: base64(nonce + ciphertext) 格式的密文。
//
// 返回:
//   - 明文 API 密钥。
//   - error: 当密文格式错误或验证失败时返回。
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

// DecryptAPIKey 从 Redis 配置中解密 API 密钥，主要供 ConfigManager 使用。
//
// 参数:
//   - encryptedKey: 从 Redis 读取的密文 API 密钥。
//
// 返回:
//   - 解密后的 API 密钥。
//   - error: 当密文为空或解密失败时返回。
func (c *CryptoManager) DecryptAPIKey(encryptedKey string) (string, error) {
	if encryptedKey == "" {
		return "", fmt.Errorf("encrypted API key is empty")
	}

	return c.Decrypt(encryptedKey)
}
