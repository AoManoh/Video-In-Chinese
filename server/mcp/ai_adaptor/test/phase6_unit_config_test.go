package test

import (
	"context"
	"testing"

	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
)

// TestConfigManager_ParseConfig_NewFields 测试新字段解析
func TestConfigManager_ParseConfig_NewFields(t *testing.T) {
	// 创建 Mock Redis 客户端
	redisClient := config.NewMockRedisClient()

	// 设置测试数据
	setTestEncryptionKey(t)
	cryptoManager, err := config.NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager() error = %v", err)
	}
	encrypt := func(value string) string {
		enc, err := cryptoManager.Encrypt(value)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		return enc
	}
	testSettings := map[string]string{
		// ASR 配置
		"asr_provider":      "aliyun",
		"asr_api_key":       encrypt("test-asr-key"),
		"asr_endpoint":      "https://nls-gateway.cn-shanghai.aliyuncs.com",
		"asr_language_code": "zh-CN",
		"asr_region":        "cn-shanghai",

		// 翻译配置
		"translation_provider": "google",
		"translation_api_key":  encrypt("test-translation-key"),

		// 文本润色配置
		"polishing_enabled":    "true",
		"polishing_provider":   "openai-gpt4o",
		"polishing_api_key":    encrypt("test-polishing-key"),
		"polishing_model_name": "gpt-4o",

		// 译文优化配置
		"optimization_enabled":    "true",
		"optimization_provider":   "gemini",
		"optimization_api_key":    encrypt("test-optimization-key"),
		"optimization_model_name": "gemini-1.5-flash",

		// 声音克隆配置
		"voice_cloning_provider":   "aliyun_cosyvoice",
		"voice_cloning_api_key":    encrypt("test-voice-key"),
		"voice_cloning_output_dir": "/tmp/voice-output",

		// 阿里云 OSS 配置
		"aliyun_oss_access_key_id":     encrypt("test-oss-key-id"),
		"aliyun_oss_access_key_secret": encrypt("test-oss-secret"),
		"aliyun_oss_bucket_name":       "test-bucket",
		"aliyun_oss_endpoint":          "oss-cn-shanghai.aliyuncs.com",
		"aliyun_oss_region":            "cn-shanghai",
	}

	// 将测试数据写入 Mock Redis
	for key, value := range testSettings {
		if err := redisClient.HSetField(context.Background(), "app:settings", key, value); err != nil {
			t.Fatalf("HSetField() error = %v", err)
		}
	}

	// 创建 ConfigManager
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 获取配置
	appConfig, err := configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证 ASR 配置
	if appConfig.ASRProvider != "aliyun" {
		t.Errorf("ASRProvider = %v, want aliyun", appConfig.ASRProvider)
	}
	if appConfig.ASRLanguageCode != "zh-CN" {
		t.Errorf("ASRLanguageCode = %v, want zh-CN", appConfig.ASRLanguageCode)
	}
	if appConfig.ASRRegion != "cn-shanghai" {
		t.Errorf("ASRRegion = %v, want cn-shanghai", appConfig.ASRRegion)
	}

	// 验证文本润色配置
	if appConfig.PolishingModelName != "gpt-4o" {
		t.Errorf("PolishingModelName = %v, want gpt-4o", appConfig.PolishingModelName)
	}

	// 验证译文优化配置
	if appConfig.OptimizationModelName != "gemini-1.5-flash" {
		t.Errorf("OptimizationModelName = %v, want gemini-1.5-flash", appConfig.OptimizationModelName)
	}

	// 验证声音克隆配置
	if appConfig.VoiceCloningOutputDir != "/tmp/voice-output" {
		t.Errorf("VoiceCloningOutputDir = %v, want /tmp/voice-output", appConfig.VoiceCloningOutputDir)
	}

	// 验证阿里云 OSS 配置
	if appConfig.AliyunOSSAccessKeyID != "test-oss-key-id" {
		t.Errorf("AliyunOSSAccessKeyID = %v, want test-oss-key-id", appConfig.AliyunOSSAccessKeyID)
	}
	if appConfig.AliyunOSSAccessKeySecret != "test-oss-secret" {
		t.Errorf("AliyunOSSAccessKeySecret = %v, want test-oss-secret", appConfig.AliyunOSSAccessKeySecret)
	}
	if appConfig.AliyunOSSBucketName != "test-bucket" {
		t.Errorf("AliyunOSSBucketName = %v, want test-bucket", appConfig.AliyunOSSBucketName)
	}
	if appConfig.AliyunOSSEndpoint != "oss-cn-shanghai.aliyuncs.com" {
		t.Errorf("AliyunOSSEndpoint = %v, want oss-cn-shanghai.aliyuncs.com", appConfig.AliyunOSSEndpoint)
	}
	if appConfig.AliyunOSSRegion != "cn-shanghai" {
		t.Errorf("AliyunOSSRegion = %v, want cn-shanghai", appConfig.AliyunOSSRegion)
	}

	t.Logf("配置解析成功，所有新字段验证通过")
}

