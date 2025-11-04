package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis 客户端封装
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient 创建新的 Redis 客户端
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

// GetAppSettings 从 Redis 读取应用配置
// 返回: map[string]string 配置键值对
func (r *RedisClient) GetAppSettings(ctx context.Context) (map[string]string, error) {
	result, err := r.client.HGetAll(ctx, "app:settings").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read app:settings from Redis: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("app:settings not found in Redis")
	}

	return result, nil
}

// GetVoiceCache 从 Redis 读取音色缓存
// 参数: speakerID - 说话人 ID
// 返回: map[string]string 音色信息（voice_id, created_at, reference_audio）
func (r *RedisClient) GetVoiceCache(ctx context.Context, speakerID string) (map[string]string, error) {
	key := fmt.Sprintf("voice_cache:%s", speakerID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read voice cache from Redis: %w", err)
	}

	// 如果缓存不存在，返回空 map（不是错误）
	return result, nil
}

// SetVoiceCache 设置音色缓存到 Redis
// 参数:
//   - speakerID: 说话人 ID
//   - voiceID: 阿里云返回的音色 ID
//   - createdAt: 创建时间（ISO 8601 格式）
//   - referenceAudio: 参考音频路径
func (r *RedisClient) SetVoiceCache(ctx context.Context, speakerID, voiceID, createdAt, referenceAudio string) error {
	key := fmt.Sprintf("voice_cache:%s", speakerID)
	
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

// DeleteVoiceCache 删除音色缓存
// 参数: speakerID - 说话人 ID
func (r *RedisClient) DeleteVoiceCache(ctx context.Context, speakerID string) error {
	key := fmt.Sprintf("voice_cache:%s", speakerID)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete voice cache from Redis: %w", err)
	}
	return nil
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

