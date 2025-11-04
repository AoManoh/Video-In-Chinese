package test

import (
	"context"
	"os"
	"testing"
	"time"

	"video-in-chinese/ai_adaptor/internal/config"
	"video-in-chinese/ai_adaptor/internal/voice_cache"
)

// TestVoiceManagerMemoryCache 测试内存缓存功能
func TestVoiceManagerMemoryCache(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("VOICE_REGISTER_TIMEOUT", "10")
	os.Setenv("VOICE_REGISTER_RETRY", "2")
	os.Setenv("VOICE_REGISTER_RETRY_INTERVAL", "1")
	defer func() {
		os.Unsetenv("VOICE_REGISTER_TIMEOUT")
		os.Unsetenv("VOICE_REGISTER_RETRY")
		os.Unsetenv("VOICE_REGISTER_RETRY_INTERVAL")
	}()

	// 创建 Mock Redis 客户端
	redisClient, err := createMockRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	// 创建 VoiceManager
	vm := voice_cache.NewVoiceManager(redisClient)

	ctx := context.Background()
	speakerID := "test_speaker_001"
	referenceAudio := "/tmp/test_audio.wav"
	apiKey := "test_api_key"
	endpoint := "https://test.aliyuncs.com"

	// 第一次调用：应该注册新音色
	voiceID1, err := vm.GetOrRegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to register voice: %v", err)
	}
	if voiceID1 == "" {
		t.Fatal("Voice ID is empty")
	}
	t.Logf("First call: voice_id=%s", voiceID1)

	// 第二次调用：应该从内存缓存中获取
	voiceID2, err := vm.GetOrRegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to get voice from cache: %v", err)
	}
	if voiceID2 != voiceID1 {
		t.Fatalf("Expected voice_id=%s from cache, got %s", voiceID1, voiceID2)
	}
	t.Logf("Second call (from memory cache): voice_id=%s", voiceID2)
}

// TestVoiceManagerRegisterVoice 测试音色注册功能
func TestVoiceManagerRegisterVoice(t *testing.T) {
	os.Setenv("VOICE_REGISTER_TIMEOUT", "10")
	defer os.Unsetenv("VOICE_REGISTER_TIMEOUT")

	redisClient, err := createMockRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	vm := voice_cache.NewVoiceManager(redisClient)

	ctx := context.Background()
	speakerID := "test_speaker_002"
	referenceAudio := "/tmp/test_audio_002.wav"
	apiKey := "test_api_key"
	endpoint := "https://test.aliyuncs.com"

	// 注册音色
	voiceID, err := vm.RegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to register voice: %v", err)
	}
	if voiceID == "" {
		t.Fatal("Voice ID is empty")
	}
	t.Logf("Voice registered: speaker_id=%s, voice_id=%s", speakerID, voiceID)

	// 验证音色 ID 格式（应该包含 speaker_id）
	if len(voiceID) < 10 {
		t.Fatalf("Voice ID too short: %s", voiceID)
	}
}

// TestVoiceManagerPollVoiceStatus 测试音色轮询功能
func TestVoiceManagerPollVoiceStatus(t *testing.T) {
	os.Setenv("VOICE_REGISTER_TIMEOUT", "5")
	defer os.Unsetenv("VOICE_REGISTER_TIMEOUT")

	redisClient, err := createMockRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	vm := voice_cache.NewVoiceManager(redisClient)

	ctx := context.Background()
	voiceID := "test_voice_id_003"
	apiKey := "test_api_key"
	endpoint := "https://test.aliyuncs.com"

	// 测试轮询（临时占位符实现总是返回 OK）
	err = vm.PollVoiceStatus(ctx, voiceID, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to poll voice status: %v", err)
	}
	t.Logf("Voice status polling completed: voice_id=%s", voiceID)
}

