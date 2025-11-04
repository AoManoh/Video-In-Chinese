package voice_cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"video-in-chinese/ai_adaptor/internal/config"
	"video-in-chinese/ai_adaptor/internal/utils"
)

// VoiceInfo 音色信息结构
type VoiceInfo struct {
	VoiceID        string    // 阿里云返回的音色 ID
	CreatedAt      time.Time // 创建时间
	ReferenceAudio string    // 参考音频路径
}

// VoiceManager 音色缓存管理器
// 负责音色注册、缓存、轮询，实现 Redis + 内存二级缓存
type VoiceManager struct {
	redisClient *config.RedisClient

	// 内存缓存
	cache map[string]*VoiceInfo // key: speaker_id, value: VoiceInfo
	mu    sync.RWMutex          // 并发安全保护

	// 配置参数
	registerTimeout       time.Duration // 音色注册超时时间（默认 60 秒）
	registerRetry         int           // 音色注册失败重试次数（默认 3 次）
	registerRetryInterval time.Duration // 音色注册重试间隔（默认 5 秒）
}

// NewVoiceManager 创建新的音色缓存管理器
func NewVoiceManager(redisClient *config.RedisClient) *VoiceManager {
	// 从环境变量读取配置
	registerTimeout := 60 * time.Second
	if timeoutStr := getEnv("VOICE_REGISTER_TIMEOUT", "60"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			registerTimeout = time.Duration(timeout) * time.Second
		}
	}

	registerRetry := 3
	if retryStr := getEnv("VOICE_REGISTER_RETRY", "3"); retryStr != "" {
		if retry, err := strconv.Atoi(retryStr); err == nil {
			registerRetry = retry
		}
	}

	registerRetryInterval := 5 * time.Second
	if intervalStr := getEnv("VOICE_REGISTER_RETRY_INTERVAL", "5"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			registerRetryInterval = time.Duration(interval) * time.Second
		}
	}

	log.Printf("[VoiceManager] Initialized with timeout=%s, retry=%d, retry_interval=%s",
		registerTimeout, registerRetry, registerRetryInterval)

	return &VoiceManager{
		redisClient:           redisClient,
		cache:                 make(map[string]*VoiceInfo),
		registerTimeout:       registerTimeout,
		registerRetry:         registerRetry,
		registerRetryInterval: registerRetryInterval,
	}
}

// GetOrRegisterVoice 获取或注册音色
// 步骤 1: 检查内存缓存
// 步骤 2: 检查 Redis 缓存
// 步骤 3: 缓存未命中，注册新音色
func (vm *VoiceManager) GetOrRegisterVoice(ctx context.Context, speakerID, referenceAudio, apiKey, endpoint string) (string, error) {
	// 步骤 1: 检查内存缓存
	vm.mu.RLock()
	if voiceInfo, ok := vm.cache[speakerID]; ok {
		vm.mu.RUnlock()
		log.Printf("[VoiceManager] Voice found in memory cache: speaker_id=%s, voice_id=%s", speakerID, voiceInfo.VoiceID)
		return voiceInfo.VoiceID, nil
	}
	vm.mu.RUnlock()

	// 步骤 2: 检查 Redis 缓存
	redisCache, err := vm.redisClient.GetVoiceCache(ctx, speakerID)
	if err != nil {
		log.Printf("[VoiceManager] WARNING: Failed to read Redis cache: %v", err)
		// Redis 读取失败，继续注册新音色
	} else if len(redisCache) > 0 && redisCache["voice_id"] != "" {
		// Redis 缓存命中，构建 VoiceInfo 并存储到内存
		voiceInfo := &VoiceInfo{
			VoiceID:        redisCache["voice_id"],
			ReferenceAudio: redisCache["reference_audio"],
		}
		// 解析创建时间
		if createdAtStr := redisCache["created_at"]; createdAtStr != "" {
			if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				voiceInfo.CreatedAt = createdAt
			}
		}

		// 存储到内存缓存
		vm.mu.Lock()
		vm.cache[speakerID] = voiceInfo
		vm.mu.Unlock()

		log.Printf("[VoiceManager] Voice loaded from Redis cache: speaker_id=%s, voice_id=%s", speakerID, voiceInfo.VoiceID)
		return voiceInfo.VoiceID, nil
	}

	// 步骤 3: 缓存未命中，注册新音色
	log.Printf("[VoiceManager] Voice not found in cache, registering new voice: speaker_id=%s", speakerID)
	return vm.RegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
}

