package config

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

// AppConfig 描述了 AIAdaptor 的运行时配置快照，集中保存 ASR、翻译、LLM、语音克隆等供应商参数。
//
// 功能说明:
//   - 作为 ConfigManager 的返回结果，为业务层提供解密后的配置数据。
//   - 支撑 VoiceManager 与各适配器读取统一的 API 密钥与端点。
//
// 设计决策:
//   - 字段按功能模块分组，避免嵌套过深便于序列化与维护。
//   - 记录 LoadedAt 以帮助判定缓存是否过期并支持降级策略。
//
// 使用示例:
//
//	cfg, err := manager.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(cfg.ASRProvider, cfg.TranslationProvider)
//
// 参数说明:
//   - 不适用: 结构体由 ConfigManager 内部构造与填充。
//
// 返回值说明:
//   - 不适用: 结构体实例作为配置载体在各层之间传递。
//
// 错误处理说明:
//   - ConfigManager 在解析失败时返回 error，本结构体定义本身不产生错误。
//
// 注意事项:
//   - 新增字段时需同步更新 Redis 序列化格式与 parseConfig 逻辑以保持兼容性。
type AppConfig struct {
	// ASR 配置
	ASRProvider     string
	ASRAPIKey       string // 解密后的 API 密钥
	ASREndpoint     string
	ASRLanguageCode string // 语言代码（如 "zh-CN", "en-US"）
	ASRRegion       string // 区域信息（Azure ASR 需要）

	// 翻译配置
	TranslationProvider  string
	TranslationAPIKey    string
	TranslationEndpoint  string
	TranslationVideoType string

	// 文本润色配置
	PolishingEnabled      bool
	PolishingProvider     string
	PolishingAPIKey       string
	PolishingEndpoint     string
	PolishingVideoType    string
	PolishingCustomPrompt string
	PolishingModelName    string // LLM 模型名称（如 "gpt-4o", "gemini-1.5-flash"）

	// 译文优化配置
	OptimizationEnabled   bool
	OptimizationProvider  string
	OptimizationAPIKey    string
	OptimizationEndpoint  string
	OptimizationModelName string // LLM 模型名称

	// 声音克隆配置
	VoiceCloningProvider            string
	VoiceCloningAPIKey              string
	VoiceCloningEndpoint            string
	VoiceCloningAutoSelectReference bool
	VoiceCloningOutputDir           string // 音频输出目录

	// 阿里云 OSS 配置（用于文件上传）
	AliyunOSSAccessKeyID     string
	AliyunOSSAccessKeySecret string
	AliyunOSSBucketName      string
	AliyunOSSEndpoint        string
	AliyunOSSRegion          string

	// 元数据
	LoadedAt time.Time // 配置加载时间
}

// ConfigManager 管理配置拉取、解密与缓存策略，为各适配器持续提供一致的运行时配置。
//
// 功能说明:
//   - 从 Redis 读取应用配置并通过 CryptoManager 解密敏感字段。
//   - 在 Redis 不可用时提供缓存降级数据，保障服务可用性。
//
// 设计决策:
//   - 使用读写锁保护缓存，避免重复 reload 导致的外部压力。
//   - 通过 degraded 标志对外暴露降级状态，方便监控埋点。
//
// 使用示例:
//
//	manager := NewConfigManager(redisClient, cryptoManager)
//	cfg, err := manager.GetConfig(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(manager.IsDegraded(), cfg.LoadedAt)
//
// 参数说明:
//   - redisClient *RedisClient: 封装 Redis 操作的客户端。
//   - cryptoManager *CryptoManager: 负责 API 密钥的解密。
//
// 返回值说明:
//   - 不适用: 结构体通过指针在内部传递。
//
// 错误处理说明:
//   - GetConfig/ReloadConfig 返回 error 时调用方应决定是否阻断请求或使用降级配置。
//
// 注意事项:
//   - 服务启动或配置热更新后应调用 InvalidateCache 以确保下次访问从 Redis 重新加载。
type ConfigManager struct {
	redisClient   *RedisClient
	cryptoManager *CryptoManager

	// 配置缓存
	cache       *AppConfig
	cacheExpiry time.Time
	cacheTTL    time.Duration
	mu          sync.RWMutex

	// 降级标志
	degraded bool
}

