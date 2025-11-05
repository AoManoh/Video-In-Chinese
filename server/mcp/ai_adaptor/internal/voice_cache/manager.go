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

	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
	"video-in-chinese/server/mcp/ai_adaptor/internal/utils"
)

// VoiceInfo 表示已注册的语音资源元数据，用于在内存与 Redis 间同步缓存状态。
//
// 功能说明:
//   - 持久化语音 ID、参考音频及创建时间，供 VoiceManager 命中缓存时快速返回。
//
// 设计决策:
//   - 使用简单结构体避免额外依赖序列化库，便于在 Redis 中以 Hash 格式存储。
//
// 使用示例:
//
//	info := &VoiceInfo{VoiceID: "voice-1", ReferenceAudio: ref}
//
// 参数说明:
//   - 不适用: VoiceInfo 由 VoiceManager 创建。
//
// 返回值说明:
//   - 不适用: 结构体作为缓存值。
//
// 错误处理说明:
//   - 不涉及。
//
// 注意事项:
//   - CreatedAt 用于判断缓存是否过期，写入 Redis 时需保持 RFC3339 格式。
type VoiceInfo struct {
	VoiceID        string    // 阿里云返回的音色 ID
	CreatedAt      time.Time // 创建时间
	ReferenceAudio string    // 参考音频路径
}

// VoiceManager 负责管理语音克隆缓存，协调 Redis 与内存二级缓存以减少重复注册。
//
// 功能说明:
//   - 提供语音注册、缓存命中、状态轮询等能力。
//   - 封装与阿里云 CosyVoice API 的交互，包括上传、查询和降级策略。
//
// 设计决策:
//   - 使用内存 map + Redis 组合缓存，兼顾本地性能与服务重启后的持久性。
//   - 通过配置化的重试与超时参数适配不同部署环境。
//
// 使用示例:
//
//	manager := NewVoiceManager(redisClient)
//	voiceID, err := manager.GetOrRegisterVoice(ctx, speakerID, refAudio, apiKey, endpoint)
//
// 参数说明:
//   - redisClient *config.RedisClient: 用于读写语音缓存的客户端。
//
// 返回值说明:
//   - 不适用: 结构体实例供业务层调用。
//
// 错误处理说明:
//   - 具体方法返回错误，本结构体初始化阶段不产生错误。
//
// 注意事项:
//   - 在高并发场景下需配合外部锁避免同一 speakerID 重复注册。
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

// NewVoiceManager 创建 VoiceManager 并根据环境变量初始化注册超时、重试等策略。
//
// 功能说明:
//   - 读取 VOICE_REGISTER_* 环境变量设置注册流程参数。
//   - 构造内存缓存结构并记录初始化日志，便于运维确认。
//
// 设计决策:
//   - 将参数解析集中在构造函数中，避免每次注册时重复读取环境变量。
//   - 默认值兼顾线上性能与可扩展性，可通过环境变量调整。
//
// 使用示例:
//
//	manager := NewVoiceManager(redisClient)
//
// 参数说明:
//
//	redisClient *config.RedisClient: 用于访问 Redis 缓存的客户端。
//
// 返回值说明:
//
//	*VoiceManager: 初始化完成的缓存管理器。
//
// 错误处理说明:
//   - 函数不返回错误，环境变量解析失败会回退默认值并记录日志。
//
// 注意事项:
//   - 需确保 redisClient 长期有效且支持并发安全调用。
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

// GetOrRegisterVoice 根据 speakerID 命中缓存或执行注册流程，返回语音 ID。
//
// 功能说明:
//   - 按顺序检查内存缓存、Redis 缓存，未命中时触发 RegisterVoice。
//   - 将命中的缓存写回内存，以缩短后续调用延迟。
//
// 设计决策:
//   - 采用读写锁保护内存缓存，避免并发读写冲突。
//   - 缓存命中后提前返回，减少外部 API 调用次数。
//
// 使用示例:
//
//	voiceID, err := vm.GetOrRegisterVoice(ctx, "speaker-1", refAudio, apiKey, endpoint)
//
// 参数说明:
//
//	ctx context.Context: 控制整个流程的超时与取消。
//	speakerID string: 语音缓存键，通常对应用户或角色。
//	referenceAudio string: 参考音频路径，用于缺失时注册。
//	apiKey string: 供应商认证密钥。
//	endpoint string: 可选自定义 API 地址。
//
// 返回值说明:
//
//	string: 命中或新注册的语音 ID。
//	error: 当注册失败且无法回退缓存时返回。
//
// 错误处理说明:
//   - 内存或 Redis 命中均不会返回错误。
//   - 注册失败会记录警告并返回错误，由调用方决定是否重试或降级。
//
// 注意事项:
//   - 并发场景下建议在调用外层使用散列锁，避免重复注册同一 speakerID。
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

