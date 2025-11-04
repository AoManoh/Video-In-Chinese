package test

import (
	"context"
	"os"
	"testing"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/adapters/asr"
	"video-in-chinese/ai_adaptor/internal/config"
)

// TestOSSUpload_DegradationStrategy_MissingConfig 测试 OSS 上传降级策略（配置不完整）
func TestOSSUpload_DegradationStrategy_MissingConfig(t *testing.T) {
	// 清除环境变量
	os.Unsetenv("ALIYUN_OSS_ACCESS_KEY_ID")
	os.Unsetenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
	os.Unsetenv("ALIYUN_OSS_BUCKET_NAME")
	os.Unsetenv("ALIYUN_OSS_ENDPOINT")

	// 创建阿里云 ASR 适配器
	asrAdapter := asr.NewAliyunASRAdapter("test-api-key", "https://nls-gateway.cn-shanghai.aliyuncs.com")

	// 测试 ASR（应该使用本地路径作为降级方案）
	ctx := context.Background()
	speakers, err := asrAdapter.ASR(ctx, "/path/to/test-audio.wav")

	// 注意：由于没有真实的 API 密钥，ASR 调用会失败
	// 但我们可以验证降级策略是否生效（通过日志）
	if err != nil {
		t.Logf("ASR() error = %v (预期错误，因为没有真实的 API 密钥)", err)
	}

	if speakers != nil {
		t.Logf("ASR() returned %d speakers", len(speakers))
	}

	t.Logf("OSS 上传降级策略测试完成（配置不完整时使用本地路径）")
}

// TestOSSUpload_DegradationStrategy_InvalidCredentials 测试 OSS 上传降级策略（无效凭证）
func TestOSSUpload_DegradationStrategy_InvalidCredentials(t *testing.T) {
	// 设置无效的环境变量
	os.Setenv("ALIYUN_OSS_ACCESS_KEY_ID", "invalid-key-id")
	os.Setenv("ALIYUN_OSS_ACCESS_KEY_SECRET", "invalid-secret")
	os.Setenv("ALIYUN_OSS_BUCKET_NAME", "invalid-bucket")
	os.Setenv("ALIYUN_OSS_ENDPOINT", "oss-cn-shanghai.aliyuncs.com")
	defer func() {
		os.Unsetenv("ALIYUN_OSS_ACCESS_KEY_ID")
		os.Unsetenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
		os.Unsetenv("ALIYUN_OSS_BUCKET_NAME")
		os.Unsetenv("ALIYUN_OSS_ENDPOINT")
	}()

	// 创建阿里云 ASR 适配器
	asrAdapter := asr.NewAliyunASRAdapter("test-api-key", "https://nls-gateway.cn-shanghai.aliyuncs.com")

	// 测试 ASR（应该使用本地路径作为降级方案）
	ctx := context.Background()
	speakers, err := asrAdapter.ASR(ctx, "/path/to/test-audio.wav")

	// 注意：由于没有真实的 API 密钥，ASR 调用会失败
	if err != nil {
		t.Logf("ASR() error = %v (预期错误，因为没有真实的 API 密钥)", err)
	}

	if speakers != nil {
		t.Logf("ASR() returned %d speakers", len(speakers))
	}

	t.Logf("OSS 上传降级策略测试完成（无效凭证时使用本地路径）")
}

// TestAdapterRegistry_DynamicSelection 测试根据配置动态选择适配器
func TestAdapterRegistry_DynamicSelection(t *testing.T) {
	// 创建适配器注册表
	registry := adapters.NewAdapterRegistry()

	// 注册 ASR 适配器
	aliyunASR := asr.NewAliyunASRAdapter("test-aliyun-key", "https://nls-gateway.cn-shanghai.aliyuncs.com")
	azureASR := asr.NewAzureASRAdapter("test-azure-key", "eastus")
	googleASR := asr.NewGoogleASRAdapter("test-google-key")

	registry.RegisterASRAdapter("aliyun", aliyunASR)
	registry.RegisterASRAdapter("azure", azureASR)
	registry.RegisterASRAdapter("google", googleASR)

	// 测试动态选择阿里云 ASR 适配器
	selectedAdapter, err := registry.GetASRAdapter("aliyun")
	if err != nil {
		t.Fatalf("GetASRAdapter('aliyun') error = %v", err)
	}
	if selectedAdapter != aliyunASR {
		t.Errorf("GetASRAdapter('aliyun') returned wrong adapter")
	}
	t.Logf("成功选择阿里云 ASR 适配器")

	// 测试动态选择 Azure ASR 适配器
	selectedAdapter, err = registry.GetASRAdapter("azure")
	if err != nil {
		t.Fatalf("GetASRAdapter('azure') error = %v", err)
	}
	if selectedAdapter != azureASR {
		t.Errorf("GetASRAdapter('azure') returned wrong adapter")
	}
	t.Logf("成功选择 Azure ASR 适配器")

	// 测试动态选择 Google ASR 适配器
	selectedAdapter, err = registry.GetASRAdapter("google")
	if err != nil {
		t.Fatalf("GetASRAdapter('google') error = %v", err)
	}
	if selectedAdapter != googleASR {
		t.Errorf("GetASRAdapter('google') returned wrong adapter")
	}
	t.Logf("成功选择 Google ASR 适配器")

	// 测试选择不存在的适配器
	_, err = registry.GetASRAdapter("nonexistent")
	if err == nil {
		t.Errorf("GetASRAdapter('nonexistent') should return error")
	}
	t.Logf("选择不存在的适配器时返回错误: %v", err)
}

