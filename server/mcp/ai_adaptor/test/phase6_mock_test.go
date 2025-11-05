package test

import (
	"context"
	"testing"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/adapters/asr"
	"video-in-chinese/ai_adaptor/internal/config"
)

func encrypt(value string, cm *config.CryptoManager, t *testing.T) string {
	t.Helper()
	enc, err := cm.Encrypt(value)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	return enc
}

// TestOSSUpload_DegradationStrategy_MissingConfig 测试 OSS 上传降级策略（配置不完整）
func TestOSSUpload_DegradationStrategy_MissingConfig(t *testing.T) {
	// 现阶段 OSS 上传失败直接返回错误，保留占位并标记为跳过。
	t.Skip("OSS 上传失败现在直接返回错误，跳过旧降级策略用例")
}

// TestOSSUpload_DegradationStrategy_InvalidCredentials 测试 OSS 上传降级策略（无效凭证）
func TestOSSUpload_DegradationStrategy_InvalidCredentials(t *testing.T) {
	// 现阶段 OSS 上传失败直接返回错误，保留占位并标记为跳过。
	t.Skip("OSS 上传失败现在直接返回错误，跳过旧降级策略用例")
}

// TestAdapterRegistry_DynamicSelection 测试根据配置动态选择适配器
func TestAdapterRegistry_DynamicSelection(t *testing.T) {
	// 创建适配器注册表
	registry := adapters.NewAdapterRegistry()

	// 注册 ASR 适配器
	aliyunASR := asr.NewAliyunASRAdapter()
	azureASR := asr.NewAzureASRAdapter()
	googleASR := asr.NewGoogleASRAdapter()

	registry.RegisterASR("aliyun", aliyunASR)
	registry.RegisterASR("azure", azureASR)
	registry.RegisterASR("google", googleASR)

	// 测试动态选择阿里云 ASR 适配器
	selectedAdapter, err := registry.GetASR("aliyun")
	if err != nil {
		t.Fatalf("GetASRAdapter('aliyun') error = %v", err)
	}
	if selectedAdapter != aliyunASR {
		t.Errorf("GetASRAdapter('aliyun') returned wrong adapter")
	}
	t.Logf("成功选择阿里云 ASR 适配器")

	// 测试动态选择 Azure ASR 适配器
	selectedAdapter, err = registry.GetASR("azure")
	if err != nil {
		t.Fatalf("GetASRAdapter('azure') error = %v", err)
	}
	if selectedAdapter != azureASR {
		t.Errorf("GetASRAdapter('azure') returned wrong adapter")
	}
	t.Logf("成功选择 Azure ASR 适配器")

	// 测试动态选择 Google ASR 适配器
	selectedAdapter, err = registry.GetASR("google")
	if err != nil {
		t.Fatalf("GetASRAdapter('google') error = %v", err)
	}
	if selectedAdapter != googleASR {
		t.Errorf("GetASRAdapter('google') returned wrong adapter")
	}
	t.Logf("成功选择 Google ASR 适配器")

	// 测试选择不存在的适配器
	_, err = registry.GetASR("nonexistent")
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
	setTestEncryptionKey(t)
	cryptoManager, err := config.NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager() error = %v", err)
	}

	encryptedASRKey := encrypt("test-aliyun-key", cryptoManager, t)

	testSettings := map[string]string{
		"asr_provider":           "aliyun",
		"asr_api_key":            encryptedASRKey,
		"asr_endpoint":           "https://nls-gateway.cn-shanghai.aliyuncs.com",
		"translation_provider":   "google",
		"translation_api_key":    encrypt("test-translation-key", cryptoManager, t),
		"voice_cloning_provider": "aliyun_cosyvoice",
		"voice_cloning_api_key":  encrypt("test-voice-key", cryptoManager, t),
	}

	for key, value := range testSettings {
		if err := redisClient.HSetField(context.Background(), "app:settings", key, value); err != nil {
			t.Fatalf("HSetField() error = %v", err)
		}
	}

	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 获取配置
	appConfig, err := configManager.GetConfig(context.Background())
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
	aliyunASR := asr.NewAliyunASRAdapter()
	registry.RegisterASR(appConfig.ASRProvider, aliyunASR)

	// 根据配置选择适配器
	selectedAdapter, err := registry.GetASR(appConfig.ASRProvider)
	if err != nil {
		t.Fatalf("GetASRAdapter() error = %v", err)
	}

	if selectedAdapter != aliyunASR {
		t.Errorf("GetASRAdapter() returned wrong adapter")
	}

	t.Logf("根据配置成功选择适配器: %s", appConfig.ASRProvider)

	// 修改配置为 Azure ASR
	if err := redisClient.HSetField(context.Background(), "app:settings", "asr_provider", "azure"); err != nil {
		t.Fatalf("HSetField() error = %v", err)
	}
	if err := redisClient.HSetField(context.Background(), "app:settings", "asr_api_key", encrypt("test-azure-key", cryptoManager, t)); err != nil {
		t.Fatalf("HSetField() error = %v", err)
	}
	if err := redisClient.HSetField(context.Background(), "app:settings", "asr_region", "eastus"); err != nil {
		t.Fatalf("HSetField() error = %v", err)
	}

	// 清除缓存
	configManager.InvalidateCache()

	// 重新获取配置
	appConfig, err = configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证配置
	if appConfig.ASRProvider != "azure" {
		t.Errorf("ASRProvider = %v, want azure", appConfig.ASRProvider)
	}

	// 根据新配置注册适配器
	azureASR := asr.NewAzureASRAdapter()
	registry.RegisterASR(appConfig.ASRProvider, azureASR)

	// 根据新配置选择适配器
	selectedAdapter, err = registry.GetASR(appConfig.ASRProvider)
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
	t.Skip("OSS 上传失败现在直接返回错误，跳过旧降级策略用例")
}
