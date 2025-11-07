package mediautil

import (
	"os"
	"os/exec"

	"github.com/zeromicro/go-zero/core/logx"
)

// ffmpegBinary 保存要执行的 ffmpeg 可执行文件路径，默认从系统 PATH 查找。
var ffmpegBinary = "ffmpeg"

// SetFFmpegBinary 允许在运行时覆盖 ffmpeg 可执行文件的路径。
// 传入空字符串时会保持默认值（即依赖系统 PATH）。
func SetFFmpegBinary(path string) {
	if path == "" {
		logx.Infof("[Mediautil] Using ffmpeg from system PATH")
		return
	}

	if _, err := os.Stat(path); err != nil {
		logx.Errorf("[Mediautil] Provided ffmpeg path is invalid: %s, err: %v", path, err)
		return
	}

	logx.Infof("[Mediautil] Using custom ffmpeg binary: %s", path)
	ffmpegBinary = path
}

// NewFFmpegCommand 创建一个使用当前 ffmpegBinary 的命令。
func NewFFmpegCommand(args ...string) *exec.Cmd {
	return exec.Command(ffmpegBinary, args...)
}
