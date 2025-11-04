package logic

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"video-in-chinese/task/internal/svc"
	pb "video-in-chinese/task/proto"
)

// CreateTaskLogic 创建任务逻辑
type CreateTaskLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

// NewCreateTaskLogic 创建 CreateTaskLogic 实例
func NewCreateTaskLogic(ctx context.Context, svc *svc.ServiceContext) *CreateTaskLogic {
	return &CreateTaskLogic{
		ctx: ctx,
		svc: svc,
	}
}

// CreateTask 创建任务
// 执行流程（参考 Task-design-detail.md 第二层文档）：
//   1. 生成任务 ID（UUID v4）
//   2. 构建正式文件路径（{LOCAL_STORAGE_PATH}/videos/{task_id}/original.mp4）
//   3. 创建任务目录（如果不存在）
//   4. 文件交接（临时文件 → 正式文件，使用 os.Rename）
//   5. 创建任务记录（Redis Hash，初始状态: PENDING）
//   6. 推入任务到队列（Redis LPUSH, Key: task:pending）
//   7. 返回任务 ID
func (l *CreateTaskLogic) CreateTask(req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	log.Printf("[CreateTask] Starting task creation (temp_file: %s)", req.TempFilePath)

	// 步骤1: 生成任务 ID（UUID v4）
	taskID := uuid.New().String()
	log.Printf("[CreateTask] Task ID generated: %s", taskID)

	// 步骤2: 构建正式文件路径
	originalFilePath := l.svc.FileStorage.GetOriginalFilePath(taskID)
	log.Printf("[CreateTask] Original file path: %s", originalFilePath)

	// 步骤3: 创建任务目录
	if err := l.svc.FileStorage.CreateTaskDir(taskID); err != nil {
		log.Printf("[CreateTask] Failed to create task directory: %v", err)
		return nil, fmt.Errorf("failed to create task directory: %v", err)
	}

	// 步骤4: 文件交接（临时文件 → 正式文件）
	if err := l.svc.FileStorage.MoveFile(req.TempFilePath, originalFilePath); err != nil {
		log.Printf("[CreateTask] Failed to move file: %v", err)
		return nil, fmt.Errorf("failed to move file: %v", err)
	}
	log.Printf("[CreateTask] File moved: %s → %s", req.TempFilePath, originalFilePath)

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

	if err := l.svc.RedisClient.SetTaskFields(l.ctx, taskID, taskFields); err != nil {
		log.Printf("[CreateTask] Failed to create task record: %v", err)
		return nil, fmt.Errorf("failed to create task record: %v", err)
	}
	log.Printf("[CreateTask] Task record created in Redis: %s", taskID)

	// 步骤6: 推入任务到队列（Redis LPUSH）
	if err := l.svc.RedisClient.PushTask(l.ctx, taskID); err != nil {
		log.Printf("[CreateTask] Failed to push task to queue: %v", err)
		return nil, fmt.Errorf("failed to push task to queue: %v", err)
	}
	log.Printf("[CreateTask] Task pushed to queue: %s", taskID)

	// 步骤7: 返回任务 ID
	log.Printf("[CreateTask] ✓ Task created successfully: %s", taskID)
	return &pb.CreateTaskResponse{
		TaskId: taskID,
	}, nil
}