// RegisterVoice 注册新音色
// 步骤 1: 初始化重试参数
// 步骤 2: 上传参考音频到临时 OSS
// 步骤 3: 调用阿里云 API 创建音色
// 步骤 4: 轮询音色状态
// 步骤 5: 缓存音色信息到 Redis
// 步骤 6: 缓存音色信息到内存
// 步骤 7: 记录成功日志
// 步骤 8: 错误处理和重试
func (vm *VoiceManager) RegisterVoice(ctx context.Context, speakerID, referenceAudio, apiKey, endpoint string) (string, error) {
	var lastErr error

	// 步骤 1: 初始化重试参数
	for retryCount := 0; retryCount <= vm.registerRetry; retryCount++ {
		if retryCount > 0 {
			log.Printf("[VoiceManager] Retrying voice registration (attempt %d/%d): speaker_id=%s",
				retryCount, vm.registerRetry, speakerID)
			time.Sleep(vm.registerRetryInterval)
		}

		// 步骤 2: 上传参考音频到阿里云 OSS
		publicURL, err := vm.uploadToOSS(ctx, referenceAudio, apiKey)
		if err != nil {
			lastErr = fmt.Errorf("failed to upload reference audio to OSS: %w", err)
			continue
		}

		// 步骤 3: 调用阿里云 API 创建音色
		voiceID, err := vm.createVoice(ctx, publicURL, apiKey, endpoint)
		if err != nil {
			lastErr = fmt.Errorf("failed to create voice: %w", err)
			continue
		}

		// 步骤 4: 轮询音色状态
		if err := vm.PollVoiceStatus(ctx, voiceID, apiKey, endpoint); err != nil {
			lastErr = fmt.Errorf("failed to poll voice status: %w", err)
			continue
		}

		// 步骤 5: 缓存音色信息到 Redis
		createdAt := time.Now().Format(time.RFC3339)
		if err := vm.redisClient.SetVoiceCache(ctx, speakerID, voiceID, createdAt, referenceAudio); err != nil {
			log.Printf("[VoiceManager] WARNING: Failed to cache voice to Redis: %v", err)
			// Redis 缓存失败不影响注册流程，继续执行
		}

		// 步骤 6: 缓存音色信息到内存
		voiceInfo := &VoiceInfo{
			VoiceID:        voiceID,
			CreatedAt:      time.Now(),
			ReferenceAudio: referenceAudio,
		}
		vm.mu.Lock()
		vm.cache[speakerID] = voiceInfo
		vm.mu.Unlock()

		// 步骤 7: 记录成功日志
		log.Printf("[VoiceManager] Voice registered successfully: speaker_id=%s, voice_id=%s", speakerID, voiceID)
		return voiceID, nil
	}

	// 步骤 8: 所有重试失败，返回错误
	log.Printf("[VoiceManager] ERROR: Voice registration failed after %d retries: speaker_id=%s, error=%v",
		vm.registerRetry, speakerID, lastErr)
	return "", fmt.Errorf("voice registration failed after %d retries: %w", vm.registerRetry, lastErr)
}

// PollVoiceStatus 轮询音色状态
// 固定间隔 1 秒轮询，60 秒超时
// 状态检查：OK（成功）、FAILED（失败）、PROCESSING（处理中）
func (vm *VoiceManager) PollVoiceStatus(ctx context.Context, voiceID, apiKey, endpoint string) error {
	log.Printf("[VoiceManager] Polling voice status: voice_id=%s, timeout=%s", voiceID, vm.registerTimeout)

	// 创建超时上下文
	pollCtx, cancel := context.WithTimeout(ctx, vm.registerTimeout)
	defer cancel()

	// 固定间隔 1 秒轮询
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			// 超时
			log.Printf("[VoiceManager] ERROR: Voice registration timeout: voice_id=%s", voiceID)
			return fmt.Errorf("voice registration timeout after %s", vm.registerTimeout)

		case <-ticker.C:
			// 查询音色状态
			status, err := vm.getVoiceStatus(ctx, voiceID, apiKey, endpoint)
			if err != nil {
				log.Printf("[VoiceManager] WARNING: Failed to get voice status: %v", err)
				continue
			}

			log.Printf("[VoiceManager] Voice status: voice_id=%s, status=%s", voiceID, status)

			switch status {
			case "OK":
				// 音色注册成功
				log.Printf("[VoiceManager] Voice registration completed: voice_id=%s", voiceID)
				return nil

			case "FAILED":
				// 音色注册失败
				log.Printf("[VoiceManager] ERROR: Voice registration failed: voice_id=%s", voiceID)
				return fmt.Errorf("voice registration failed: status=FAILED")

			case "PROCESSING":
				// 继续轮询
				continue

			default:
				// 未知状态
				log.Printf("[VoiceManager] WARNING: Unknown voice status: voice_id=%s, status=%s", voiceID, status)
				continue
			}
		}
	}
}

