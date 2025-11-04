package test

import (
	"context"
	"os"
	"testing"
	"time"

	"video-in-chinese/ai_adaptor/internal/config"
	"video-in-chinese/ai_adaptor/internal/utils"
	"video-in-chinese/ai_adaptor/internal/voice_cache"
)

// TestConfigManager_Integration_RedisReadWrite 测试配置读取和写入（需要 Redis）
func TestConfigManager_Integration_RedisReadWrite(t *testing.T) {
	// 跳过此测试，因为需要真实的 Redis 环境
	t.Skip("需要真实的 Redis 环境，跳过此测试")

	// 创建 Redis 客户端（连接到本地 Redis）
	redisClient := config.NewRedisClient("localhost:6379", "", 0)

	// 创建 CryptoManager
	cryptoManager := config.NewCryptoManager("test-encryption-key-32-bytes!!")

	// 创建 ConfigManager
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 设置测试配置
	testSettings := map[string]string{
		"asr_provider":      "aliyun",
		"asr_api_key":       "test-asr-key",
		"asr_language_code": "zh-CN",
		"asr_region":        "cn-shanghai",
	}

	// 写入配置到 Redis
	for key, value := range testSettings {
		if err := redisClient.HSet("app:settings", key, value); err != nil {
			t.Fatalf("HSet() error = %v", err)
		}
	}

	// 读取配置
	appConfig, err := configManager.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证配置
	if appConfig.ASRProvider != "aliyun" {
		t.Errorf("ASRProvider = %v, want aliyun", appConfig.ASRProvider)
	}
	if appConfig.ASRLanguageCode != "zh-CN" {
		t.Errorf("ASRLanguageCode = %v, want zh-CN", appConfig.ASRLanguageCode)
	}

	t.Logf("配置读取成功")
}

// TestVoiceManager_Integration_CacheReadWrite 测试音色缓存读写（需要 Redis）
func TestVoiceManager_Integration_CacheReadWrite(t *testing.T) {
	// 跳过此测试，因为需要真实的 Redis 环境
	t.Skip("需要真实的 Redis 环境，跳过此测试")

	// 创建 Redis 客户端
	redisClient := config.NewRedisClient("localhost:6379", "", 0)

	// 创建 ConfigManager
	cryptoManager := config.NewCryptoManager("test-encryption-key-32-bytes!!")
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 创建 VoiceManager
	voiceManager := voice_cache.NewVoiceManager(redisClient, configManager)

	// 测试音色缓存写入
	ctx := context.Background()
	speakerID := "test-speaker-001"
	voiceID := "test-voice-id-12345"

	err := voiceManager.CacheVoice(ctx, speakerID, voiceID)
	if err != nil {
		t.Fatalf("CacheVoice() error = %v", err)
	}

	// 测试音色缓存读取
	cachedVoiceID, err := voiceManager.GetCachedVoice(ctx, speakerID)
	if err != nil {
		t.Fatalf("GetCachedVoice() error = %v", err)
	}

	if cachedVoiceID != voiceID {
		t.Errorf("GetCachedVoice() = %v, want %v", cachedVoiceID, voiceID)
	}

	t.Logf("音色缓存读写成功")
}

// TestOSSUploader_Integration_RealUpload 测试真实 OSS 上传（需要 OSS 环境）
func TestOSSUploader_Integration_RealUpload(t *testing.T) {
	// 跳过此测试，因为需要真实的 OSS 环境
	t.Skip("需要真实的 OSS 环境，跳过此测试")

	// 从环境变量读取 OSS 配置
	accessKeyID := os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("ALIYUN_OSS_BUCKET_NAME")
	endpoint := os.Getenv("ALIYUN_OSS_ENDPOINT")

	if accessKeyID == "" || accessKeySecret == "" || bucketName == "" || endpoint == "" {
		t.Skip("OSS 配置不完整，跳过此测试")
	}

	// 创建 OSS 上传器
	uploader, err := utils.NewOSSUploader(accessKeyID, accessKeySecret, endpoint, bucketName)
	if err != nil {
		t.Fatalf("NewOSSUploader() error = %v", err)
	}

	// 创建临时测试文件
	tmpFile, err := os.CreateTemp("", "test-audio-*.wav")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入测试数据
	testData := []byte("test audio data")
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	tmpFile.Close()

	// 生成对象键
	objectKey := utils.GenerateObjectKey(tmpFile.Name(), "test-upload")

	// 上传文件
	ctx := context.Background()
	publicURL, err := uploader.UploadFile(ctx, tmpFile.Name(), objectKey)
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}

	if publicURL == "" {
		t.Errorf("UploadFile() returned empty URL")
	}

	t.Logf("文件上传成功: %s", publicURL)

	// 清理：删除上传的文件
	if err := uploader.DeleteFile(ctx, objectKey); err != nil {
		t.Logf("DeleteFile() error = %v (可能文件已被删除)", err)
	}
}