// TestVoiceManagerHandleVoiceNotFound 测试缓存失效处理（404）
func TestVoiceManagerHandleVoiceNotFound(t *testing.T) {
	os.Setenv("VOICE_REGISTER_TIMEOUT", "10")
	defer os.Unsetenv("VOICE_REGISTER_TIMEOUT")

	redisClient, err := createMockRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	vm := voice_cache.NewVoiceManager(redisClient)

	ctx := context.Background()
	speakerID := "test_speaker_004"
	referenceAudio := "/tmp/test_audio_004.wav"
	apiKey := "test_api_key"
	endpoint := "https://test.aliyuncs.com"

	// 先注册一个音色
	voiceID1, err := vm.RegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to register voice: %v", err)
	}
	t.Logf("Initial voice registered: voice_id=%s", voiceID1)

	// 模拟音色失效（404），清除缓存并重新注册
	voiceID2, err := vm.HandleVoiceNotFound(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to handle voice not found: %v", err)
	}
	if voiceID2 == "" {
		t.Fatal("Re-registered voice ID is empty")
	}
	t.Logf("Voice re-registered after 404: voice_id=%s", voiceID2)
}

// TestVoiceManagerConcurrentAccess 测试并发安全
func TestVoiceManagerConcurrentAccess(t *testing.T) {
	os.Setenv("VOICE_REGISTER_TIMEOUT", "10")
	defer os.Unsetenv("VOICE_REGISTER_TIMEOUT")

	redisClient, err := createMockRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	vm := voice_cache.NewVoiceManager(redisClient)

	ctx := context.Background()
	speakerID := "test_speaker_005"
	referenceAudio := "/tmp/test_audio_005.wav"
	apiKey := "test_api_key"
	endpoint := "https://test.aliyuncs.com"

	// 并发调用 GetOrRegisterVoice
	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			voiceID, err := vm.GetOrRegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
			if err != nil {
				errors <- err
				return
			}
			results <- voiceID
		}()
	}

	// 收集结果
	var voiceIDs []string
	for i := 0; i < numGoroutines; i++ {
		select {
		case voiceID := <-results:
			voiceIDs = append(voiceIDs, voiceID)
		case err := <-errors:
			t.Fatalf("Concurrent call failed: %v", err)
		case <-time.After(15 * time.Second):
			t.Fatal("Concurrent calls timeout")
		}
	}

	// 验证所有 goroutine 返回相同的 voice_id
	if len(voiceIDs) != numGoroutines {
		t.Fatalf("Expected %d results, got %d", numGoroutines, len(voiceIDs))
	}

	firstVoiceID := voiceIDs[0]
	for i, voiceID := range voiceIDs {
		if voiceID != firstVoiceID {
			t.Fatalf("Goroutine %d returned different voice_id: expected %s, got %s", i, firstVoiceID, voiceID)
		}
	}

	t.Logf("Concurrent access test passed: all %d goroutines returned voice_id=%s", numGoroutines, firstVoiceID)
}

// TestVoiceManagerRedisCacheIntegration 测试 Redis 缓存集成
func TestVoiceManagerRedisCacheIntegration(t *testing.T) {
	os.Setenv("VOICE_REGISTER_TIMEOUT", "10")
	defer os.Unsetenv("VOICE_REGISTER_TIMEOUT")

	// 创建真实的 Redis 客户端
	redisClient, err := config.NewRedisClient()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer redisClient.Close()

	vm := voice_cache.NewVoiceManager(redisClient)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	speakerID := "test_speaker_redis_006"
	referenceAudio := "/tmp/test_audio_006.wav"
	apiKey := "test_api_key"
	endpoint := "https://test.aliyuncs.com"

	// 清理旧缓存
	redisClient.DeleteVoiceCache(ctx, speakerID)

	// 第一次调用：注册新音色并缓存到 Redis
	voiceID1, err := vm.GetOrRegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to register voice: %v", err)
	}
	t.Logf("First call: voice_id=%s", voiceID1)

	// 创建新的 VoiceManager（清空内存缓存）
	vm2 := voice_cache.NewVoiceManager(redisClient)

	// 第二次调用：应该从 Redis 缓存中加载
	voiceID2, err := vm2.GetOrRegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		t.Fatalf("Failed to get voice from Redis cache: %v", err)
	}
	if voiceID2 != voiceID1 {
		t.Fatalf("Expected voice_id=%s from Redis cache, got %s", voiceID1, voiceID2)
	}
	t.Logf("Second call (from Redis cache): voice_id=%s", voiceID2)

	// 清理测试数据
	redisClient.DeleteVoiceCache(ctx, speakerID)
}

// createMockRedisClient 创建 Mock Redis 客户端
func createMockRedisClient() (*config.RedisClient, error) {
	return config.NewRedisClient()
}

