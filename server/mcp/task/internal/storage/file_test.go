package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFileStorage(t *testing.T) (*FileStorage, string) {
	t.Helper()
	baseDir := filepath.Join(t.TempDir(), "storage")
	fs, err := NewFileStorage(baseDir)
	require.NoError(t, err)
	return fs, baseDir
}

func createTempContentFile(t *testing.T, dir string, content []byte) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	tempFile, err := os.CreateTemp(dir, "temp-*.bin")
	require.NoError(t, err)
	_, err = tempFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tempFile.Close())
	return tempFile.Name()
}

func TestFileStorage_MoveFileSuccess(t *testing.T) {
	fs, baseDir := setupFileStorage(t)
	taskID := "task-success"
	require.NoError(t, fs.CreateTaskDir(taskID))

	srcPath := createTempContentFile(t, t.TempDir(), []byte("file-content"))
	dstPath := fs.GetOriginalFilePath(taskID)

	err := fs.MoveFile(srcPath, dstPath)
	require.NoError(t, err)

	// 源文件应被移除
	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err))

	// 目标文件存在且内容一致
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "file-content", string(data))

	// 验证位于存储目录下
	assert.True(t, fs.FileExists(dstPath))
	assert.Contains(t, dstPath, baseDir)
}

func TestFileStorage_MoveFile_SourceNotExist(t *testing.T) {
	fs, _ := setupFileStorage(t)
	taskID := "missing"
	require.NoError(t, fs.CreateTaskDir(taskID))

	dstPath := fs.GetOriginalFilePath(taskID)
	err := fs.MoveFile(filepath.Join(os.TempDir(), "non-existent.bin"), dstPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source file")
}

func TestFileStorage_CreateTaskDirIdempotent(t *testing.T) {
	fs, _ := setupFileStorage(t)
	taskID := "idempotent"

	require.NoError(t, fs.CreateTaskDir(taskID))
	require.NoError(t, fs.CreateTaskDir(taskID))

	info, err := os.Stat(fs.GetTaskDir(taskID))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestFileStorage_GetPaths(t *testing.T) {
	fs, baseDir := setupFileStorage(t)
	taskID := "path-check"

	expectedDir := filepath.Join(baseDir, "videos", taskID)
	assert.Equal(t, expectedDir, fs.GetTaskDir(taskID))
	assert.Equal(t, filepath.Join(expectedDir, "original.mp4"), fs.GetOriginalFilePath(taskID))
}

func TestFileStorage_FileExistsAndSize(t *testing.T) {
	fs, _ := setupFileStorage(t)
	taskID := "size"
	require.NoError(t, fs.CreateTaskDir(taskID))

	dstPath := fs.GetOriginalFilePath(taskID)
	require.NoError(t, os.WriteFile(dstPath, []byte("12345"), 0o644))

	assert.True(t, fs.FileExists(dstPath))
	size, err := fs.GetFileSize(dstPath)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)

	assert.False(t, fs.FileExists(filepath.Join(os.TempDir(), "not-exist")))
}

func TestFileStorage_DeleteTaskDir(t *testing.T) {
	fs, _ := setupFileStorage(t)
	taskID := "delete"
	require.NoError(t, fs.CreateTaskDir(taskID))

	path := fs.GetTaskDir(taskID)
	require.NoError(t, os.WriteFile(fs.GetOriginalFilePath(taskID), []byte("data"), 0o644))

	require.NoError(t, fs.DeleteTaskDir(taskID))
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestFileStorage_MoveFileCrossFilesystemFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("fallback 测试仅在 Windows 下验证")
	}

	fs, _ := setupFileStorage(t)
	taskID := "fallback"
	require.NoError(t, fs.CreateTaskDir(taskID))

	dstPath := fs.GetOriginalFilePath(taskID)
	srcPath := createTempContentFile(t, t.TempDir(), []byte("fallback"))

	// 在 Windows 上保持文件句柄打开会导致 os.Rename 失败，迫使进入 fallback 逻辑。
	f, err := os.Open(srcPath)
	require.NoError(t, err)
	defer f.Close()

	err = fs.MoveFile(srcPath, dstPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove source file")
}