// HandleVoiceNotFound 处理音色失效错误（404）
// 步骤 1: 清除内存缓存
// 步骤 2: 清除 Redis 缓存
// 步骤 3: 重新注册音色
func (vm *VoiceManager) HandleVoiceNotFound(ctx context.Context, speakerID, referenceAudio, apiKey, endpoint string) (string, error) {
	log.Printf("[VoiceManager] Voice not found (404), clearing cache and re-registering: speaker_id=%s", speakerID)

	// 步骤 1: 清除内存缓存
	vm.mu.Lock()
	delete(vm.cache, speakerID)
	vm.mu.Unlock()

	// 步骤 2: 清除 Redis 缓存
	if err := vm.redisClient.DeleteVoiceCache(ctx, speakerID); err != nil {
		log.Printf("[VoiceManager] WARNING: Failed to delete Redis cache: %v", err)
		// Redis 删除失败不影响重新注册流程，继续执行
	}

	// 步骤 3: 重新注册音色
	voiceID, err := vm.RegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to re-register voice after 404: %w", err)
	}

	log.Printf("[VoiceManager] Voice re-registered successfully: speaker_id=%s, voice_id=%s", speakerID, voiceID)
	return voiceID, nil
}

// createVoice 调用阿里云 CosyVoice API 创建音色
// 参数:
//   - ctx: 上下文
//   - publicURL: 参考音频的公开 URL（OSS URL）
//   - apiKey: API 密钥
//   - endpoint: 自定义端点（为空则使用默认端点）
//
// 返回:
//   - voiceID: 音色 ID
//   - error: 错误信息
func (vm *VoiceManager) createVoice(ctx context.Context, publicURL, apiKey, endpoint string) (string, error) {
	log.Printf("[VoiceManager] Creating voice with reference audio: url=%s", publicURL)

	// 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		apiEndpoint = "https://nls-gateway.cn-shanghai.aliyuncs.com/cosyvoice/v1/voices"
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"reference_audio_url": publicURL,
		"speaker_name":        "speaker_" + time.Now().Format("20060102150405"),
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiEndpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取音色 ID
	voiceID, ok := response["voice_id"].(string)
	if !ok {
		return "", fmt.Errorf("响应中缺少 voice_id 字段")
	}

	log.Printf("[VoiceManager] Voice created successfully: voice_id=%s", voiceID)
	return voiceID, nil
}

// getVoiceStatus 调用阿里云 API 查询音色状态
// 参数:
//   - ctx: 上下文
//   - voiceID: 音色 ID
//   - apiKey: API 密钥
//   - endpoint: 自定义端点（为空则使用默认端点）
//
// 返回:
//   - status: 音色状态（"OK", "FAILED", "PROCESSING"）
//   - error: 错误信息
func (vm *VoiceManager) getVoiceStatus(ctx context.Context, voiceID, apiKey, endpoint string) (string, error) {
	log.Printf("[VoiceManager] Querying voice status: voice_id=%s", voiceID)

	// 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		apiEndpoint = "https://nls-gateway.cn-shanghai.aliyuncs.com/cosyvoice/v1/voices/" + voiceID
	} else {
		apiEndpoint = apiEndpoint + "/" + voiceID
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取状态
	status, ok := response["status"].(string)
	if !ok {
		return "", fmt.Errorf("响应中缺少 status 字段")
	}

	log.Printf("[VoiceManager] Voice status: voice_id=%s, status=%s", voiceID, status)
	return status, nil
}

// uploadToOSS 上传参考音频到阿里云 OSS
// 参数:
//   - ctx: 上下文
//   - referenceAudio: 参考音频的本地路径
//   - apiKey: API 密钥（用于 OSS 认证）
//
// 返回:
//   - publicURL: 上传后的公开 URL
//   - error: 错误信息
func (vm *VoiceManager) uploadToOSS(ctx context.Context, referenceAudio, apiKey string) (string, error) {
	log.Printf("[VoiceManager] Uploading reference audio to OSS: path=%s", referenceAudio)

	// 从环境变量读取 OSS 配置
	accessKeyID := os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("ALIYUN_OSS_BUCKET_NAME")
	endpoint := os.Getenv("ALIYUN_OSS_ENDPOINT")

	// 验证配置
	if accessKeyID == "" || accessKeySecret == "" || bucketName == "" || endpoint == "" {
		log.Printf("[VoiceManager] WARNING: OSS 配置不完整，使用模拟 URL 作为降级方案")
		return "https://example.oss.aliyuncs.com/temp/" + filepath.Base(referenceAudio), nil
	}

	// 创建 OSS 上传器
	uploader, err := utils.NewOSSUploader(accessKeyID, accessKeySecret, endpoint, bucketName)
	if err != nil {
		log.Printf("[VoiceManager] WARNING: 创建 OSS 上传器失败: %v，使用模拟 URL", err)
		return "https://example.oss.aliyuncs.com/temp/" + filepath.Base(referenceAudio), nil
	}

	// 生成对象键
	objectKey := utils.GenerateObjectKey(referenceAudio, "voice-reference")

	// 上传文件
	publicURL, err := uploader.UploadFile(ctx, referenceAudio, objectKey)
	if err != nil {
		log.Printf("[VoiceManager] WARNING: 上传文件到 OSS 失败: %v，使用模拟 URL", err)
		return "https://example.oss.aliyuncs.com/temp/" + filepath.Base(referenceAudio), nil
	}

	log.Printf("[VoiceManager] OSS upload completed: %s", publicURL)
	return publicURL, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