// TestConfigManager_ParseConfig_EncryptedOSSConfig 测试 OSS 配置解密
func TestConfigManager_ParseConfig_EncryptedOSSConfig(t *testing.T) {
	// 创建 Mock Redis 客户端
	redisClient := config.NewMockRedisClient()

	// 创建 CryptoManager
	setTestEncryptionKey(t)
	cryptoManager, err := config.NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager() error = %v", err)
	}
	encrypt := func(value string) string {
		enc, err := cryptoManager.Encrypt(value)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		return enc
	}

	encryptedKeyID := encrypt("test-oss-key-id")
	encryptedSecret := encrypt("test-oss-secret")

	// 设置测试数据（使用加密的凭证）
	testSettings := map[string]string{
		"asr_provider":                 "aliyun",
		"asr_api_key":                  encrypt("test-asr-key"),
		"translation_provider":         "google",
		"translation_api_key":          encrypt("test-translation-key"),
		"voice_cloning_provider":       "aliyun_cosyvoice",
		"voice_cloning_api_key":        encrypt("test-voice-key"),
		"aliyun_oss_access_key_id":     encryptedKeyID,
		"aliyun_oss_access_key_secret": encryptedSecret,
		"aliyun_oss_bucket_name":       "test-bucket",
		"aliyun_oss_endpoint":          "oss-cn-shanghai.aliyuncs.com",
	}

	// 将测试数据写入 Mock Redis
	for key, value := range testSettings {
		if err := redisClient.HSetField(context.Background(), "app:settings", key, value); err != nil {
			t.Fatalf("HSetField() error = %v", err)
		}
	}

	// 创建 ConfigManager
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 获取配置
	appConfig, err := configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证解密后的 OSS 配置
	if appConfig.AliyunOSSAccessKeyID != "test-oss-key-id" {
		t.Errorf("AliyunOSSAccessKeyID = %v, want test-oss-key-id", appConfig.AliyunOSSAccessKeyID)
	}
	if appConfig.AliyunOSSAccessKeySecret != "test-oss-secret" {
		t.Errorf("AliyunOSSAccessKeySecret = %v, want test-oss-secret", appConfig.AliyunOSSAccessKeySecret)
	}

	t.Logf("OSS 配置解密成功")
}

