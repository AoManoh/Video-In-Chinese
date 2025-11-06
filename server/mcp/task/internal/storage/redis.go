// Package storage provides Redis and file storage abstractions for the Task service.
//
// This package implements two key storage components:
//   - RedisClient: Manages task queue (List) and task state (Hash) in Redis
//   - FileStorage: Manages video file operations with cross-filesystem fallback
//
// Design Decisions:
//   - Redis is used for both queue (task:pending) and state storage (task:{task_id})
//   - Queue uses LPUSH/RPOP pattern for FIFO processing by Processor service
//   - State uses Hash structure for efficient field-level updates
//
// Integration with go-zero:
//   - Uses go-zero's redis.Redis client (built-in connection pooling and retry logic)
//   - Uses logx for logging (go-zero standard logging library)
//   - Configuration passed from ServiceContext (go-zero dependency injection)
package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RedisClient encapsulates Redis operations for task queue and state management.
//
// Data Structures:
//   - Queue: List (Key: "task:pending", Operations: LPUSH/RPOP)
//   - State: Hash (Key: "task:{task_id}", Fields: status, original_file_path, etc.)
//
// Thread Safety:
//   - The underlying go-zero redis.Redis client is thread-safe and uses connection pooling
//   - Multiple goroutines can safely call methods on the same RedisClient instance
//
// Integration with go-zero:
//   - Uses go-zero's redis.Redis client (automatic connection pooling, retry, and circuit breaker)
//   - Configuration managed by go-zero config system (config.Redis)
type RedisClient struct {
	client *redis.Redis
}

// 以下钩子用于测试场景中注入故障或自定义行为。
var (
	setTaskFieldsHook func(ctx context.Context, taskID string, fields map[string]interface{}) error
	pushTaskHook      func(ctx context.Context, taskID, originalFilePath string) error
	getTaskFieldsHook func(ctx context.Context, taskID string) error
)

// SetSetTaskFieldsHook 设置测试钩子，在实际执行 Redis HSET 之前触发。
func SetSetTaskFieldsHook(hook func(ctx context.Context, taskID string, fields map[string]interface{}) error) {
	setTaskFieldsHook = hook
}

// SetPushTaskHook 设置测试钩子，在实际执行队列写入之前触发。
func SetPushTaskHook(hook func(ctx context.Context, taskID, originalFilePath string) error) {
	pushTaskHook = hook
}

// SetGetTaskFieldsHook 设置测试钩子，在执行 HGETALL 之前触发。
func SetGetTaskFieldsHook(hook func(ctx context.Context, taskID string) error) {
	getTaskFieldsHook = hook
}

// ResetTestHooks 重置所有测试钩子，避免测试之间互相影响。
func ResetTestHooks() {
	setTaskFieldsHook = nil
	pushTaskHook = nil
	getTaskFieldsHook = nil
}

// RedisConfig defines the Redis configuration structure.
type RedisConfig struct {
	Host string
	Type string
	Pass string
}

// NewRedisClient creates a new Redis client instance using go-zero's redis.Redis.
//
// Configuration is passed from go-zero config (config.Redis):
//   - Host: Redis server address (e.g., "localhost:6379")
//   - Type: Redis type ("node" for standalone, "cluster" for cluster)
//   - Pass: Redis password (optional)
//
// go-zero redis.Redis Features:
//   - Automatic connection pooling
//   - Automatic retry on transient failures
//   - Circuit breaker for fault tolerance
//   - Prometheus metrics integration
//
// Parameters:
//   - redisConfig: Redis configuration (from config.Redis)
//
// Returns an error if connection to Redis fails.
func NewRedisClient(redisConfig RedisConfig) (*RedisClient, error) {
	// 转换为 go-zero RedisConf
	redisConf := redis.RedisConf{
		Host: redisConfig.Host,
		Type: redisConfig.Type,
		Pass: redisConfig.Pass,
	}

	// 创建 go-zero Redis 客户端
	client := redis.MustNewRedis(redisConf)

	// 测试连接
	ok := client.Ping()
	if !ok {
		return nil, fmt.Errorf("failed to connect to Redis")
	}

	logx.Infof("Redis client connected to %s", redisConf.Host)

	return &RedisClient{client: client}, nil
}

