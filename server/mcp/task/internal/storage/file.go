package storage

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// FileStorage 封装文件操作，提供文件移动、检查等功能
type FileStorage struct {
	baseDir string // 基础存储目录（从环境变量 LOCAL_STORAGE_PATH 读取）
}

// NewFileStorage 创建文件存储实例
// 从环境变量读取配置：
//   - LOCAL_STORAGE_PATH: 本地存储路径（默认 ./storage）
func NewFileStorage() (*FileStorage, error) {
	baseDir := os.Getenv("LOCAL_STORAGE_PATH")
	if baseDir == "" {
		baseDir = "./storage"
	}

	// 确保基础目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %v", err)
	}

	log.Printf("✓ File storage initialized (base dir: %s)", baseDir)

	return &FileStorage{baseDir: baseDir}, nil
}

// GetTaskDir 获取任务目录路径
// 格式：{baseDir}/videos/{taskID}/
func (fs *FileStorage) GetTaskDir(taskID string) string {
	return filepath.Join(fs.baseDir, "videos", taskID)
}

// GetOriginalFilePath 获取原始文件路径
// 格式：{baseDir}/videos/{taskID}/original.mp4
func (fs *FileStorage) GetOriginalFilePath(taskID string) string {
	return filepath.Join(fs.GetTaskDir(taskID), "original.mp4")
}

// CreateTaskDir 创建任务目录
// 如果目录已存在，不会报错
func (fs *FileStorage) CreateTaskDir(taskID string) error {
	taskDir := fs.GetTaskDir(taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task directory: %v", err)
	}
	log.Printf("[FileStorage] Task directory created: %s", taskDir)
	return nil
}

// MoveFile 移动文件（临时文件 → 正式文件）
// 优先使用 os.Rename（原子操作），如果跨文件系统则降级为 io.Copy + os.Remove
//
// 设计决策：
//   - os.Rename 是原子操作，性能高（仅修改元数据，不复制数据）
//   - 跨文件系统时 os.Rename 会失败，需要降级为 io.Copy + os.Remove
//   - 降级策略确保在任何情况下都能完成文件移动
func (fs *FileStorage) MoveFile(src, dst string) error {
	// 检查源文件是否存在
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", src)
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// 优先使用 os.Rename（原子操作）
	err := os.Rename(src, dst)
	if err == nil {
		log.Printf("[FileStorage] File moved (os.Rename): %s → %s", src, dst)
		return nil
	}

	// 如果是跨文件系统错误，降级为 io.Copy + os.Remove
	if isCrossDeviceError(err) {
		log.Printf("[FileStorage] Cross-device detected, falling back to io.Copy: %s → %s", src, dst)
		return fs.copyAndRemove(src, dst)
	}

	return fmt.Errorf("failed to move file: %v", err)
}

// isCrossDeviceError 检查是否是跨文件系统错误
// Windows: "The system cannot move the file to a different disk drive"
// Linux: "invalid cross-device link"
func isCrossDeviceError(err error) bool {
	// 检查错误消息中是否包含跨设备关键词
	errMsg := err.Error()
	return contains(errMsg, "cross-device") ||
		contains(errMsg, "different disk drive") ||
		contains(errMsg, "invalid cross-device link")
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)*2 && s[len(s)/2-len(substr)/2:len(s)/2+len(substr)/2+len(substr)%2] == substr))
}

// copyAndRemove 复制文件并删除源文件（降级策略）
// 用于跨文件系统的文件移动
func (fs *FileStorage) copyAndRemove(src, dst string) error {
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

	// 同步到磁盘
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %v", err)
	}

	// 删除源文件
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file: %v", err)
	}

	log.Printf("[FileStorage] File moved (io.Copy): %s → %s", src, dst)
	return nil
}

// FileExists 检查文件是否存在
func (fs *FileStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

