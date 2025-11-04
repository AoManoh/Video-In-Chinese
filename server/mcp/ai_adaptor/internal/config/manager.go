package config

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AppConfig 应用配置结构
type AppConfig struct {
	// ASR 配置
	ASRProvider  string
	ASRAPIKey    string // 解密后的 API 密钥
	ASREndpoint  string

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

	// 译文优化配置
	OptimizationEnabled  bool
	OptimizationProvider string
	OptimizationAPIKey   string
	OptimizationEndpoint string

	// 声音克隆配置
	VoiceCloningProvider            string
	VoiceCloningAPIKey              string
	VoiceCloningEndpoint            string
	VoiceCloningAutoSelectReference bool

	// 元数据
	LoadedAt time.Time // 配置加载时间
}

// ConfigManager 配置管理器
type ConfigManager struct {
	redisClient   *RedisClient
	cryptoManager *CryptoManager

	// 配置缓存
	cache      *AppConfig
	cacheExpiry time.Time
	cacheTTL    time.Duration
	mu          sync.RWMutex

	// 降级标志
	degraded bool
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(redisClient *RedisClient, cryptoManager *CryptoManager) *ConfigManager {
	// 从环境变量读取缓存 TTL，默认 10 分钟
	cacheTTL := 10 * time.Minute
	if ttlStr := getEnv("CONFIG_CACHE_TTL", "600"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr + "s"); err == nil {
			cacheTTL = ttl
		}
	}

	return &ConfigManager{
		redisClient:   redisClient,
		cryptoManager: cryptoManager,
		cacheTTL:      cacheTTL,
	}
}

// GetConfig 获取应用配置（带缓存）
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

// reloadConfig 从 Redis 重新加载配置
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

// parseConfig 解析 Redis 配置
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

	return config, nil
}

// validateConfig 验证配置有效性
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

// isValidProvider 验证提供商是否有效
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

// IsDegraded 返回配置管理器是否处于降级模式
func (m *ConfigManager) IsDegraded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.degraded
}

// InvalidateCache 使缓存失效，强制下次重新加载
func (m *ConfigManager) InvalidateCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheExpiry = time.Time{}
	log.Println("[ConfigManager] Cache invalidated")
}

