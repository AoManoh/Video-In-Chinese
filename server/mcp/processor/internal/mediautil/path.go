package mediautil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// ResolveProjectPath 将可能以 data/ 开头的相对路径解析为项目根目录下的绝对路径。
// 对于已经是绝对路径的场景，直接返回；否则回退到当前工作目录进行解析。
func ResolveProjectPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path is empty")
	}

	if filepath.IsAbs(p) {
		return p, nil
	}

	normalized := filepath.ToSlash(p)
	if strings.HasPrefix(normalized, "data/") {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}

		projectRoot := filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
		absPath := filepath.Join(projectRoot, p)
		logx.Infof("[ResolveProjectPath] Converted relative path: %s -> %s", p, absPath)
		return absPath, nil
	}

	absPath, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}
