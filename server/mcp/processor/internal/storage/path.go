package storage

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/zeromicro/go-zero/core/logx"
)

// PathManager manages file paths for the Processor service.
//
// This manager provides methods for generating task-related file paths
// and creating necessary directories.
type PathManager struct {
	baseDir string
}

// NewPathManager creates a new PathManager instance.
//
// Parameters:
//   - baseDir: base directory for all task files
//
// Returns:
//   - *PathManager: initialized path manager
func NewPathManager(baseDir string) *PathManager {
	return &PathManager{
		baseDir: baseDir,
	}
}

// GetTaskDir returns the task directory path.
//
// Parameters:
//   - taskID: task ID
//
// Returns:
//   - string: task directory path (e.g., "./data/videos/{taskID}")
func (p *PathManager) GetTaskDir(taskID string) string {
	return filepath.Join(p.baseDir, taskID)
}

// GetIntermediateDir returns the intermediate files directory path.
//
// Parameters:
//   - taskID: task ID
//
// Returns:
//   - string: intermediate directory path (e.g., "./data/videos/{taskID}/intermediate")
func (p *PathManager) GetIntermediateDir(taskID string) string {
	return filepath.Join(p.baseDir, taskID, "intermediate")
}

// GetVideoPath returns the original video file path.
//
// Parameters:
//   - taskID: task ID
//
// Returns:
//   - string: video file path (e.g., "./data/videos/{taskID}/video.mp4")
func (p *PathManager) GetVideoPath(taskID string) string {
	return filepath.Join(p.baseDir, taskID, "video.mp4")
}

// GetIntermediatePath returns an intermediate file path.
//
// Parameters:
//   - taskID: task ID
//   - filename: intermediate file name
//
// Returns:
//   - string: intermediate file path (e.g., "./data/videos/{taskID}/intermediate/{filename}")
func (p *PathManager) GetIntermediatePath(taskID, filename string) string {
	return filepath.Join(p.GetIntermediateDir(taskID), filename)
}

// GetOutputPath returns the final output video path.
//
// Parameters:
//   - taskID: task ID
//
// Returns:
//   - string: output video path (e.g., "./data/videos/{taskID}/output.mp4")
func (p *PathManager) GetOutputPath(taskID string) string {
	return filepath.Join(p.baseDir, taskID, "output.mp4")
}

// EnsureIntermediateDir creates the intermediate directory if it doesn't exist.
//
// Parameters:
//   - taskID: task ID
//
// Returns:
//   - error: error if directory creation fails
func (p *PathManager) EnsureIntermediateDir(taskID string) error {
	dir := p.GetIntermediateDir(taskID)
	
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		logx.Errorf("[PathManager] Failed to create intermediate directory %s: %v", dir, err)
		return fmt.Errorf("failed to create intermediate directory: %w", err)
	}
	
	logx.Infof("[PathManager] Created intermediate directory: %s", dir)
	return nil
}

// CleanupIntermediateFiles removes all intermediate files for a task.
//
// Parameters:
//   - taskID: task ID
//
// Returns:
//   - error: error if cleanup fails
func (p *PathManager) CleanupIntermediateFiles(taskID string) error {
	dir := p.GetIntermediateDir(taskID)
	
	err := os.RemoveAll(dir)
	if err != nil {
		logx.Errorf("[PathManager] Failed to cleanup intermediate files %s: %v", dir, err)
		return fmt.Errorf("failed to cleanup intermediate files: %w", err)
	}
	
	logx.Infof("[PathManager] Cleaned up intermediate files: %s", dir)
	return nil
}

