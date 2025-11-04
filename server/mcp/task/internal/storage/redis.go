package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient 封装 Redis 客户端，提供任务队列和状态存储操作
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient 创建 Redis 客户端实例
// 从环境变量读取配置：
//   - REDIS_ADDR: Redis 地址（默认 localhost:6379）
//   - REDIS_PASSWORD: Redis 密码（默认为空）
//   - REDIS_DB: Redis 数据库编号（默认 0）
func NewRedisClient() (*RedisClient, error) {
	// 读取环境变量
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	dbStr := os.Getenv("REDIS_DB")
	db := 0
	if dbStr != "" {
		var err error
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB value: %v", err)
		}
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		MaxRetries:   3,              // 最大重试次数
		PoolSize:     10,             // 连接池大小
		MinIdleConns: 2,              // 最小空闲连接数
		DialTimeout:  5 * time.Second,  // 连接超时
		ReadTimeout:  3 * time.Second,  // 读超时
		WriteTimeout: 3 * time.Second,  // 写超时
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Printf("✓ Redis client connected to %s (DB: %d)", addr, db)

	return &RedisClient{client: client}, nil
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping 测试 Redis 连接
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// PushTask 推入任务到队列
// 使用 LPUSH 将任务ID推入队列头部（Key: task:pending）
func (r *RedisClient) PushTask(ctx context.Context, taskID string) error {
	key := "task:pending"
	if err := r.client.LPush(ctx, key, taskID).Err(); err != nil {
		return fmt.Errorf("failed to push task to queue: %v", err)
	}
	log.Printf("[Redis] Task pushed to queue: %s", taskID)
	return nil
}

// GetQueueLength 获取队列长度
// 使用 LLEN 查询队列长度（Key: task:pending）
func (r *RedisClient) GetQueueLength(ctx context.Context) (int64, error) {
	key := "task:pending"
	length, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %v", err)
	}
	return length, nil
}

// SetTaskField 设置任务字段
// 使用 HSET 设置任务的单个字段（Key: task:{task_id}）
func (r *RedisClient) SetTaskField(ctx context.Context, taskID, field, value string) error {
	key := fmt.Sprintf("task:%s", taskID)
	if err := r.client.HSet(ctx, key, field, value).Err(); err != nil {
		return fmt.Errorf("failed to set task field: %v", err)
	}
	return nil
}

// SetTaskFields 批量设置任务字段
// 使用 HSET 批量设置任务的多个字段（Key: task:{task_id}）
func (r *RedisClient) SetTaskFields(ctx context.Context, taskID string, fields map[string]interface{}) error {
	key := fmt.Sprintf("task:%s", taskID)
	if err := r.client.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("failed to set task fields: %v", err)
	}
	log.Printf("[Redis] Task fields set: %s (fields: %d)", taskID, len(fields))
	return nil
}

// GetTaskFields 获取任务所有字段
// 使用 HGETALL 读取任务的所有字段（Key: task:{task_id}）
func (r *RedisClient) GetTaskFields(ctx context.Context, taskID string) (map[string]string, error) {
	key := fmt.Sprintf("task:%s", taskID)
	fields, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task fields: %v", err)
	}

	// 检查任务是否存在
	if len(fields) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return fields, nil
}

// TaskExists 检查任务是否存在
// 使用 EXISTS 检查任务是否存在（Key: task:{task_id}）
func (r *RedisClient) TaskExists(ctx context.Context, taskID string) (bool, error) {
	key := fmt.Sprintf("task:%s", taskID)
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check task existence: %v", err)
	}
	return exists > 0, nil
}

