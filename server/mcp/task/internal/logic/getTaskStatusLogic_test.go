package logic

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"video-in-chinese/server/mcp/task/internal/storage"
	"video-in-chinese/server/mcp/task/internal/svc"
	"video-in-chinese/server/mcp/task/proto"
)

func setupTaskForStatusTests(t *testing.T) (*proto.CreateTaskResponse, *svc.ServiceContext, func()) {
	svcCtx, cleanup := SetupTestRedis(t)
	tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)
	createLogic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := createLogic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})
	require.NoError(t, err)
	return resp, svcCtx, cleanup
}

func TestGetTaskStatusLogic_Success_Pending(t *testing.T) {
	defer cleanupHooks(t)()
	resp, svcCtx, cleanup := setupTaskForStatusTests(t)
	defer cleanup()

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	statusResp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: resp.TaskId})

	require.NoError(t, err)
	require.NotNil(t, statusResp)
	assert.Equal(t, proto.TaskStatus_PENDING, statusResp.Status)
	assert.Empty(t, statusResp.ResultFilePath)
	assert.Empty(t, statusResp.ErrorMessage)
	_, err = time.Parse(time.RFC3339, statusResp.CreatedAt)
	assert.NoError(t, err)
	_, err = time.Parse(time.RFC3339, statusResp.UpdatedAt)
	assert.NoError(t, err)
}

func TestGetTaskStatusLogic_Success_Processing(t *testing.T) {
	defer cleanupHooks(t)()
	resp, svcCtx, cleanup := setupTaskForStatusTests(t)
	defer cleanup()

	now := time.Now().Format(time.RFC3339)
	require.NoError(t, svcCtx.RedisClient.SetTaskFields(context.Background(), resp.TaskId, map[string]interface{}{
		"status":     "PROCESSING",
		"updated_at": now,
	}))

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	statusResp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: resp.TaskId})

	require.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_PROCESSING, statusResp.Status)
	assert.Empty(t, statusResp.ResultFilePath)
	assert.Empty(t, statusResp.ErrorMessage)
	assert.Equal(t, now, statusResp.UpdatedAt)
}

func TestGetTaskStatusLogic_Success_Completed(t *testing.T) {
	defer cleanupHooks(t)()
	resp, svcCtx, cleanup := setupTaskForStatusTests(t)
	defer cleanup()

	now := time.Now().Format(time.RFC3339)
	resultPath := "/storage/videos/" + resp.TaskId + "/result.mp4"
	require.NoError(t, svcCtx.RedisClient.SetTaskFields(context.Background(), resp.TaskId, map[string]interface{}{
		"status":           "COMPLETED",
		"result_file_path": resultPath,
		"updated_at":       now,
	}))

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	statusResp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: resp.TaskId})

	require.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_COMPLETED, statusResp.Status)
	assert.Equal(t, resultPath, statusResp.ResultFilePath)
	assert.Empty(t, statusResp.ErrorMessage)
	assert.Equal(t, now, statusResp.UpdatedAt)
}

func TestGetTaskStatusLogic_Success_Failed(t *testing.T) {
	defer cleanupHooks(t)()
	resp, svcCtx, cleanup := setupTaskForStatusTests(t)
	defer cleanup()

	now := time.Now().Format(time.RFC3339)
	message := "Failed to process video"
	require.NoError(t, svcCtx.RedisClient.SetTaskFields(context.Background(), resp.TaskId, map[string]interface{}{
		"status":        "FAILED",
		"error_message": message,
		"updated_at":    now,
	}))

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	statusResp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: resp.TaskId})

	require.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_FAILED, statusResp.Status)
	assert.Equal(t, message, statusResp.ErrorMessage)
	assert.Empty(t, statusResp.ResultFilePath)
}

func TestGetTaskStatusLogic_TaskNotFound(t *testing.T) {
	defer cleanupHooks(t)()
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	resp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: "non-existent"})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "task not found")
}

func TestGetTaskStatusLogic_RedisGetFailed(t *testing.T) {
	defer cleanupHooks(t)()
	storage.SetGetTaskFieldsHook(func(ctx context.Context, taskID string) error {
		return assert.AnError
	})

	resp, svcCtx, cleanup := setupTaskForStatusTests(t)
	defer cleanup()

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	statusResp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: resp.TaskId})

	require.Error(t, err)
	assert.Nil(t, statusResp)
	assert.Contains(t, err.Error(), "task not found")
}

func TestGetTaskStatusLogic_InvalidStatus(t *testing.T) {
	defer cleanupHooks(t)()
	resp, svcCtx, cleanup := setupTaskForStatusTests(t)
	defer cleanup()

	require.NoError(t, svcCtx.RedisClient.SetTaskFields(context.Background(), resp.TaskId, map[string]interface{}{
		"status": "UNKNOWN_STATE",
	}))

	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	statusResp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{TaskId: resp.TaskId})

	require.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_PENDING, statusResp.Status)
}
