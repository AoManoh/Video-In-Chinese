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

	"github.com/zeromicro/go-zero/core/logx"
)

type DownloadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
}

// Download task result file
func NewDownloadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter) *DownloadFileLogic {
	return &DownloadFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
	}
}

func (l *DownloadFileLogic) DownloadFile(req *types.DownloadFileRequest) error {
	// Step 1: Parse path parameters taskId and fileName (already done by goctl)
	taskId := req.TaskId
	fileName := req.FileName
	l.Infof("[DownloadFile] Download request: taskId=%s, fileName=%s", taskId, fileName)

	// Step 2: Path security check (prevent path traversal attack)
	basePath := l.svcCtx.Config.LocalStoragePath
	taskDir := filepath.Join(basePath, taskId)
	filePath := filepath.Join(taskDir, fileName)

	if err := utils.IsPathSafe(basePath, filePath); err != nil {
		l.Errorf("[DownloadFile] Path security check failed: %v", err)
		return fmt.Errorf("invalid file path")
	}

	// Step 3: Construct full file path (LOCAL_STORAGE_PATH/taskId/fileName)
	// Already done above

	// Step 4: Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		l.Errorf("[DownloadFile] File not found: %s", filePath)
		return fmt.Errorf("file not found")
	}

	// Step 5: Set response headers (Content-Type, Content-Disposition)
	mimeType, err := utils.DetectMimeType(filePath)
	if err != nil {
		l.Errorf("[DownloadFile] Failed to detect MIME type: %v", err)
		mimeType = "application/octet-stream"
	}

	l.w.Header().Set("Content-Type", mimeType)
	l.w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	// Step 6: Stream return file content (io.Copy)
	file, err := os.Open(filePath)
	if err != nil {
		l.Errorf("[DownloadFile] Failed to open file: %v", err)
		return fmt.Errorf("failed to open file")
	}
	defer file.Close()

	written, err := io.Copy(l.w, file)
	if err != nil {
		l.Errorf("[DownloadFile] Failed to copy file: %v", err)
		return fmt.Errorf("failed to send file")
	}

	l.Infof("[DownloadFile] File sent successfully: %s, written: %d bytes", filePath, written)
	return nil
}
