// Package logic implements business logic for the Task service.
//
// This package contains the core business logic for task creation and status queries.
// It orchestrates operations across storage layers (Redis and file system) to implement
// the 7-step task creation workflow and 3-step status query workflow.
//
// Design Pattern:
//   - Each RPC method has a corresponding Logic struct (CreateTaskLogic, GetTaskStatusLogic)
//   - Logic structs hold context and service dependencies (dependency injection)
//   - Business logic is separated from gRPC transport layer (server layer)
//   - Uses go-zero logx for structured logging
//
// Reference: Task-design-detail.md (second-layer documentation)
package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"video-in-chinese/server/mcp/task/internal/svc"
	"video-in-chinese/server/mcp/task/proto"
)

// CreateTaskLogic encapsulates the business logic for task creation.
//
// This struct holds the context and service dependencies needed to execute
// the 7-step task creation workflow. It is created per-request and is not reused.
//
// Integration with go-zero:
//   - Embeds logx.Logger for structured logging
//   - Uses ServiceContext for dependency injection
//   - Follows go-zero logic layer conventions
type CreateTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewCreateTaskLogic creates a new CreateTaskLogic instance.
//
// This constructor is called by the gRPC handler (server layer) for each CreateTask request.
//
// Parameters:
//   - ctx: Request context (carries deadline, cancellation signal, request-scoped values)
//   - svcCtx: Service context (provides access to Redis and FileStorage)
//
// Returns a new CreateTaskLogic instance ready to process a request.
func NewCreateTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTaskLogic {
	return &CreateTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateTask implements the 7-step task creation workflow.
//
// Workflow (Reference: Task-design-detail.md, second-layer documentation):
//  1. Generate task ID (UUID v4)
//  2. Build permanent file path ({LOCAL_STORAGE_PATH}/videos/{task_id}/original.mp4)
//  3. Create task directory (if not exists)
//  4. File handoff (temporary file → permanent file, using os.Rename with fallback)
//  5. Create task record (Redis Hash, initial status: PENDING)
//  6. Push task to queue (Redis LPUSH, Key: task:pending)
//  7. Return task ID
//
// Design Decisions:
//   - UUID v4 ensures globally unique task IDs without coordination
//   - File is moved before Redis operations to fail fast on file errors
//   - Task record is created before queue push to ensure state exists when Processor pulls
//   - All operations are logged using go-zero logx for debugging and monitoring
//
// Error Handling:
//   - If any step fails, the operation is aborted and an error is returned
//   - File operations are atomic (os.Rename) or have fallback (io.Copy)
//   - Redis operations are atomic (HSET, LPUSH)
//   - No cleanup is performed on failure (files and Redis records are left for manual inspection)
//
// Parameters:
//   - in: CreateTaskRequest containing temp_file_path (path to uploaded file)
//
// Returns:
//   - CreateTaskResponse containing task_id (UUID v4 string)
//   - error if any step fails
func (l *CreateTaskLogic) CreateTask(in *proto.CreateTaskRequest) (*proto.CreateTaskResponse, error) {
	l.Infof("[CreateTask] Starting task creation (temp_file: %s)", in.TempFilePath)

	// 步骤1: 生成任务 ID（UUID v4）
	taskID := uuid.New().String()
	l.Infof("[CreateTask] Task ID generated: %s", taskID)

	// 步骤2: 构建正式文件路径
	originalFilePath := l.svcCtx.FileStorage.GetOriginalFilePath(taskID)
	l.Infof("[CreateTask] Original file path: %s", originalFilePath)

	// 步骤3: 创建任务目录
	if err := l.svcCtx.FileStorage.CreateTaskDir(taskID); err != nil {
		l.Errorf("[CreateTask] Failed to create task directory: %v", err)
		return nil, fmt.Errorf("failed to create task directory: %v", err)
	}

	// 步骤4: 文件交接（临时文件 → 正式文件）
	if err := l.svcCtx.FileStorage.MoveFile(in.TempFilePath, originalFilePath); err != nil {
		l.Errorf("[CreateTask] Failed to move file: %v", err)
		return nil, fmt.Errorf("failed to move file: %v", err)
	}
	l.Infof("[CreateTask] File moved: %s → %s", in.TempFilePath, originalFilePath)

	// 步骤5: 创建任务记录（Redis Hash）
	now := time.Now().Format(time.RFC3339)
	taskFields := map[string]interface{}{
		"task_id":            taskID,
		"status":             "PENDING",
		"original_file_path": originalFilePath,
		"result_file_path":   "",
		"error_message":      "",
		"created_at":         now,
		"updated_at":         now,
	}

	if err := l.svcCtx.RedisClient.SetTaskFields(l.ctx, taskID, taskFields); err != nil {
		l.Errorf("[CreateTask] Failed to create task record: %v", err)
		return nil, fmt.Errorf("failed to create task record: %v", err)
	}
	l.Infof("[CreateTask] Task record created in Redis: %s", taskID)

	// 步骤6: 推入任务到队列（Redis LPUSH）
	if err := l.svcCtx.RedisClient.PushTask(l.ctx, taskID, originalFilePath); err != nil {
		l.Errorf("[CreateTask] Failed to push task to queue: %v", err)
		return nil, fmt.Errorf("failed to push task to queue: %v", err)
	}
	l.Infof("[CreateTask] Task pushed to queue: %s", taskID)

	// 步骤7: 返回任务 ID
	l.Infof("[CreateTask] ✓ Task created successfully: %s", taskID)
	return &proto.CreateTaskResponse{
		TaskId: taskID,
	}, nil
}
