package logic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"video-in-chinese/server/mcp/task-gozero/internal/svc"
	"video-in-chinese/server/mcp/task-gozero/proto"
)

// GetTaskStatusLogic encapsulates the business logic for task status queries.
//
// This struct holds the context and service dependencies needed to execute
// the 3-step status query workflow. It is created per-request and is not reused.
//
// Integration with go-zero:
//   - Embeds logx.Logger for structured logging
//   - Uses ServiceContext for dependency injection
//   - Follows go-zero logic layer conventions
type GetTaskStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewGetTaskStatusLogic creates a new GetTaskStatusLogic instance.
//
// This constructor is called by the gRPC handler (server layer) for each GetTaskStatus request.
//
// Parameters:
//   - ctx: Request context (carries deadline, cancellation signal, request-scoped values)
//   - svcCtx: Service context (provides access to Redis and FileStorage)
//
// Returns a new GetTaskStatusLogic instance ready to process a request.
func NewGetTaskStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskStatusLogic {
	return &GetTaskStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTaskStatus implements the 3-step task status query workflow.
//
// Workflow (Reference: Task-design-detail.md, second-layer documentation):
//  1. Read task state from Redis (HGETALL task:{task_id})
//  2. Check if task exists (return NOT_FOUND error if not)
//  3. Return task status (status, result_file_path, error_message, created_at, updated_at)
//
// Task Status Values:
//   - PENDING: Task is in queue, waiting for Processor to pick it up
//   - PROCESSING: Task is being processed by Processor service
//   - COMPLETED: Task processing completed successfully, result file is available
//   - FAILED: Task processing failed, error_message contains details
//
// Design Decisions:
//   - Status is stored as string in Redis for human readability
//   - parseTaskStatus converts string to protobuf enum for type safety
//   - Empty Hash (len(fields) == 0) indicates task does not exist
//
// Error Handling:
//   - Returns error if task does not exist (task not found)
//   - Returns error if Redis operation fails
//   - Invalid status strings default to PENDING (defensive programming)
//
// Parameters:
//   - in: GetTaskStatusRequest containing task_id (UUID v4 string)
//
// Returns:
//   - GetTaskStatusResponse containing status, result_file_path, error_message, timestamps
//   - error if task does not exist or Redis operation fails
func (l *GetTaskStatusLogic) GetTaskStatus(in *proto.GetTaskStatusRequest) (*proto.GetTaskStatusResponse, error) {
	l.Infof("[GetTaskStatus] Querying task status: %s", in.TaskId)

	// 步骤1: 从 Redis 读取任务状态
	fields, err := l.svcCtx.RedisClient.GetTaskFields(l.ctx, in.TaskId)
	if err != nil {
		l.Errorf("[GetTaskStatus] Task not found: %s", in.TaskId)
		return nil, fmt.Errorf("task not found: %s", in.TaskId)
	}

	// 步骤2: 解析任务状态
	status := parseTaskStatus(fields["status"])
	resultFilePath := fields["result_file_path"]
	errorMessage := fields["error_message"]
	createdAt := fields["created_at"]
	updatedAt := fields["updated_at"]

	l.Infof("[GetTaskStatus] Task status: %s (status: %s)", in.TaskId, fields["status"])

	// 步骤3: 返回任务状态
	return &proto.GetTaskStatusResponse{
		Status:         status,
		ResultFilePath: resultFilePath,
		ErrorMessage:   errorMessage,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

// parseTaskStatus converts a status string to a protobuf enum value.
//
// This function provides type-safe conversion from Redis string values
// to strongly-typed protobuf enums used in gRPC responses.
//
// Status Mapping:
//   - "PENDING" → TaskStatus_PENDING
//   - "PROCESSING" → TaskStatus_PROCESSING
//   - "COMPLETED" → TaskStatus_COMPLETED
//   - "FAILED" → TaskStatus_FAILED
//   - (any other value) → TaskStatus_PENDING (default)
//
// Why Default to PENDING?
//   - Defensive programming: Unknown status values are treated as pending
//   - Prevents client errors from invalid enum values
//   - Allows for graceful degradation if new status values are added
//
// Parameters:
//   - statusStr: Status string from Redis (e.g., "PENDING", "COMPLETED")
//
// Returns the corresponding TaskStatus enum value.
func parseTaskStatus(statusStr string) proto.TaskStatus {
	switch statusStr {
	case "PENDING":
		return proto.TaskStatus_PENDING
	case "PROCESSING":
		return proto.TaskStatus_PROCESSING
	case "COMPLETED":
		return proto.TaskStatus_COMPLETED
	case "FAILED":
		return proto.TaskStatus_FAILED
	default:
		return proto.TaskStatus_PENDING
	}
}
