package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zeromicro/go-zero/core/logx"
)

// FileStorage encapsulates file operations for video storage.
//
// Directory Structure:
//   - {baseDir}/videos/{taskID}/original.mp4 (original video)
//   - {baseDir}/videos/{taskID}/result.mp4 (translated video, created by Processor)
//
// Design Decisions:
//   - Each task has its own directory to isolate files and simplify cleanup
//   - File names are fixed (original.mp4, result.mp4) for consistency
//   - MoveFile uses os.Rename (atomic) with fallback to io.Copy (cross-filesystem)
//
// Integration with go-zero:
//   - Uses logx for logging (go-zero standard logging library)
//   - Configuration passed from ServiceContext (go-zero dependency injection)
type FileStorage struct {
	baseDir string // Base storage directory (from config.LocalStoragePath)
}

// NewFileStorage creates a new FileStorage instance using go-zero configuration.
//
// Configuration is passed from go-zero config (config.LocalStoragePath):
//   - baseDir: Base directory for file storage (e.g., "./storage")
//
// The base directory is created if it does not exist (with 0755 permissions).
//
// Parameters:
//   - baseDir: Base directory path from config
//
// Returns an error if the base directory cannot be created.
func NewFileStorage(baseDir string) (*FileStorage, error) {
	if baseDir == "" {
		baseDir = "./storage"
	}

	// 确保基础目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %v", err)
	}

	logx.Infof("File storage initialized (base dir: %s)", baseDir)

	return &FileStorage{baseDir: baseDir}, nil
}

// GetTaskDir returns the directory path for a task.
//
// Path Format: {baseDir}/videos/{taskID}/
//
// This directory contains all files related to the task:
//   - original.mp4: Original video uploaded by the user
//   - result.mp4: Translated video (created by Processor service)
//   - Intermediate files (audio, subtitles, etc., created by Processor)
func (fs *FileStorage) GetTaskDir(taskID string) string {
	return filepath.Join(fs.baseDir, "videos", taskID)
}

// GetOriginalFilePath returns the path to the original video file.
//
// Path Format: {baseDir}/videos/{taskID}/original.mp4
//
// This file is created by the CreateTask operation when the temporary
// file is moved to its permanent location.
func (fs *FileStorage) GetOriginalFilePath(taskID string) string {
	return filepath.Join(fs.GetTaskDir(taskID), "original.mp4")
}

// CreateTaskDir creates the task directory if it does not exist.
//
// This method is idempotent: if the directory already exists, no error is returned.
// The directory is created with 0755 permissions (rwxr-xr-x).
//
// Parameters:
//   - taskID: UUID v4 string identifying the task
//
// Returns an error if the directory cannot be created.
func (fs *FileStorage) CreateTaskDir(taskID string) error {
	taskDir := fs.GetTaskDir(taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task directory: %v", err)
	}
	logx.Infof("[FileStorage] Task directory created: %s", taskDir)
	return nil
}

// MoveFile moves a file from source to destination with cross-filesystem fallback.
//
// File Move Strategy:
//  1. Primary: os.Rename (atomic, fast, same filesystem only)
//  2. Fallback: io.Copy + os.Remove (slower, works across filesystems)
//
// Why os.Rename First?
//   - Atomic operation: Either succeeds completely or fails (no partial state)
//   - Fast: Only updates filesystem metadata, no data copying
//   - Safe: No risk of partial file corruption
//
// Why Fallback to io.Copy?
//   - os.Rename fails when source and destination are on different filesystems
//   - io.Copy works across filesystems but is slower (copies all data)
//   - After successful copy, source file is removed
//
// Parameters:
//   - src: Source file path (e.g., temporary upload path)
//   - dst: Destination file path (e.g., permanent storage path)
//
// Returns an error if both strategies fail.
func (fs *FileStorage) MoveFile(src, dst string) error {
	// 策略 1: 尝试 os.Rename (原子操作，快速)
	if err := os.Rename(src, dst); err == nil {
		logx.Infof("[FileStorage] File moved (os.Rename): %s -> %s", src, dst)
		return nil
	}

	// 策略 2: 跨文件系统移动 (io.Copy + os.Remove)
	logx.Infof("[FileStorage] os.Rename failed, falling back to io.Copy: %s -> %s", src, dst)

	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	// 复制文件内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// 确保数据写入磁盘
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %v", err)
	}

	// 删除源文件
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file: %v", err)
	}

	logx.Infof("[FileStorage] File moved (io.Copy): %s -> %s", src, dst)
	return nil
}

// FileExists checks if a file exists at the given path.
//
// This method is useful for validating file operations and checking
// whether a file has been successfully created or moved.
//
// Parameters:
//   - path: File path to check
//
// Returns:
//   - true if the file exists
//   - false if the file does not exist or an error occurs
func (fs *FileStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileSize returns the size of a file in bytes.
//
// This method is useful for validating file uploads and monitoring
// storage usage.
//
// Parameters:
//   - path: File path to check
//
// Returns:
//   - File size in bytes
//   - Error if the file does not exist or cannot be accessed
func (fs *FileStorage) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %v", err)
	}
	return info.Size(), nil
}

// DeleteTaskDir deletes the task directory and all its contents.
//
// This method is used for cleanup when a task is cancelled or completed.
// It recursively deletes all files and subdirectories.
//
// Parameters:
//   - taskID: UUID v4 string identifying the task
//
// Returns an error if the directory cannot be deleted.
func (fs *FileStorage) DeleteTaskDir(taskID string) error {
	taskDir := fs.GetTaskDir(taskID)
	if err := os.RemoveAll(taskDir); err != nil {
		return fmt.Errorf("failed to delete task directory: %v", err)
	}
	logx.Infof("[FileStorage] Task directory deleted: %s", taskDir)
	return nil
}