// TestVoiceManager_Integration_RealAPI 测试真实 CosyVoice API（需要 API 密钥）
func TestVoiceManager_Integration_RealAPI(t *testing.T) {
	// 跳过此测试，因为需要真实的 CosyVoice API 密钥
	t.Skip("需要真实的 CosyVoice API 密钥，跳过此测试")

	// 从环境变量读取 API 密钥
	apiKey := os.Getenv("ALIYUN_COSYVOICE_API_KEY")
	if apiKey == "" {
		t.Skip("CosyVoice API 密钥未设置，跳过此测试")
	}

	// 创建 VoiceManager
	redisClient := config.NewMockRedisClient()
	cryptoManager := config.NewCryptoManager("test-encryption-key-32-bytes!!")
	configManager := config.NewConfigManager(redisClient, cryptoManager)
	voiceManager := voice_cache.NewVoiceManager(redisClient, configManager)

	// 测试音色注册
	ctx := context.Background()
	speakerID := "test-speaker-" + time.Now().Format("20060102150405")
	referenceAudio := "/path/to/reference.wav" // 需要替换为真实的音频文件路径

	voiceID, err := voiceManager.RegisterVoice(ctx, speakerID, referenceAudio, apiKey, "")
	if err != nil {
		t.Fatalf("RegisterVoice() error = %v", err)
	}

	if voiceID == "" {
		t.Errorf("RegisterVoice() returned empty voice ID")
	}

	t.Logf("音色注册成功: voice_id=%s", voiceID)

	// 测试音色状态轮询
	status, err := voiceManager.PollVoiceStatus(ctx, voiceID, apiKey, "")
	if err != nil {
		t.Fatalf("PollVoiceStatus() error = %v", err)
	}

	if status != "OK" && status != "PROCESSING" && status != "FAILED" {
		t.Errorf("PollVoiceStatus() returned unexpected status: %s", status)
	}

	t.Logf("音色状态查询成功: status=%s", status)
}

// TestConfigManager_Integration_Encryption 测试配置加密和解密（需要 Redis）
func TestConfigManager_Integration_Encryption(t *testing.T) {
	// 跳过此测试，因为需要真实的 Redis 环境
	t.Skip("需要真实的 Redis 环境，跳过此测试")

	// 创建 Redis 客户端
	redisClient := config.NewRedisClient("localhost:6379", "", 0)

	// 创建 CryptoManager
	cryptoManager := config.NewCryptoManager("test-encryption-key-32-bytes!!")

	// 加密 API 密钥
	plainAPIKey := "test-api-key-12345"
	encryptedAPIKey, err := cryptoManager.EncryptAPIKey(plainAPIKey)
	if err != nil {
		t.Fatalf("EncryptAPIKey() error = %v", err)
	}

	// 写入加密的 API 密钥到 Redis
	if err := redisClient.HSet("app:settings", "asr_api_key", encryptedAPIKey); err != nil {
		t.Fatalf("HSet() error = %v", err)
	}

	// 创建 ConfigManager
	configManager := config.NewConfigManager(redisClient, cryptoManager)

	// 读取配置（应该自动解密）
	appConfig, err := configManager.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// 验证解密后的 API 密钥
	if appConfig.ASRAPIKey != plainAPIKey {
		t.Errorf("ASRAPIKey = %v, want %v", appConfig.ASRAPIKey, plainAPIKey)
	}

	t.Logf("配置加密和解密成功")
}

