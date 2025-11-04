package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/task/internal/svc"
	pb "video-in-chinese/task/proto"
)

// GetTaskStatusLogic 查询任务状态逻辑
type GetTaskStatusLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

// NewGetTaskStatusLogic 创建 GetTaskStatusLogic 实例
func NewGetTaskStatusLogic(ctx context.Context, svc *svc.ServiceContext) *GetTaskStatusLogic {
	return &GetTaskStatusLogic{
		ctx: ctx,
		svc: svc,
	}
}

// GetTaskStatus 查询任务状态
// 执行流程（参考 Task-design-detail.md 第二层文档）：
//   1. 从 Redis 读取任务状态（HGETALL task:{task_id}）
//   2. 检查任务是否存在（如果不存在返回 NOT_FOUND 错误）
//   3. 返回任务状态（status, result_file_path, error_message, created_at, updated_at）
func (l *GetTaskStatusLogic) GetTaskStatus(req *pb.GetTaskStatusRequest) (*pb.GetTaskStatusResponse, error) {
	log.Printf("[GetTaskStatus] Querying task status: %s", req.TaskId)

	// 步骤1: 从 Redis 读取任务状态
	fields, err := l.svc.RedisClient.GetTaskFields(l.ctx, req.TaskId)
	if err != nil {
		log.Printf("[GetTaskStatus] Task not found: %s", req.TaskId)
		return nil, fmt.Errorf("task not found: %s", req.TaskId)
	}

	// 步骤2: 解析任务状态
	status := parseTaskStatus(fields["status"])
	resultFilePath := fields["result_file_path"]
	errorMessage := fields["error_message"]
	createdAt := fields["created_at"]
	updatedAt := fields["updated_at"]

	log.Printf("[GetTaskStatus] Task status: %s (status: %s)", req.TaskId, fields["status"])

	// 步骤3: 返回任务状态
	return &pb.GetTaskStatusResponse{
		Status:         status,
		ResultFilePath: resultFilePath,
		ErrorMessage:   errorMessage,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

// parseTaskStatus 解析任务状态字符串为枚举值
func parseTaskStatus(statusStr string) pb.TaskStatus {
	switch statusStr {
	case "PENDING":
		return pb.TaskStatus_PENDING
	case "PROCESSING":
		return pb.TaskStatus_PROCESSING
	case "COMPLETED":
		return pb.TaskStatus_COMPLETED
	case "FAILED":
		return pb.TaskStatus_FAILED
	default:
		return pb.TaskStatus_PENDING
	}
}