// NewRedisClientWithOptions 允许在跳过 Ping 校验的情况下创建 Redis 客户端（主要用于测试注入故障场景）。
func NewRedisClientWithOptions(redisConf redis.RedisConf, skipPing bool) (*RedisClient, error) {
	client := redis.MustNewRedis(redisConf)
	if !skipPing {
		ok := client.Ping()
		if !ok {
			return nil, fmt.Errorf("failed to connect to Redis")
		}
	}
	return &RedisClient{client: client}, nil
}

// TaskQueueMessage represents the message structure pushed to the task queue.
// This must match the structure expected by Processor service.
type TaskQueueMessage struct {
	TaskID           string `json:"task_id"`
	OriginalFilePath string `json:"original_file_path"`
}

// PushTask pushes a task message to the pending task queue in JSON format.
//
// Queue Design:
//   - Key: "task:pending"
//   - Operation: RPUSH (append to tail for FIFO semantics)
//   - Message Format: JSON with task_id and original_file_path
//   - Consumer: Processor service pops from the head to maintain order
//
// Why LPUSH?
//   - LPUSH + RPOP provides FIFO (First-In-First-Out) semantics
//   - Tasks are processed in the order they are created
//   - Processor service can use blocking BRPOP for efficient polling
//
// Why JSON Format?
//   - Processor needs both task_id and original_file_path to start processing
//   - JSON provides structured data that's easy to parse and extend
//   - Matches the TaskMessage structure in Processor service
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - taskID: UUID v4 string identifying the task
//   - originalFilePath: Path to the original video file
//
// Returns an error if the Redis operation fails.
func (r *RedisClient) PushTask(ctx context.Context, taskID, originalFilePath string) error {
	if pushTaskHook != nil {
		if err := pushTaskHook(ctx, taskID, originalFilePath); err != nil {
			return err
		}
	}

	// Create task queue message
	message := TaskQueueMessage{
		TaskID:           taskID,
		OriginalFilePath: originalFilePath,
	}

	// Marshal to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal task message: %v", err)
	}

	// Push to queue
	_, err = r.client.RpushCtx(ctx, "task:pending", string(messageJSON))
	if err != nil {
		return fmt.Errorf("failed to push task to queue: %v", err)
	}
	logx.WithContext(ctx).Infof("[RedisClient] Task pushed to queue: %s (path: %s)", taskID, originalFilePath)
	return nil
}

// SetTaskFields sets multiple fields in a task's Redis Hash.
//
// State Design:
//   - Key: "task:{task_id}"
//   - Structure: Hash (field-value pairs)
//   - Fields: status, original_file_path, result_file_path, error_message, created_at, updated_at
//
// Why Hash?
//   - Efficient field-level updates (no need to read-modify-write entire object)
//   - Atomic operations (HSET is atomic)
//   - Easy to query individual fields (HGET) or all fields (HGETALL)
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - taskID: UUID v4 string identifying the task
//   - fields: Map of field names to values (e.g., {"status": "PENDING", "created_at": "2024-01-01T00:00:00Z"})
//
// Returns an error if the Redis operation fails.
func (r *RedisClient) SetTaskFields(ctx context.Context, taskID string, fields map[string]interface{}) error {
	key := fmt.Sprintf("task:%s", taskID)

	if setTaskFieldsHook != nil {
		if err := setTaskFieldsHook(ctx, taskID, fields); err != nil {
			return err
		}
	}

	// 转换 map[string]interface{} 为 map[string]string
	stringFields := make(map[string]string, len(fields))
	for k, v := range fields {
		stringFields[k] = fmt.Sprintf("%v", v)
	}

	// go-zero redis.Redis 使用 Hmset 方法
	err := r.client.HmsetCtx(ctx, key, stringFields)
	if err != nil {
		return fmt.Errorf("failed to set task fields: %v", err)
	}

	logx.WithContext(ctx).Infof("[RedisClient] Task fields set: %s (fields: %d)", taskID, len(fields))
	return nil
}

