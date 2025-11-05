package logic

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"video-in-chinese/server/mcp/task-gozero/internal/storage"
	"video-in-chinese/server/mcp/task-gozero/proto"
)

// cleanupHooks 在测试前后重置 Redis 钩子，防止交叉污染。
func cleanupHooks(t *testing.T) func() {
	t.Helper()
	storage.ResetTestHooks()
	return func() {
		storage.ResetTestHooks()
	}
}

func TestCreateTaskLogic_Success(t *testing.T) {
	defer cleanupHooks(t)()
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)

	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)

	originalFilePath := svcCtx.FileStorage.GetOriginalFilePath(resp.TaskId)
	_, err = os.Stat(originalFilePath)
	assert.NoError(t, err)

	fields, err := svcCtx.RedisClient.GetTaskFields(context.Background(), resp.TaskId)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", fields["status"])
	assert.Equal(t, resp.TaskId, fields["task_id"])

	queueLength, err := svcCtx.RedisClient.GetQueueLength(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, queueLength)
}

func TestCreateTaskLogic_InvalidTempFilePath(t *testing.T) {
	defer cleanupHooks(t)()
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	invalidPath := filepath.Join(svcCtx.Config.LocalStoragePath, "non-existent.mp4")

	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: invalidPath})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to move file")
}

func TestCreateTaskLogic_FileMoveFailed(t *testing.T) {
	defer cleanupHooks(t)()
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)
	f, err := os.OpenFile(tempFile, os.O_RDWR, 0)
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(tempFile)
	}()

	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to move file")
}

func TestCreateTaskLogic_RedisSetFailed(t *testing.T) {
	defer cleanupHooks(t)()
	storage.SetSetTaskFieldsHook(func(ctx context.Context, taskID string, fields map[string]interface{}) error {
		return assert.AnError
	})

	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)

	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create task record")
}

func TestCreateTaskLogic_RedisPushFailed(t *testing.T) {
	defer cleanupHooks(t)()
	storage.SetPushTaskHook(func(ctx context.Context, taskID string) error {
		return assert.AnError
	})

	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)

	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to push task to queue")
}

func TestCreateTaskLogic_DirectoryCreationFailed(t *testing.T) {
	defer cleanupHooks(t)()
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	// 将存储基础路径替换为文件，触发目录创建失败。
	taskDir := svcCtx.FileStorage.GetTaskDir("dummy")
	baseDir := filepath.Dir(filepath.Dir(taskDir))

	require.NoError(t, os.RemoveAll(baseDir))
	require.NoError(t, os.WriteFile(baseDir, []byte("locked"), 0o644))
	defer func() { _ = os.Remove(baseDir) }()

	tempFile := createTempFile(t, t.TempDir())

	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create task directory")
}

func TestCreateTaskLogic_ConcurrentCalls(t *testing.T) {
	defer cleanupHooks(t)()
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()

	const concurrency = 5
	var wg sync.WaitGroup
	wg.Add(concurrency)

	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)
			logic := NewCreateTaskLogic(context.Background(), svcCtx)
			_, err := logic.CreateTask(&proto.CreateTaskRequest{TempFilePath: tempFile})
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	for err := range results {
		assert.NoError(t, err)
	}

	if runtime.GOOS == "windows" {
		// Windows 可能存在文件句柄未及时释放导致的残留文件，等待短暂时间
		time.Sleep(50 * time.Millisecond)
	}

	queueLength, err := svcCtx.RedisClient.GetQueueLength(context.Background())
	require.NoError(t, err)
	assert.Equal(t, concurrency, queueLength)
}