// RegisterVoice 注册新的语音资源并返回语音 ID，内部包含上传、轮询与缓存写入流程。
//
// 功能说明:
//   - 上传参考音频到 OSS，调用供应商 API 发起语音克隆任务。
//   - 轮询任务状态，成功后写入 Redis 与内存缓存。
//
// 设计决策:
//   - 通过配置化重试与超时参数提升稳定性。
//   - 失败时保留最后一次错误供调用方决策。
//
// 使用示例:
//
//	voiceID, err := vm.RegisterVoice(ctx, speakerID, refAudio, apiKey, endpoint)
//
// 参数说明:
//
//	ctx context.Context: 控制注册流程的生命周期。
//	speakerID string: 语音缓存键。
//	referenceAudio string: 参考音频路径。
//	apiKey string: 供应商认证密钥。
//	endpoint string: 可选自定义 API 地址。
//
// 返回值说明:
//
//	string: 成功创建的语音 ID。
//	error: 注册失败时返回最后一次错误。
//
// 错误处理说明:
//   - 失败会根据配置重试多次，所有尝试失败后返回错误。
//   - OSS 上传、API 调用与缓存写入任一环节失败均记录日志。
//
// 注意事项:
//   - 上层应结合 registerRetry 配置合理设置幂等保护，避免长时间阻塞。
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

// PollVoiceStatus 轮询语音克隆任务状态，直到成功、失败或超时。
//
// 功能说明:
//   - 定期向供应商查询语音状态，直至返回 OK 或达到超时时间。
//
// 设计决策:
//   - 使用固定间隔与总超时配置，兼顾稳定性与资源消耗。
//
// 使用示例:
//
//	if err := vm.PollVoiceStatus(ctx, voiceID, apiKey, endpoint); err != nil {
//	    return err
//	}
//
// 参数说明:
//
//	ctx context.Context: 控制轮询周期与取消信号。
//	voiceID string: 待查询的语音 ID。
//	apiKey string: 供应商认证密钥。
//	endpoint string: 可选自定义 API 地址。
//
// 返回值说明:
//
//	error: 当状态为失败或超时时返回详细错误，为 nil 表示成功。
//
// 错误处理说明:
//   - 状态为 FAILED 时返回供应商错误信息。
//   - 超时时返回超时错误，供调用方决定是否重试。
//
// 注意事项:
//   - endpoint 为空时使用默认 URL，调用前需确保网络可达。
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

// HandleVoiceNotFound 处理供应商返回的 404 错误，通过清理缓存后重新注册音色。
//
// 功能说明:
//   - 清除内存与 Redis 缓存，确保后续注册使用最新数据。
//   - 触发 RegisterVoice 再次注册语音。
//
// 设计决策:
//   - 将清理与重新注册封装为一个方法，方便调用方直接处理。
//
// 使用示例:
//
//	voiceID, err := vm.HandleVoiceNotFound(ctx, speakerID, refAudio, apiKey, endpoint)
//
// 参数说明:
//
//	ctx context.Context: 控制重注册流程生命周期。
//	speakerID string: 语音缓存键。
//	referenceAudio string: 参考音频路径。
//	apiKey string: 供应商认证密钥。
//	endpoint string: 可选自定义 API 地址。
//
// 返回值说明:
//
//	string: 重新注册后的语音 ID。
//	error: 清理或注册失败时返回错误。
//
// 错误处理说明:
//   - Redis 清理失败会记录警告但继续执行注册。
//   - 注册失败直接返回错误供调用方处理。
//
// 注意事项:
//   - 该方法可能较耗时，应在调用方设置超时或异步处理。
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

