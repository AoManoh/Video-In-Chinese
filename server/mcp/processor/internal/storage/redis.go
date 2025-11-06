package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RedisClient encapsulates Redis operations for the Processor service.
//
// This client provides methods for task queue operations and task status management.
type RedisClient struct {
	client *redis.Redis
}

// NewRedisClient creates a new RedisClient instance.
//
// Parameters:
//   - client: go-zero Redis client
//
// Returns:
//   - *RedisClient: initialized Redis client
func NewRedisClient(client *redis.Redis) *RedisClient {
	return &RedisClient{
		client: client,
	}
}

// PopTask pops a JSON task message string from the pending queue (LPOP).
//
// Parameters:
//   - ctx: context for cancellation
//   - queueKey: Redis queue key (e.g., "task:pending")
//
// Returns:
//   - string: raw JSON of TaskMessage {task_id, original_file_path}; empty if queue is empty
//   - error: error if Redis operation fails
func (r *RedisClient) PopTask(ctx context.Context, queueKey string) (string, error) {
	taskID, err := r.client.Lpop(queueKey)
	if err != nil {
		if err == redis.Nil {
			// Queue is empty, not an error
			return "", nil
		}
		logx.Errorf("[RedisClient] Failed to pop task from queue %s: %v", queueKey, err)
		return "", fmt.Errorf("failed to pop task: %w", err)
	}

	return taskID, nil
}

// GetTaskFields retrieves all fields of a task from Redis Hash (HGETALL).
//
// Parameters:
//   - ctx: context for cancellation
//   - taskID: task ID
//
// Returns:
//   - fields: map of field names to values
//   - error: error if task not found or Redis operation fails
func (r *RedisClient) GetTaskFields(ctx context.Context, taskID string) (map[string]string, error) {
	key := fmt.Sprintf("task:%s", taskID)

	fields, err := r.client.Hgetall(key)
	if err != nil {
		logx.Errorf("[RedisClient] Failed to get task fields for %s: %v", taskID, err)
		return nil, fmt.Errorf("failed to get task fields: %w", err)
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return fields, nil
}

// SetTaskFields sets multiple fields of a task in Redis Hash (HMSET).
//
// Parameters:
//   - ctx: context for cancellation
//   - taskID: task ID
//   - fields: map of field names to values
//
// Returns:
//   - error: error if Redis operation fails
func (r *RedisClient) SetTaskFields(ctx context.Context, taskID string, fields map[string]interface{}) error {
	key := fmt.Sprintf("task:%s", taskID)

	// Convert map[string]interface{} to map[string]string
	stringFields := make(map[string]string, len(fields))
	for k, v := range fields {
		stringFields[k] = fmt.Sprintf("%v", v)
	}

	err := r.client.HmsetCtx(ctx, key, stringFields)
	if err != nil {
		logx.Errorf("[RedisClient] Failed to set task fields for %s: %v", taskID, err)
		return fmt.Errorf("failed to set task fields: %w", err)
	}

	return nil
}

// UpdateTaskStatus updates the status field of a task.
//
// Parameters:
//   - ctx: context for cancellation
//   - taskID: task ID
//   - status: new status (PENDING, PROCESSING, COMPLETED, FAILED)
//
// Returns:
//   - error: error if Redis operation fails
func (r *RedisClient) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	key := fmt.Sprintf("task:%s", taskID)

	fields := map[string]string{
		"status":     status,
		"updated_at": time.Now().Format(time.RFC3339),
	}
	if err := r.client.HmsetCtx(ctx, key, fields); err != nil {
		logx.Errorf("[RedisClient] Failed to update task status for %s: %v", taskID, err)
		return fmt.Errorf("failed to update task status: %w", err)
	}

	logx.Infof("[RedisClient] Updated task %s status to %s", taskID, status)
	return nil
}

// UpdateTaskError updates the error field of a task.
//
// Parameters:
//   - ctx: context for cancellation
//   - taskID: task ID
//   - errorMsg: error message
//
// Returns:
//   - error: error if Redis operation fails
func (r *RedisClient) UpdateTaskError(ctx context.Context, taskID, errorMsg string) error {
	key := fmt.Sprintf("task:%s", taskID)

	fields := map[string]string{
		"error_message": errorMsg,
		"updated_at":    time.Now().Format(time.RFC3339),
	}
	if err := r.client.HmsetCtx(ctx, key, fields); err != nil {
		logx.Errorf("[RedisClient] Failed to update task error for %s: %v", taskID, err)
		return fmt.Errorf("failed to update task error: %w", err)
	}

	logx.Infof("[RedisClient] Updated task %s error: %s", taskID, errorMsg)
	return nil
}

// GetAppSettings retrieves application settings from Redis Hash.
//
// Workflow:
//  1. HGETALL app:settings
//  2. Return all settings as map
//
// Parameters:
//   - ctx: context for cancellation
//
// Returns:
//   - map[string]string: all application settings
//   - error: if Redis operation fails
func (r *RedisClient) GetAppSettings(ctx context.Context) (map[string]string, error) {
	key := "app:settings"

	settings, err := r.client.Hgetall(key)
	if err != nil {
		logx.Errorf("[RedisClient] Failed to get app settings: %v", err)
		return nil, fmt.Errorf("failed to get app settings: %w", err)
	}

	if len(settings) == 0 {
		logx.Infof("[RedisClient] App settings not found, using defaults")
		return make(map[string]string), nil
	}

	return settings, nil
}