// TestConfigManager_ParseConfig_MissingNewFields 测试缺失新字段时的默认值
func TestConfigManager_ParseConfig_MissingNewFields(t *testing.T) {
	// 创建 Mock Redis 客户端
	redisClient := config.NewMockRedisClient()

	// 设置最小测试数据（不包含新字段）
	setTestEncryptionKey(t)
	cryptoManager, err := config.NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager() error = %v", err)
	}
	encrypt := func(value string) string {
		enc, err := cryptoManager.Encrypt(value)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		return enc
	}
	testSettings := map[string]string{
		"asr_provider":           "aliyun",
		"asr_api_key":            encrypt("test-asr-key"),
		"translation_provider":   "google",
		"translation_api_key":    encrypt("test-translation-key"),
		"voice_cloning_provider": "aliyun_cosyvoice",
		"voice_cloning_api_key":  encrypt("test-voice-key"),
	}

	// 将测试数据写入 Mock Redis
	for key, value := range testSettings {
		if err := redisClient.HSetField(context.Background(), "app:settings", key, value); err != nil {
			t.Fatalf("HSetField() error = %v", err)
		}
	}

	// 创建 ConfigManager
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 获取配置
	appConfig, err := configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证新字段的默认值（应该为空字符串）
	if appConfig.ASRLanguageCode != "" {
		t.Errorf("ASRLanguageCode = %v, want empty string", appConfig.ASRLanguageCode)
	}
	if appConfig.ASRRegion != "" {
		t.Errorf("ASRRegion = %v, want empty string", appConfig.ASRRegion)
	}
	if appConfig.PolishingModelName != "" {
		t.Errorf("PolishingModelName = %v, want empty string", appConfig.PolishingModelName)
	}
	if appConfig.OptimizationModelName != "" {
		t.Errorf("OptimizationModelName = %v, want empty string", appConfig.OptimizationModelName)
	}
	if appConfig.VoiceCloningOutputDir != "" {
		t.Errorf("VoiceCloningOutputDir = %v, want empty string", appConfig.VoiceCloningOutputDir)
	}
	if appConfig.AliyunOSSAccessKeyID != "" {
		t.Errorf("AliyunOSSAccessKeyID = %v, want empty string", appConfig.AliyunOSSAccessKeyID)
	}

	t.Logf("缺失新字段时默认值验证通过")
}

// TestConfigManager_Cache_NewFields 测试新字段的缓存机制
func TestConfigManager_Cache_NewFields(t *testing.T) {
	// 创建 Mock Redis 客户端
	redisClient := config.NewMockRedisClient()

	// 设置测试数据
	setTestEncryptionKey(t)
	cryptoManager, err := config.NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager() error = %v", err)
	}
	encrypt := func(value string) string {
		enc, err := cryptoManager.Encrypt(value)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		return enc
	}
	testSettings := map[string]string{
		"asr_provider":           "aliyun",
		"asr_api_key":            encrypt("test-asr-key"),
		"asr_language_code":      "zh-CN",
		"asr_region":             "cn-shanghai",
		"translation_provider":   "google",
		"translation_api_key":    encrypt("test-translation-key"),
		"voice_cloning_provider": "aliyun_cosyvoice",
		"voice_cloning_api_key":  encrypt("test-voice-key"),
	}

	// 将测试数据写入 Mock Redis
	for key, value := range testSettings {
		if err := redisClient.HSetField(context.Background(), "app:settings", key, value); err != nil {
			t.Fatalf("HSetField() error = %v", err)
		}
	}

	// 创建 ConfigManager
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 第一次获取配置（从 Redis 读取）
	appConfig1, err := configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证第一次读取的配置
	if appConfig1.ASRLanguageCode != "zh-CN" {
		t.Errorf("ASRLanguageCode = %v, want zh-CN", appConfig1.ASRLanguageCode)
	}

	// 修改 Redis 中的配置
	if err := redisClient.HSetField(context.Background(), "app:settings", "asr_language_code", "en-US"); err != nil {
		t.Fatalf("HSetField() error = %v", err)
	}

	// 第二次获取配置（从缓存读取，应该还是旧值）
	appConfig2, err := configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证第二次读取的配置（应该还是缓存的旧值）
	if appConfig2.ASRLanguageCode != "zh-CN" {
		t.Errorf("ASRLanguageCode = %v, want zh-CN (cached value)", appConfig2.ASRLanguageCode)
	}

	// 清除缓存
	configManager.InvalidateCache()

	// 第三次获取配置（从 Redis 重新读取）
	appConfig3, err := configManager.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证第三次读取的配置（应该是新值）
	if appConfig3.ASRLanguageCode != "en-US" {
		t.Errorf("ASRLanguageCode = %v, want en-US (new value)", appConfig3.ASRLanguageCode)
	}

	t.Logf("缓存机制验证通过")
}
