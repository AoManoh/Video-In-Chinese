package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsPathSafe checks if targetPath is safe to access within basePath
// Returns error if targetPath attempts path traversal or symlink attacks
func IsPathSafe(basePath, targetPath string) error {
	// Clean and resolve absolute paths
	cleanBase, err := filepath.Abs(filepath.Clean(basePath))
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	cleanTarget, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Check if target is within base directory
	if !strings.HasPrefix(cleanTarget, cleanBase) {
		return fmt.Errorf("path traversal detected: target path is outside base directory")
	}

	// Check for symlink attacks
	// Evaluate symlinks in the target path
	evalTarget, err := filepath.EvalSymlinks(cleanTarget)
	if err != nil {
		// If file doesn't exist yet, that's okay
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to evaluate symlinks: %w", err)
		}
		// File doesn't exist, check parent directory
		parentDir := filepath.Dir(cleanTarget)
		evalParent, err := filepath.EvalSymlinks(parentDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to evaluate parent symlinks: %w", err)
		}
		if evalParent != "" && !strings.HasPrefix(evalParent, cleanBase) {
			return fmt.Errorf("symlink attack detected: parent directory points outside base")
		}
	} else {
		// File exists, check if evaluated path is still within base
		if !strings.HasPrefix(evalTarget, cleanBase) {
			return fmt.Errorf("symlink attack detected: target resolves outside base directory")
		}
	}

	return nil
}

// SafeJoinPath safely joins basePath and relativePath, checking for path traversal
func SafeJoinPath(basePath, relativePath string) (string, error) {
	// Join paths
	targetPath := filepath.Join(basePath, relativePath)

	// Check if safe
	if err := IsPathSafe(basePath, targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