// NewConfigManager 构造 ConfigManager 并初始化缓存 TTL、解密器等依赖。
//
// 功能说明:
//   - 读取 CONFIG_CACHE_TTL 环境变量决定缓存过期时间，默认 10 分钟。
//   - 组合 Redis 客户端与加解密器，为后续加载流程做准备。
//
// 设计决策:
//   - 将 TTL 解析放在构造函数，避免每次 GetConfig 都重复读取环境变量。
//   - 使用依赖注入方便在测试中提供模拟实现。
//
// 使用示例:
//
//	manager := NewConfigManager(redisClient, cryptoManager)
//	defer manager.InvalidateCache()
//
// 参数说明:
//
//	redisClient *RedisClient: 负责访问 Redis 的客户端。
//	cryptoManager *CryptoManager: 处理加密配置的解密器。
//
// 返回值说明:
//
//	*ConfigManager: 已初始化完成的配置管理器实例。
//
// 错误处理说明:
//   - 函数自身不返回错误，环境变量解析失败将回退为默认 TTL 并记录日志。
//
// 注意事项:
//   - 需确保传入的 redisClient 和 cryptoManager 在应用生命周期内有效且线程安全。
func NewConfigManager(redisClient *RedisClient, cryptoManager *CryptoManager) *ConfigManager {
	// 从环境变量读取缓存 TTL，默认 10 分钟
	cacheTTL := 10 * time.Minute
	if ttlStr := getEnv("CONFIG_CACHE_TTL", "600"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = ttl
		} else if seconds, err := strconv.Atoi(ttlStr); err == nil {
			cacheTTL = time.Duration(seconds) * time.Second
		} else {
			log.Printf("[ConfigManager] WARNING: invalid CONFIG_CACHE_TTL value %q, using default %s", ttlStr, cacheTTL)
		}
	}

	return &ConfigManager{
		redisClient:   redisClient,
		cryptoManager: cryptoManager,
		cacheTTL:      cacheTTL,
	}
}

// GetConfig 获取应用配置并应用缓存策略，优先返回有效的内存快照。
//
// 功能说明:
//   - 首先读取内存缓存，若未过期直接返回，减少对 Redis 的访问压力。
//   - 当缓存失效时触发 reloadConfig，从 Redis 加载并解密配置。
//
// 设计决策:
//   - 使用读写锁保障并发安全，避免重复刷新配置。
//   - 通过日志记录缓存命中与降级信息，便于运维排查。
//
// 使用示例:
//
//	cfg, err := manager.GetConfig(ctx)
//	if err != nil {
//	    return nil, fmt.Errorf("load config failed: %w", err)
//	}
//	return cfg, nil
//
// 参数说明:
//
//	ctx context.Context: 控制 Redis 请求超时与取消信号。
//
// 返回值说明:
//
//	*AppConfig: 当前有效的配置快照，字段已解密。
//	error: 当缓存为空且 Redis 读取/解析失败时返回。
//
// 错误处理说明:
//   - 如果 Redis 失败但存在旧缓存，会返回缓存并标记降级；若无缓存则返回错误。
//
// 注意事项:
//   - 调用方应在请求链路上设置合理的上下文超时，防止阻塞。
func (m *ConfigManager) GetConfig(ctx context.Context) (*AppConfig, error) {
	m.mu.RLock()
	// 检查缓存是否有效
	if m.cache != nil && time.Now().Before(m.cacheExpiry) {
		defer m.mu.RUnlock()
		log.Println("[ConfigManager] Using cached config")
		return m.cache, nil
	}
	m.mu.RUnlock()

	// 缓存失效，重新加载
	return m.reloadConfig(ctx)
}

