package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisClient(t *testing.T) (*RedisClient, func()) {
	t.Helper()
	ctx := context.Background()

	redisContainer, err := testredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client, err := NewRedisClient(RedisConfig{Host: endpoint, Type: "node"})
	require.NoError(t, err)

	cleanup := func() {
		_ = client.Close()
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	}

	return client, cleanup
}

func TestRedisClient_PushTaskSingle(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, client.PushTask(ctx, "task-1"))

	length, err := client.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, length)
}

func TestRedisClient_PushTaskMultiple(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, client.PushTask(ctx, "task-"+string(rune('0'+i))))
	}

	length, err := client.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, length)
}

func TestRedisClient_SetAndGetTaskFields(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	fields := map[string]interface{}{
		"task_id":            "task-123",
		"status":             "PENDING",
		"original_file_path": "/path/original.mp4",
		"created_at":         time.Now().Format(time.RFC3339),
	}

	require.NoError(t, client.SetTaskFields(ctx, "task-123", fields))

	stored, err := client.GetTaskFields(ctx, "task-123")
	require.NoError(t, err)
	assert.Equal(t, "PENDING", stored["status"])
	assert.Equal(t, "/path/original.mp4", stored["original_file_path"])
}

func TestRedisClient_SetTaskFieldsPartialUpdate(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	ctx := context.Background()

	require.NoError(t, client.SetTaskFields(ctx, "task-456", map[string]interface{}{
		"task_id": "task-456",
		"status":  "PENDING",
	}))

	require.NoError(t, client.SetTaskFields(ctx, "task-456", map[string]interface{}{
		"status":           "COMPLETED",
		"result_file_path": "/result.mp4",
	}))

	stored, err := client.GetTaskFields(ctx, "task-456")
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", stored["status"])
	assert.Equal(t, "/result.mp4", stored["result_file_path"])
}

func TestRedisClient_GetTaskFieldsNotFound(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	_, err := client.GetTaskFields(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestRedisClient_TaskExists(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, client.SetTaskFields(ctx, "task-789", map[string]interface{}{"status": "PENDING"}))

	exists, err := client.TaskExists(ctx, "task-789")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = client.TaskExists(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisClient_SetTaskField(t *testing.T) {
	client, cleanup := setupRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, client.SetTaskField(ctx, "task-900", "status", "PROCESSING"))

	stored, err := client.GetTaskFields(ctx, "task-900")
	require.NoError(t, err)
	assert.Equal(t, "PROCESSING", stored["status"])
}