// GetTaskFields retrieves all fields from a task's Redis Hash.
//
// This method uses HGETALL to retrieve all field-value pairs from the task's Hash.
// If the task does not exist, returns an error.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - taskID: UUID v4 string identifying the task
//
// Returns:
//   - Map of field names to values (e.g., {"status": "PENDING", "created_at": "2024-01-01T00:00:00Z"})
//   - Error if the task does not exist or Redis operation fails
func (r *RedisClient) GetTaskFields(ctx context.Context, taskID string) (map[string]string, error) {
	key := fmt.Sprintf("task:%s", taskID)

	if getTaskFieldsHook != nil {
		if err := getTaskFieldsHook(ctx, taskID); err != nil {
			return nil, err
		}
	}

	// go-zero redis.Redis 使用 Hgetall 方法
	fields, err := r.client.HgetallCtx(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task fields: %v", err)
	}

	// Check if task exists (empty Hash means task not found)
	if len(fields) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	logx.WithContext(ctx).Infof("[RedisClient] Task fields retrieved: %s (fields: %d)", taskID, len(fields))
	return fields, nil
}

// SetTaskField sets a single field in a task's Redis Hash.
//
// This is a convenience method for updating a single field without reading
// the entire Hash. It uses HSET which is atomic.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - taskID: UUID v4 string identifying the task
//   - field: Field name (e.g., "status", "error_message")
//   - value: Field value (e.g., "COMPLETED", "File not found")
//
// Returns an error if the Redis operation fails.
func (r *RedisClient) SetTaskField(ctx context.Context, taskID, field, value string) error {
	key := fmt.Sprintf("task:%s", taskID)

	// go-zero redis.Redis 使用 Hset 方法
	err := r.client.HsetCtx(ctx, key, field, value)
	if err != nil {
		return fmt.Errorf("failed to set task field: %v", err)
	}

	logx.WithContext(ctx).Infof("[RedisClient] Task field set: %s.%s = %s", taskID, field, value)
	return nil
}

// TaskExists checks if a task exists in Redis.
//
// This method uses EXISTS to check if the task's Hash key exists.
// It is more efficient than HGETALL when you only need to check existence.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - taskID: UUID v4 string identifying the task
//
// Returns:
//   - true if the task exists
//   - false if the task does not exist
//   - Error if the Redis operation fails
func (r *RedisClient) TaskExists(ctx context.Context, taskID string) (bool, error) {
	key := fmt.Sprintf("task:%s", taskID)

	// go-zero redis.Redis 使用 Exists 方法
	exists, err := r.client.ExistsCtx(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check task existence: %v", err)
	}

	return exists, nil
}

// GetQueueLength returns the number of tasks in the pending queue.
//
// This method uses LLEN to get the length of the task queue.
// It is useful for monitoring and debugging.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//
// Returns:
//   - Number of tasks in the queue
//   - Error if the Redis operation fails
func (r *RedisClient) GetQueueLength(ctx context.Context) (int, error) {
	// go-zero redis.Redis 使用 Llen 方法
	length, err := r.client.LlenCtx(ctx, "task:pending")
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %v", err)
	}

	return length, nil
}

// Close closes the Redis connection and releases all resources.
//
// This method should be called when the service is shutting down.
// It is safe to call Close multiple times.
//
// Note: go-zero's redis.Redis does not expose a Close method,
// so this is a no-op for compatibility with the old interface.
func (r *RedisClient) Close() error {
	// go-zero redis.Redis 自动管理连接池，无需手动关闭
	logx.Info("[RedisClient] Redis client closed (no-op for go-zero redis.Redis)")
	return nil
}