// reloadConfig 从 Redis 重新加载配置并刷新内存缓存，是缓存失效时的核心路径。
//
// 功能说明:
//   - 再次检查缓存是否已被其它协程刷新，避免重复加载。
//   - 调用 Redis 获取配置，解析并解密后写入缓存，同时更新降级标志。
//
// 设计决策:
//   - 使用写锁保证同一时间只有一个协程执行加载，避免竞态。
//   - 降级策略优先返回旧缓存，确保服务不中断。
//
// 使用示例:
//
//	cfg, err := m.reloadConfig(ctx)
//	if err != nil {
//	    return nil, err
//	}
//	return cfg, nil
//
// 参数说明:
//
//	ctx context.Context: 控制 Redis 调用的超时与取消。
//
// 返回值说明:
//
//	*AppConfig: 最新的配置快照。
//	error: Redis 读取或配置解析失败且无缓存可用时返回。
//
// 错误处理说明:
//   - 当 Redis 读取失败但存在缓存时，记录警告并返回旧缓存。
//   - 当配置解析失败时同样保留旧缓存并开启降级模式。
//
// 注意事项:
//   - 调用方应仅在缓存失效时进入该函数，避免高频访问导致 Redis 压力。
func (m *ConfigManager) reloadConfig(ctx context.Context) (*AppConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查：可能其他 goroutine 已经加载了
	if m.cache != nil && time.Now().Before(m.cacheExpiry) {
		log.Println("[ConfigManager] Config already reloaded by another goroutine")
		return m.cache, nil
	}

	log.Println("[ConfigManager] Reloading config from Redis...")

	// 从 Redis 读取配置
	settings, err := m.redisClient.GetAppSettings(ctx)
	if err != nil {
		// Redis 读取失败，使用降级策略
		if m.cache != nil {
			log.Printf("[ConfigManager] WARNING: Failed to reload config from Redis: %v, using cached config (degraded mode)", err)
			m.degraded = true
			// 延长缓存有效期（降级模式下使用旧配置）
			m.cacheExpiry = time.Now().Add(m.cacheTTL)
			return m.cache, nil
		}
		return nil, fmt.Errorf("failed to load config from Redis and no cache available: %w", err)
	}

	// 解析配置
	config, err := m.parseConfig(settings)
	if err != nil {
		// 解析失败，使用降级策略
		if m.cache != nil {
			log.Printf("[ConfigManager] WARNING: Failed to parse config: %v, using cached config (degraded mode)", err)
			m.degraded = true
			m.cacheExpiry = time.Now().Add(m.cacheTTL)
			return m.cache, nil
		}
		return nil, fmt.Errorf("failed to parse config and no cache available: %w", err)
	}

	// 验证配置
	if err := m.validateConfig(config); err != nil {
		// 验证失败，使用降级策略
		if m.cache != nil {
			log.Printf("[ConfigManager] WARNING: Config validation failed: %v, using cached config (degraded mode)", err)
			m.degraded = true
			m.cacheExpiry = time.Now().Add(m.cacheTTL)
			return m.cache, nil
		}
		return nil, fmt.Errorf("config validation failed and no cache available: %w", err)
	}

	// 更新缓存
	config.LoadedAt = time.Now()
	m.cache = config
	m.cacheExpiry = time.Now().Add(m.cacheTTL)
	m.degraded = false

	log.Printf("[ConfigManager] Config reloaded successfully, cache expires at %s", m.cacheExpiry.Format(time.RFC3339))
	return m.cache, nil
}