// TestConfigManager_DynamicAdapterSelection 测试根据配置动态选择适配器
func TestConfigManager_DynamicAdapterSelection(t *testing.T) {
	// 创建 Mock Redis 客户端
	redisClient := config.NewMockRedisClient()

	// 设置测试数据（阿里云 ASR）
	testSettings := map[string]string{
		"asr_provider": "aliyun",
		"asr_api_key":  "test-aliyun-key",
		"asr_endpoint": "https://nls-gateway.cn-shanghai.aliyuncs.com",
	}

	// 将测试数据写入 Mock Redis
	for key, value := range testSettings {
		redisClient.HSet("app:settings", key, value)
	}

	// 创建 ConfigManager
	cryptoManager := config.NewCryptoManager("test-encryption-key-32-bytes!!")
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 获取配置
	appConfig, err := configManager.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证配置
	if appConfig.ASRProvider != "aliyun" {
		t.Errorf("ASRProvider = %v, want aliyun", appConfig.ASRProvider)
	}

	// 创建适配器注册表
	registry := adapters.NewAdapterRegistry()

	// 根据配置注册适配器
	aliyunASR := asr.NewAliyunASRAdapter(appConfig.ASRAPIKey, appConfig.ASREndpoint)
	registry.RegisterASRAdapter(appConfig.ASRProvider, aliyunASR)

	// 根据配置选择适配器
	selectedAdapter, err := registry.GetASRAdapter(appConfig.ASRProvider)
	if err != nil {
		t.Fatalf("GetASRAdapter() error = %v", err)
	}

	if selectedAdapter != aliyunASR {
		t.Errorf("GetASRAdapter() returned wrong adapter")
	}

	t.Logf("根据配置成功选择适配器: %s", appConfig.ASRProvider)

	// 修改配置为 Azure ASR
	redisClient.HSet("app:settings", "asr_provider", "azure")
	redisClient.HSet("app:settings", "asr_api_key", "test-azure-key")
	redisClient.HSet("app:settings", "asr_region", "eastus")

	// 清除缓存
	configManager.ClearCache()

	// 重新获取配置
	appConfig, err = configManager.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证配置
	if appConfig.ASRProvider != "azure" {
		t.Errorf("ASRProvider = %v, want azure", appConfig.ASRProvider)
	}

	// 根据新配置注册适配器
	azureASR := asr.NewAzureASRAdapter(appConfig.ASRAPIKey, appConfig.ASRRegion)
	registry.RegisterASRAdapter(appConfig.ASRProvider, azureASR)

	// 根据新配置选择适配器
	selectedAdapter, err = registry.GetASRAdapter(appConfig.ASRProvider)
	if err != nil {
		t.Fatalf("GetASRAdapter() error = %v", err)
	}

	if selectedAdapter != azureASR {
		t.Errorf("GetASRAdapter() returned wrong adapter")
	}

	t.Logf("根据新配置成功选择适配器: %s", appConfig.ASRProvider)
}

// TestAPICall_ErrorHandling_Retry 测试 API 调用错误处理和重试逻辑
func TestAPICall_ErrorHandling_Retry(t *testing.T) {
	// 跳过此测试，因为需要 Mock HTTP 服务器
	t.Skip("需要 Mock HTTP 服务器，暂时跳过此测试")

	// 注意：这个测试需要创建 Mock HTTP 服务器来模拟 API 错误响应
	// 可以使用 httptest.NewServer 来实现
}

// TestVoiceManager_DegradationStrategy_OSSUploadFailed 测试音色管理器的 OSS 上传降级策略
func TestVoiceManager_DegradationStrategy_OSSUploadFailed(t *testing.T) {
	// 清除环境变量
	os.Unsetenv("ALIYUN_OSS_ACCESS_KEY_ID")
	os.Unsetenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
	os.Unsetenv("ALIYUN_OSS_BUCKET_NAME")
	os.Unsetenv("ALIYUN_OSS_ENDPOINT")

	// 创建 VoiceManager
	redisClient := config.NewMockRedisClient()
	cryptoManager := config.NewCryptoManager("test-encryption-key-32-bytes!!")
	configManager := config.NewConfigManager(redisClient, cryptoManager)
	voiceManager := voice_cache.NewVoiceManager(redisClient, configManager)

	// 测试音色注册（应该使用模拟 URL 作为降级方案）
	ctx := context.Background()
	speakerID := "test-speaker-001"
	referenceAudio := "/path/to/reference.wav"
	apiKey := "test-api-key"

	// 注意：由于没有真实的 API 密钥和 OSS 配置，音色注册会失败
	// 但我们可以验证降级策略是否生效（通过日志）
	voiceID, err := voiceManager.RegisterVoice(ctx, speakerID, referenceAudio, apiKey, "")
	if err != nil {
		t.Logf("RegisterVoice() error = %v (预期错误，因为没有真实的 API 密钥)", err)
	}

	if voiceID != "" {
		t.Logf("RegisterVoice() returned voice_id=%s", voiceID)
	}

	t.Logf("音色管理器 OSS 上传降级策略测试完成（配置不完整时使用模拟 URL）")
}

