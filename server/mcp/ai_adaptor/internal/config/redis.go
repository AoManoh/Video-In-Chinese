package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient 对 go-redis 客户端的轻量封装，集中处理连接配置、超时与常用 Hash 操作，
// 让上层逻辑只关注业务键值与数据结构。
type RedisClient struct {
	client     *redis.Client
	useMock    bool
	mockMu     sync.RWMutex
	mockHashes map[string]map[string]string
}

// NewRedisClient 根据环境变量创建 Redis 客户端，并在启动阶段验证连接可用性。
//
// 使用的环境变量包括:
//   - REDIS_HOST (默认: redis)
//   - REDIS_PORT (默认: 6379)
//   - REDIS_PASSWORD (默认: 空)
//   - REDIS_DB (默认: 0)
//
// 设计说明:
//   - 配置统一采用 5s 建连超时、3s 读写超时，兼顾冷启动与链路重试。
//   - 创建完成后立即调用 PING，确保部署时即可发现网络或认证问题。
func NewRedisClient() (*RedisClient, error) {
	// 从环境变量读取 Redis 配置
	host := getEnv("REDIS_HOST", "redis")
	port := getEnv("REDIS_PORT", "6379")
	password := getEnv("REDIS_PASSWORD", "")
	dbStr := getEnv("REDIS_DB", "0")

	// 转换数据库编号
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %s", dbStr)
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// NewMockRedisClient 创建使用内存数据结构的 RedisClient，适合在测试场景中替代真实 Redis。
func NewMockRedisClient() *RedisClient {
	return &RedisClient{
		useMock:    true,
		mockHashes: make(map[string]map[string]string),
	}
}

// GetAppSettings 读取 Hash 键 app:settings 中的应用配置。
//
// 返回:
//   - map[string]string: 全量配置键值对。
//   - error: 读取失败或键不存在时返回。
//
// 设计说明:
//   - 若 Hash 为空，视为配置缺失并返回错误，以提醒部署先补充配置。
func (r *RedisClient) GetAppSettings(ctx context.Context) (map[string]string, error) {
	if r.useMock {
		r.mockMu.RLock()
		defer r.mockMu.RUnlock()

		hash, ok := r.mockHashes["app:settings"]
		if !ok || len(hash) == 0 {
			return nil, fmt.Errorf("app:settings not found in Redis")
		}
		return cloneStringMap(hash), nil
	}

	result, err := r.client.HGetAll(ctx, "app:settings").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read app:settings from Redis: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("app:settings not found in Redis")
	}

	return result, nil
}

// GetVoiceCache 读取 voice_cache:{speakerID} Hash 中的音色缓存；若不存在返回空 map。
//
// 参数:
//   - speakerID: 说话人 ID。
//
// 返回:
//   - map[string]string: 包含 voice_id、created_at、reference_audio 等字段。
//   - error: Redis 访问错误时返回。
func (r *RedisClient) GetVoiceCache(ctx context.Context, speakerID string) (map[string]string, error) {
	key := fmt.Sprintf("voice_cache:%s", speakerID)
	if r.useMock {
		r.mockMu.RLock()
		defer r.mockMu.RUnlock()
		if hash, ok := r.mockHashes[key]; ok {
			return cloneStringMap(hash), nil
		}
		return make(map[string]string), nil
	}

	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read voice cache from Redis: %w", err)
	}

	// 如果缓存不存在，返回空 map（不是错误）
	return result, nil
}

// SetVoiceCache 将音色信息写入 voice_cache:{speakerID}，并根据配置设置 TTL。
//
// 参数:
//   - speakerID: 说话人 ID。
//   - voiceID: 语音克隆服务返回的音色 ID。
//   - createdAt: 创建时间（ISO 8601 格式）。
//   - referenceAudio: 参考音频路径。
//
// 设计说明:
//   - 使用 HSet 一次写入多个字段，避免网络往返。
//   - VOICE_CACHE_TTL>0 时自动设置过期时间，以防缓存长期占用。
func (r *RedisClient) SetVoiceCache(ctx context.Context, speakerID, voiceID, createdAt, referenceAudio string) error {
	key := fmt.Sprintf("voice_cache:%s", speakerID)

	if r.useMock {
		r.mockMu.Lock()
		defer r.mockMu.Unlock()
		if r.mockHashes == nil {
			r.mockHashes = make(map[string]map[string]string)
		}
		hash, ok := r.mockHashes[key]
		if !ok {
			hash = make(map[string]string)
			r.mockHashes[key] = hash
		}
		hash["voice_id"] = voiceID
		hash["created_at"] = createdAt
		hash["reference_audio"] = referenceAudio
		return nil
	}

	// 使用 HSet 设置多个字段
	err := r.client.HSet(ctx, key, map[string]interface{}{
		"voice_id":        voiceID,
		"created_at":      createdAt,
		"reference_audio": referenceAudio,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to set voice cache to Redis: %w", err)
	}

	// 设置 TTL（如果配置了）
	ttlStr := getEnv("VOICE_CACHE_TTL", "0")
	ttl, err := strconv.Atoi(ttlStr)
	if err == nil && ttl > 0 {
		r.client.Expire(ctx, key, time.Duration(ttl)*time.Second)
	}

	return nil
}

// DeleteVoiceCache 删除指定说话人的音色缓存。
//
// 参数:
//   - speakerID: 说话人 ID。
func (r *RedisClient) DeleteVoiceCache(ctx context.Context, speakerID string) error {
	key := fmt.Sprintf("voice_cache:%s", speakerID)
	if r.useMock {
		r.mockMu.Lock()
		defer r.mockMu.Unlock()
		delete(r.mockHashes, key)
		return nil
	}
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete voice cache from Redis: %w", err)
	}
	return nil
}

// Close 关闭底层 Redis 连接池。
func (r *RedisClient) Close() error {
	if r.useMock {
		return nil
	}
	return r.client.Close()
}

// HSetField 在 Hash 键中设置单个字段，便于测试场景初始化 Redis 数据。
func (r *RedisClient) HSetField(ctx context.Context, key, field string, value interface{}) error {
	if r.useMock {
		r.mockMu.Lock()
		defer r.mockMu.Unlock()
		if r.mockHashes == nil {
			r.mockHashes = make(map[string]map[string]string)
		}
		hash, ok := r.mockHashes[key]
		if !ok {
			hash = make(map[string]string)
			r.mockHashes[key] = hash
		}
		hash[field] = fmt.Sprint(value)
		return nil
	}
	return r.client.HSet(ctx, key, field, value).Err()
}

// getEnv 获取环境变量，如果不存在则返回默认值。
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func cloneStringMap(src map[string]string) map[string]string {
	dup := make(map[string]string, len(src))
	for k, v := range src {
		dup[k] = v
	}
	return dup
}