// parseConfig 解析 Redis 返回的应用配置并执行字段解密与默认值填充。
//
// 功能说明:
//   - 将 Redis 字符串键值对转换为 AppConfig 结构体。
//   - 对敏感字段调用 CryptoManager 解密，补充默认参数。
//
// 设计决策:
//   - 使用显式字段赋值而非反射，以确保类型安全并便于调试。
//   - 按模块划分解析逻辑，方便未来扩展。
//
// 使用示例:
//
//	cfg, err := m.parseConfig(settings)
//	if err != nil {
//	    return nil, err
//	}
//	return cfg, nil
//
// 参数说明:
//
//	settings map[string]string: 从 Redis 获取的配置字典。
//
// 返回值说明:
//
//	*AppConfig: 解析后的配置对象。
//	error: 当必填字段缺失或解密失败时返回。
//
// 错误处理说明:
//   - 对每个加密字段分别捕获错误并包装上下文信息。
//   - 当必需字段缺失时返回带字段名的错误，便于排查。
//
// 注意事项:
//   - 新增配置项时需同步更新此函数与 validateConfig，以保持一致性。
func (m *ConfigManager) parseConfig(settings map[string]string) (*AppConfig, error) {
	config := &AppConfig{}

	// 解析 ASR 配置
	config.ASRProvider = settings["asr_provider"]
	if encryptedKey := settings["asr_api_key"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt asr_api_key: %w", err)
		}
		config.ASRAPIKey = decryptedKey
	}
	config.ASREndpoint = settings["asr_endpoint"]
	config.ASRLanguageCode = settings["asr_language_code"]
	config.ASRRegion = settings["asr_region"]

	// 解析翻译配置
	config.TranslationProvider = settings["translation_provider"]
	if encryptedKey := settings["translation_api_key"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt translation_api_key: %w", err)
		}
		config.TranslationAPIKey = decryptedKey
	}
	config.TranslationEndpoint = settings["translation_endpoint"]
	config.TranslationVideoType = settings["translation_video_type"]

	// 解析文本润色配置
	config.PolishingEnabled = settings["polishing_enabled"] == "true"
	config.PolishingProvider = settings["polishing_provider"]
	if encryptedKey := settings["polishing_api_key"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt polishing_api_key: %w", err)
		}
		config.PolishingAPIKey = decryptedKey
	}
	config.PolishingEndpoint = settings["polishing_endpoint"]
	config.PolishingVideoType = settings["polishing_video_type"]
	config.PolishingCustomPrompt = settings["polishing_custom_prompt"]
	config.PolishingModelName = settings["polishing_model_name"]

	// 解析译文优化配置
	config.OptimizationEnabled = settings["optimization_enabled"] == "true"
	config.OptimizationProvider = settings["optimization_provider"]
	if encryptedKey := settings["optimization_api_key"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt optimization_api_key: %w", err)
		}
		config.OptimizationAPIKey = decryptedKey
	}
	config.OptimizationEndpoint = settings["optimization_endpoint"]
	config.OptimizationModelName = settings["optimization_model_name"]

	// 解析声音克隆配置
	config.VoiceCloningProvider = settings["voice_cloning_provider"]
	if encryptedKey := settings["voice_cloning_api_key"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt voice_cloning_api_key: %w", err)
		}
		config.VoiceCloningAPIKey = decryptedKey
	}
	config.VoiceCloningEndpoint = settings["voice_cloning_endpoint"]
	config.VoiceCloningAutoSelectReference = settings["voice_cloning_auto_select_reference"] == "true"
	config.VoiceCloningOutputDir = settings["voice_cloning_output_dir"]

	// 解析阿里云 OSS 配置
	if encryptedKey := settings["aliyun_oss_access_key_id"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt aliyun_oss_access_key_id: %w", err)
		}
		config.AliyunOSSAccessKeyID = decryptedKey
	}
	if encryptedKey := settings["aliyun_oss_access_key_secret"]; encryptedKey != "" {
		decryptedKey, err := m.cryptoManager.DecryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt aliyun_oss_access_key_secret: %w", err)
		}
		config.AliyunOSSAccessKeySecret = decryptedKey
	}
	config.AliyunOSSBucketName = settings["aliyun_oss_bucket_name"]
	config.AliyunOSSEndpoint = settings["aliyun_oss_endpoint"]
	config.AliyunOSSRegion = settings["aliyun_oss_region"]

	return config, nil
}

// validateConfig 验证解析后的配置是否合法，确保必填字段齐全且供应商受支持。
//
// 功能说明:
//   - 校验每个模块的必填字段是否存在并符合约束。
//   - 组合 isValidProvider 限定可使用的供应商列表。
//
// 设计决策:
//   - 逐一返回具体的字段错误，方便调用方快速定位。
//
// 使用示例:
//
//	if err := m.validateConfig(config); err != nil {
//	    return nil, err
//	}
//
// 参数说明:
//
//	config *AppConfig: 待验证的配置对象。
//
// 返回值说明:
//
//	error: 当配置缺失或非法时返回详细错误，为 nil 表示验证通过。
//
// 错误处理说明:
//   - 发现错误立即返回，由上层决定是否走降级路径。
//
// 注意事项:
//   - 新增供应商或可选配置时需同步维护验证逻辑和白名单。
func (m *ConfigManager) validateConfig(config *AppConfig) error {
	// 验证 ASR 配置
	if config.ASRProvider == "" {
		return fmt.Errorf("asr_provider is required")
	}
	if !isValidProvider("asr", config.ASRProvider) {
		return fmt.Errorf("invalid asr_provider: %s", config.ASRProvider)
	}
	if config.ASRAPIKey == "" {
		return fmt.Errorf("asr_api_key is required")
	}

	// 验证翻译配置
	if config.TranslationProvider == "" {
		return fmt.Errorf("translation_provider is required")
	}
	if !isValidProvider("translation", config.TranslationProvider) {
		return fmt.Errorf("invalid translation_provider: %s", config.TranslationProvider)
	}
	if config.TranslationAPIKey == "" {
		return fmt.Errorf("translation_api_key is required")
	}

	// 验证声音克隆配置
	if config.VoiceCloningProvider == "" {
		return fmt.Errorf("voice_cloning_provider is required")
	}
	if !isValidProvider("voice_cloning", config.VoiceCloningProvider) {
		return fmt.Errorf("invalid voice_cloning_provider: %s", config.VoiceCloningProvider)
	}
	if config.VoiceCloningAPIKey == "" {
		return fmt.Errorf("voice_cloning_api_key is required")
	}

	// 验证可选的文本润色配置
	if config.PolishingEnabled {
		if config.PolishingProvider == "" {
			return fmt.Errorf("polishing_provider is required when polishing_enabled is true")
		}
		if !isValidProvider("llm", config.PolishingProvider) {
			return fmt.Errorf("invalid polishing_provider: %s", config.PolishingProvider)
		}
		if config.PolishingAPIKey == "" {
			return fmt.Errorf("polishing_api_key is required when polishing_enabled is true")
		}
	}

	// 验证可选的译文优化配置
	if config.OptimizationEnabled {
		if config.OptimizationProvider == "" {
			return fmt.Errorf("optimization_provider is required when optimization_enabled is true")
		}
		if !isValidProvider("llm", config.OptimizationProvider) {
			return fmt.Errorf("invalid optimization_provider: %s", config.OptimizationProvider)
		}
		if config.OptimizationAPIKey == "" {
			return fmt.Errorf("optimization_api_key is required when optimization_enabled is true")
		}
	}

	return nil
}

