package test

import (
	"context"
	"os"
	"testing"
	"time"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// MockASRAdapter 模拟 ASR 适配器
type MockASRAdapter struct{}

func (m *MockASRAdapter) ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error) {
	return []*pb.Speaker{
		{
			SpeakerId: "speaker1",
			Sentences: []*pb.Sentence{
				{Text: "Hello world", StartTime: 0.0, EndTime: 1.0},
			},
		},
	}, nil
}

// MockTranslationAdapter 模拟翻译适配器
type MockTranslationAdapter struct{}

func (m *MockTranslationAdapter) Translate(text, sourceLang, targetLang, videoType, apiKey, endpoint string) (string, error) {
	return "你好世界", nil
}

// MockLLMAdapter 模拟 LLM 适配器
type MockLLMAdapter struct{}

func (m *MockLLMAdapter) Polish(text, videoType, customPrompt, apiKey, endpoint string) (string, error) {
	return "Polished: " + text, nil
}

func (m *MockLLMAdapter) Optimize(text, apiKey, endpoint string) (string, error) {
	return "Optimized: " + text, nil
}

// MockVoiceCloningAdapter 模拟声音克隆适配器
type MockVoiceCloningAdapter struct{}

func (m *MockVoiceCloningAdapter) CloneVoice(speakerID, text, referenceAudio, apiKey, endpoint string) (string, error) {
	return "/tmp/cloned_audio.wav", nil
}

// TestAdapterRegistry 测试适配器注册表
func TestAdapterRegistry(t *testing.T) {
	registry := adapters.NewAdapterRegistry()

	// 测试注册 ASR 适配器
	mockASR := &MockASRAdapter{}
	registry.RegisterASR("mock_asr", mockASR)

	// 测试获取 ASR 适配器
	adapter, err := registry.GetASR("mock_asr")
	if err != nil {
		t.Fatalf("Failed to get ASR adapter: %v", err)
	}
	if adapter == nil {
		t.Fatal("ASR adapter is nil")
	}

	// 测试获取不存在的适配器
	_, err = registry.GetASR("nonexistent")
	if err == nil {
		t.Fatal("Expected error when getting nonexistent adapter")
	}

	// 测试列出提供商
	providers := registry.ListASRProviders()
	if len(providers) != 1 || providers[0] != "mock_asr" {
		t.Fatalf("Expected 1 provider 'mock_asr', got %v", providers)
	}
}

// TestAdapterRegistryTranslation 测试翻译适配器注册
func TestAdapterRegistryTranslation(t *testing.T) {
	registry := adapters.NewAdapterRegistry()

	mockTranslation := &MockTranslationAdapter{}
	registry.RegisterTranslation("mock_translation", mockTranslation)

	adapter, err := registry.GetTranslation("mock_translation")
	if err != nil {
		t.Fatalf("Failed to get translation adapter: %v", err)
	}
	if adapter == nil {
		t.Fatal("Translation adapter is nil")
	}

	providers := registry.ListTranslationProviders()
	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}
}

// TestAdapterRegistryLLM 测试 LLM 适配器注册
func TestAdapterRegistryLLM(t *testing.T) {
	registry := adapters.NewAdapterRegistry()

	mockLLM := &MockLLMAdapter{}
	registry.RegisterLLM("mock_llm", mockLLM)

	adapter, err := registry.GetLLM("mock_llm")
	if err != nil {
		t.Fatalf("Failed to get LLM adapter: %v", err)
	}
	if adapter == nil {
		t.Fatal("LLM adapter is nil")
	}

	providers := registry.ListLLMProviders()
	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}
}

// TestAdapterRegistryVoiceCloning 测试声音克隆适配器注册
func TestAdapterRegistryVoiceCloning(t *testing.T) {
	registry := adapters.NewAdapterRegistry()

	mockVoice := &MockVoiceCloningAdapter{}
	registry.RegisterVoiceCloning("mock_voice", mockVoice)

	adapter, err := registry.GetVoiceCloning("mock_voice")
	if err != nil {
		t.Fatalf("Failed to get voice cloning adapter: %v", err)
	}
	if adapter == nil {
		t.Fatal("Voice cloning adapter is nil")
	}

	providers := registry.ListVoiceCloningProviders()
	if len(providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(providers))
	}
}