// createVoice 调用供应商语音克隆创建接口，使用上传后的参考音频生成语音。
//
// 功能说明:
//   - 构造 HTTP 请求提交至语音克隆 API，并解析返回的 voice_id。
//
// 设计决策:
//   - 将供应商交互提取成独立函数，便于单元测试和重试策略的复用。
//
// 使用示例:
//
//	voiceID, err := vm.createVoice(ctx, publicURL, apiKey, endpoint)
//
// 参数说明:
//
//	ctx context.Context: 控制 HTTP 调用超时与取消。
//	publicURL string: 参考音频的公开 URL（通常来自 OSS 上传结果）。
//	apiKey string: 供应商认证密钥。
//	endpoint string: 自定义 API 端点，空字符串时使用默认值。
//
// 返回值说明:
//
//	string: 成功创建的语音 ID。
//	error: 当 HTTP 请求或响应解析失败时返回。
//
// 错误处理说明:
//   - 非 200/201 状态码映射为包含响应体的错误，便于排查。
//   - JSON 解析失败会附带响应原文，帮助分析接口变更。
//
// 注意事项:
//   - 需确保 publicURL 在供应商侧可访问，否则会导致创建失败。
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

// getVoiceStatus 查询供应商侧语音克隆任务状态，返回状态字符串。
//
// 功能说明:
//   - 向供应商发起 GET 请求，解析返回的状态字段。
//
// 设计决策:
//   - 将状态查询封装成独立函数，供 PollVoiceStatus 循环调用。
//
// 使用示例:
//
//	status, err := vm.getVoiceStatus(ctx, voiceID, apiKey, endpoint)
//
// 参数说明:
//
//	ctx context.Context: 控制请求超时与取消。
//	voiceID string: 待查询的语音 ID。
//	apiKey string: 供应商认证密钥。
//	endpoint string: 自定义端点，空字符串时使用默认值。
//
// 返回值说明:
//
//	string: 语音状态，常见值包括 "OK"、"FAILED"、"PROCESSING"。
//	error: HTTP 请求失败或响应解析失败时返回。
//
// 错误处理说明:
//   - 非 200 状态码会返回包含响应体的错误，帮助定位问题。
//
// 注意事项:
//   - endpoint 应包含完整路径，函数内部只拼接 voiceID。
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

// uploadToOSS 将参考音频上传到阿里云 OSS，并返回可公开访问的 URL。
//
// 功能说明:
//   - 根据环境变量读取 OSS 配置，构造上传客户端并执行文件上传。
//
// 设计决策:
//   - 失败时返回模拟 URL，保证在无配置的环境下仍可进行降级演练。
//
// 使用示例:
//
//	url, err := vm.uploadToOSS(ctx, referenceAudio, apiKey)
//
// 参数说明:
//
//	ctx context.Context: 控制上传流程的生命周期。
//	referenceAudio string: 本地音频路径。
//	apiKey string: 用于 OSS 认证的密钥。
//
// 返回值说明:
//
//	string: 上传后的公开 URL，可能为模拟地址。
//	error: 当 OSS 客户端构建或上传失败且无法降级时返回。
//
// 错误处理说明:
//   - OSS 配置缺失或上传失败会记录警告并返回模拟 URL。
//
// 注意事项:
//   - 生产环境应确保环境变量齐全，以使用真实的 OSS 存储。
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

// getEnv 获取环境变量的包装函数，不存在时返回默认值。
//
// 功能说明:
//   - 统一处理缺失环境变量时的回退逻辑。
//
// 设计决策:
//   - 采用简单封装避免在各处重复编写判空逻辑。
//
// 使用示例:
//
//	timeout := getEnv("VOICE_REGISTER_TIMEOUT", "60")
//
// 参数说明:
//
//	key string: 环境变量名称。
//	defaultValue string: 变量缺失时返回的默认值。
//
// 返回值说明:
//
//	string: 实际的环境变量值或默认值。
//
// 错误处理说明:
//   - 函数不返回错误，由调用方负责验证返回值格式。
//
// 注意事项:
//   - 若 defaultValue 为空且环境变量缺失，函数将返回空字符串。
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
