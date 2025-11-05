package test

import (
	"os"
	"strings"
	"testing"
)

// setTestEncryptionKey 配置 CryptoManager 所需的环境变量。
func setTestEncryptionKey(t *testing.T) {
	t.Helper()
	secret := strings.Repeat("01", 32) // 32 bytes hex
	if err := os.Setenv("API_KEY_ENCRYPTION_SECRET", secret); err != nil {
		t.Fatalf("failed to set API_KEY_ENCRYPTION_SECRET: %v", err)
	}
	t.Cleanup(func() {
		os.Unsetenv("API_KEY_ENCRYPTION_SECRET")
	})
}