// TestCryptoManager 测试加密管理器
func TestCryptoManager(t *testing.T) {
	// 设置测试环境变量
	testSecret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	os.Setenv("API_KEY_ENCRYPTION_SECRET", testSecret)
	defer os.Unsetenv("API_KEY_ENCRYPTION_SECRET")

	crypto, err := config.NewCryptoManager()
	if err != nil {
		t.Fatalf("Failed to create crypto manager: %v", err)
	}

	// 测试加密
	plaintext := "test-api-key-12345"
	ciphertext, err := crypto.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("Ciphertext is empty")
	}

	// 测试解密
	decrypted, err := crypto.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("Decrypted text doesn't match: expected %s, got %s", plaintext, decrypted)
	}
}

// TestCryptoManagerInvalidSecret 测试无效的加密密钥
func TestCryptoManagerInvalidSecret(t *testing.T) {
	// 测试空密钥
	os.Unsetenv("API_KEY_ENCRYPTION_SECRET")
	_, err := config.NewCryptoManager()
	if err == nil {
		t.Fatal("Expected error with empty secret")
	}

	// 测试无效长度的密钥
	os.Setenv("API_KEY_ENCRYPTION_SECRET", "short")
	defer os.Unsetenv("API_KEY_ENCRYPTION_SECRET")
	_, err = config.NewCryptoManager()
	if err == nil {
		t.Fatal("Expected error with invalid secret length")
	}
}

// TestRedisClient 测试 Redis 客户端（需要 Redis 运行）
func TestRedisClient(t *testing.T) {
	// 使用内存 Redis 实例，避免依赖外部环境。
	client := config.NewMockRedisClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试设置和获取音色缓存
	speakerID := "test_speaker_123"
	voiceID := "voice_abc_456"
	createdAt := time.Now().Format(time.RFC3339)
	referenceAudio := "/tmp/test_audio.wav"

	if err := client.SetVoiceCache(ctx, speakerID, voiceID, createdAt, referenceAudio); err != nil {
		t.Fatalf("Failed to set voice cache: %v", err)
	}

	// 获取音色缓存
	cache, err := client.GetVoiceCache(ctx, speakerID)
	if err != nil {
		t.Fatalf("Failed to get voice cache: %v", err)
	}

	if cache["voice_id"] != voiceID {
		t.Fatalf("Expected voice_id %s, got %s", voiceID, cache["voice_id"])
	}
	if cache["reference_audio"] != referenceAudio {
		t.Fatalf("Expected reference_audio %s, got %s", referenceAudio, cache["reference_audio"])
	}

	// 删除音色缓存
	if err := client.DeleteVoiceCache(ctx, speakerID); err != nil {
		t.Fatalf("Failed to delete voice cache: %v", err)
	}

	// 验证删除成功
	cache, err = client.GetVoiceCache(ctx, speakerID)
	if err != nil {
		t.Fatalf("Failed to get voice cache after deletion: %v", err)
	}
	if len(cache) != 0 {
		t.Fatal("Expected empty cache after deletion")
	}
}

// TestRedisClientGetNonexistentCache 测试获取不存在的缓存
func TestRedisClientGetNonexistentCache(t *testing.T) {
	if os.Getenv("SKIP_REDIS_TESTS") == "true" {
		t.Skip("Skipping Redis tests")
	}

	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	defer func() {
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
	}()

	client, err := config.NewRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取不存在的缓存应该返回空 map，不是错误
	cache, err := client.GetVoiceCache(ctx, "nonexistent_speaker")
	if err != nil {
		t.Fatalf("Expected no error for nonexistent cache, got: %v", err)
	}
	if len(cache) != 0 {
		t.Fatalf("Expected empty cache, got: %v", cache)
	}
}
