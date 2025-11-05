// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package task

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"video-in-chinese/server/app/gateway/internal/svc"
	"video-in-chinese/server/app/gateway/internal/types"
	"video-in-chinese/server/app/gateway/internal/utils"
	"video-in-chinese/server/mcp/task-gozero/proto"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type UploadTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// Upload a video file to create a new translation task
func NewUploadTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UploadTaskLogic {
	return &UploadTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UploadTaskLogic) UploadTask() (resp *types.UploadTaskResponse, err error) {
	// Step 1: Parse multipart/form-data file stream and original filename
	err = l.r.ParseMultipartForm(int64(l.svcCtx.Config.MaxUploadSizeMB) * 1024 * 1024)
	if err != nil {
		l.Errorf("[UploadTask] Failed to parse multipart form: %v", err)
		return nil, fmt.Errorf("failed to parse upload form")
	}

	file, header, err := l.r.FormFile("file")
	if err != nil {
		l.Errorf("[UploadTask] Failed to get form file: %v", err)
		return nil, fmt.Errorf("no file uploaded")
	}
	defer file.Close()

	originalFilename := header.Filename
	fileSize := header.Size
	l.Infof("[UploadTask] Received file: %s, size: %d bytes", originalFilename, fileSize)

	// Step 2: Check file size (MAX_UPLOAD_SIZE_MB)
	maxSizeBytes := int64(l.svcCtx.Config.MaxUploadSizeMB) * 1024 * 1024
	if fileSize > maxSizeBytes {
		l.Errorf("[UploadTask] File too large: %d bytes (max: %d bytes)", fileSize, maxSizeBytes)
		return nil, fmt.Errorf("file too large: max size is %d MB", l.svcCtx.Config.MaxUploadSizeMB)
	}

	// Step 3: Generate unique temporary filename (UUID + extension)
	ext := filepath.Ext(originalFilename)
	tempFilename := uuid.New().String() + ext
	tempFilePath := filepath.Join(l.svcCtx.Config.TempStoragePath, tempFilename)

	// Ensure temp directory exists
	if err := os.MkdirAll(l.svcCtx.Config.TempStoragePath, 0755); err != nil {
		l.Errorf("[UploadTask] Failed to create temp directory: %v", err)
		return nil, fmt.Errorf("internal error")
	}

	// Step 4: Check disk available space (fileSize * 3 + 500MB)
	requiredSpace := fileSize*3 + 500*1024*1024
	if err := utils.CheckDiskSpace(l.svcCtx.Config.TempStoragePath, requiredSpace); err != nil {
		l.Errorf("[UploadTask] Insufficient disk space: %v", err)
		return nil, fmt.Errorf("insufficient disk space")
	}

	// Step 5: Stream save file to temporary directory (io.Copy)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		l.Errorf("[UploadTask] Failed to create temp file: %v", err)
		return nil, fmt.Errorf("internal error")
	}
	defer tempFile.Close()

	// Step 6: Copy file content
	written, err := io.Copy(tempFile, file)
	if err != nil {
		l.Errorf("[UploadTask] Failed to copy file: %v", err)
		os.Remove(tempFilePath) // Clean up on error
		return nil, fmt.Errorf("failed to save file")
	}
	l.Infof("[UploadTask] File saved: %s, written: %d bytes", tempFilePath, written)

	// Step 7: Validate file MIME Type (whitelist)
	mimeType, err := utils.DetectAndValidateMimeType(tempFilePath, l.svcCtx.Config.SupportedMimeTypes)
	if err != nil {
		l.Errorf("[UploadTask] MIME type validation failed: %v", err)
		os.Remove(tempFilePath) // Clean up on error
		return nil, fmt.Errorf("unsupported file type")
	}
	l.Infof("[UploadTask] MIME type validated: %s", mimeType)

	// Step 8: Call Task service CreateTask (gRPC)
	taskResp, err := l.svcCtx.TaskRpcClient.CreateTask(l.ctx, &proto.CreateTaskRequest{
		TempFilePath: tempFilePath,
	})
	if err != nil {
		l.Errorf("[UploadTask] Failed to create task: %v", err)
		os.Remove(tempFilePath) // Clean up on error
		return nil, fmt.Errorf("failed to create task")
	}

	// Log original filename for reference
	l.Infof("[UploadTask] Original filename: %s", originalFilename)

	// Step 9: Return task ID
	resp = &types.UploadTaskResponse{
		TaskId: taskResp.TaskId,
	}

	l.Infof("[UploadTask] Task created successfully: taskId=%s", taskResp.TaskId)
	return resp, nil
}
