package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"video-in-chinese/server/mcp/task/internal/config"
	"video-in-chinese/server/mcp/task/internal/server"
	"video-in-chinese/server/mcp/task/internal/svc"
	"video-in-chinese/server/mcp/task/proto"
)

type integrationEnv struct {
	client      proto.TaskServiceClient
	conn        *grpc.ClientConn
	svcCtx      *svc.ServiceContext
	cleanupFunc func()
}

func createIntegrationTempFile(t *testing.T, dir string, content []byte) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	file, err := os.CreateTemp(dir, "integration-*.bin")
	require.NoError(t, err)
	_, err = file.Write(content)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	return file.Name()
}

func setupIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()
	ctx := context.Background()

	redisContainer, err := testredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	storageRoot := filepath.Join(t.TempDir(), "storage")
	cfg := config.Config{
		Redis: redis.RedisConf{
			Host: endpoint,
			Type: "node",
		},
		LocalStoragePath: storageRoot,
	}

	svcCtx := svc.NewServiceContext(cfg)

	grpcServer := grpc.NewServer()
	proto.RegisterTaskServiceServer(grpcServer, server.NewTaskServiceServer(svcCtx))

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("gRPC server stopped: %v", err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	cleanup := func() {
		conn.Close()
		grpcServer.GracefulStop()
		_ = svcCtx.RedisClient.Close()
		_ = os.RemoveAll(storageRoot)
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	}

	return &integrationEnv{
		client:      proto.NewTaskServiceClient(conn),
		conn:        conn,
		svcCtx:      svcCtx,
		cleanupFunc: cleanup,
	}
}

func TestIntegration_CreateAndGetTask(t *testing.T) {
	env := setupIntegrationEnv(t)
	defer env.cleanupFunc()

	tempFile := createIntegrationTempFile(t, t.TempDir(), []byte("video"))

	createResp, err := env.client.CreateTask(context.Background(), &proto.CreateTaskRequest{TempFilePath: tempFile})
	require.NoError(t, err)
	require.NotEmpty(t, createResp.TaskId)

	statusResp, err := env.client.GetTaskStatus(context.Background(), &proto.GetTaskStatusRequest{TaskId: createResp.TaskId})
	require.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_PENDING, statusResp.Status)
	assert.Empty(t, statusResp.ResultFilePath)

	original := env.svcCtx.FileStorage.GetOriginalFilePath(createResp.TaskId)
	_, err = os.Stat(original)
	require.NoError(t, err)
}

func TestIntegration_MultipleTasksSequential(t *testing.T) {
	env := setupIntegrationEnv(t)
	defer env.cleanupFunc()

	ids := make(map[string]struct{})
	for i := 0; i < 3; i++ {
		tempFile := createIntegrationTempFile(t, t.TempDir(), []byte("video"))
		resp, err := env.client.CreateTask(context.Background(), &proto.CreateTaskRequest{TempFilePath: tempFile})
		require.NoError(t, err)
		ids[resp.TaskId] = struct{}{}
	}

	assert.Len(t, ids, 3)

	length, err := env.svcCtx.RedisClient.GetQueueLength(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, length)
}

func TestIntegration_TaskStatusTransition(t *testing.T) {
	env := setupIntegrationEnv(t)
	defer env.cleanupFunc()

	tempFile := createIntegrationTempFile(t, t.TempDir(), []byte("video"))
	resp, err := env.client.CreateTask(context.Background(), &proto.CreateTaskRequest{TempFilePath: tempFile})
	require.NoError(t, err)

	now := time.Now().Format(time.RFC3339)
	require.NoError(t, env.svcCtx.RedisClient.SetTaskFields(context.Background(), resp.TaskId, map[string]interface{}{
		"status":           "COMPLETED",
		"result_file_path": "/result.mp4",
		"updated_at":       now,
	}))

	statusResp, err := env.client.GetTaskStatus(context.Background(), &proto.GetTaskStatusRequest{TaskId: resp.TaskId})
	require.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_COMPLETED, statusResp.Status)
	assert.Equal(t, "/result.mp4", statusResp.ResultFilePath)
}

func TestIntegration_FileHandoff(t *testing.T) {
	env := setupIntegrationEnv(t)
	defer env.cleanupFunc()

	tempDir := t.TempDir()
	tempFile := createIntegrationTempFile(t, tempDir, []byte("handoff"))

	resp, err := env.client.CreateTask(context.Background(), &proto.CreateTaskRequest{TempFilePath: tempFile})
	require.NoError(t, err)

	_, err = os.Stat(tempFile)
	assert.True(t, os.IsNotExist(err))

	original := env.svcCtx.FileStorage.GetOriginalFilePath(resp.TaskId)
	data, err := os.ReadFile(original)
	require.NoError(t, err)
	assert.Equal(t, "handoff", string(data))
}

func TestIntegration_RedisQueueOperation(t *testing.T) {
	env := setupIntegrationEnv(t)
	defer env.cleanupFunc()

	resp, err := env.client.CreateTask(context.Background(), &proto.CreateTaskRequest{TempFilePath: createIntegrationTempFile(t, t.TempDir(), []byte("job"))})
	require.NoError(t, err)

	exists, err := env.svcCtx.RedisClient.TaskExists(context.Background(), resp.TaskId)
	require.NoError(t, err)
	assert.True(t, exists)

	queueLen, err := env.svcCtx.RedisClient.GetQueueLength(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, queueLen)
}
