// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package task

import (
	"context"
	"fmt"
	"path/filepath"

	"video-in-chinese/server/app/gateway/internal/svc"
	"video-in-chinese/server/app/gateway/internal/types"
	"video-in-chinese/server/mcp/task/proto"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get task status by task ID
func NewGetTaskStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskStatusLogic {
	return &GetTaskStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTaskStatusLogic) GetTaskStatus(req *types.GetTaskStatusRequest) (resp *types.GetTaskStatusResponse, err error) {
	// Step 1: Parse path parameter taskId (already done by goctl)
	taskId := req.TaskId
	l.Infof("[GetTaskStatus] Querying task status: taskId=%s", taskId)

	// Step 2: Parameter validation (taskId non-empty)
	if taskId == "" {
		l.Errorf("[GetTaskStatus] Empty taskId")
		return nil, fmt.Errorf("taskId is required")
	}

	// Step 3: Call Task service GetTaskStatus (gRPC)
	taskResp, err := l.svcCtx.TaskRpcClient.GetTaskStatus(l.ctx, &proto.GetTaskStatusRequest{
		TaskId: taskId,
	})
	if err != nil {
		l.Errorf("[GetTaskStatus] Failed to get task status: %v", err)
		return nil, fmt.Errorf("failed to get task status")
	}

	// Step 4: Encapsulate response GetTaskStatusResponse
	resp = &types.GetTaskStatusResponse{
		TaskId:       taskId,
		Status:       taskResp.Status.String(),
		ErrorMessage: taskResp.ErrorMessage,
	}

	// If task is completed, construct result URL
	if taskResp.Status.String() == "COMPLETED" && taskResp.ResultFilePath != "" {
		// Extract filename from result file path
		filename := filepath.Base(taskResp.ResultFilePath)
		resp.ResultUrl = fmt.Sprintf("/v1/tasks/download/%s/%s", taskId, filename)
	}

	// Step 5: Error handling (gRPC error conversion already done above)
	l.Infof("[GetTaskStatus] Task status retrieved: taskId=%s, status=%s", taskId, resp.Status)
	return resp, nil
}