// isValidProvider 验证供应商是否处于白名单，防止配置注入未知厂商。
//
// 功能说明:
//   - 根据服务类型检索允许的供应商列表，并判断是否包含目标值。
//
// 设计决策:
//   - 使用常量 map 存储白名单，保持 O(1) 查询复杂度且便于维护。
//
// 使用示例:
//
//	if !isValidProvider("asr", provider) {
//	    return fmt.Errorf("invalid provider")
//	}
//
// 参数说明:
//
//	serviceType string: 服务类别标识，如 "asr"、"translation"。
//	provider string: 待校验的供应商名称。
//
// 返回值说明:
//
//	bool: true 表示支持该供应商，false 表示不支持。
//
// 错误处理说明:
//   - 函数本身不返回错误，调用方需根据返回值创建上下文错误。
//
// 注意事项:
//   - 添加新供应商时需同步扩展白名单配置。
func isValidProvider(serviceType, provider string) bool {
	validProviders := map[string][]string{
		"asr":           {"aliyun", "azure", "google"},
		"translation":   {"deepl", "google", "azure"},
		"llm":           {"openai-gpt4o", "claude", "gemini"},
		"voice_cloning": {"aliyun_cosyvoice"},
	}

	providers, ok := validProviders[serviceType]
	if !ok {
		return false
	}

	for _, p := range providers {
		if p == provider {
			return true
		}
	}
	return false
}

// IsDegraded 返回配置管理器当前是否处于降级模式，用于暴露运维指标。
//
// 功能说明:
//   - 在 Redis 读取或解析失败且使用缓存时标记降级状态。
//
// 设计决策:
//   - 提供只读方法以便调用方快速检查并上报监控。
//
// 使用示例:
//
//	if manager.IsDegraded() {
//	    metrics.Inc("config.degraded")
//	}
//
// 参数说明:
//   - 无参数。
//
// 返回值说明:
//
//	bool: true 表示当前使用缓存降级数据，false 表示正常模式。
//
// 错误处理说明:
//   - 方法不返回错误，状态由 reloadConfig 更新。
//
// 注意事项:
//   - 调用应在读取配置后执行，以获取最新状态。
func (m *ConfigManager) IsDegraded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.degraded
}

// InvalidateCache 主动使配置缓存失效，促使下一次读取从 Redis 重新加载最新数据。
//
// 功能说明:
//   - 清除缓存过期时间并输出日志，常用在配置热更新或调试场景。
//
// 设计决策:
//   - 仅重置过期时间而不清空缓存指针，避免并发读出现 nil。
//
// 使用示例:
//
//	manager.InvalidateCache()
//	cfg, _ := manager.GetConfig(ctx)
//
// 参数说明:
//   - 无参数。
//
// 返回值说明:
//   - 无返回值。
//
// 错误处理说明:
//   - 函数不返回错误，如需确认可结合日志与 IsDegraded 状态。
//
// 注意事项:
//   - 调用后下一次 GetConfig 会阻塞至 Redis 读取完成，需评估对延迟的影响。
func (m *ConfigManager) InvalidateCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheExpiry = time.Time{}
	log.Println("[ConfigManager] Cache invalidated")
}
