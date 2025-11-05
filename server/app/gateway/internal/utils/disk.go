//go:build !windows
// +build !windows

package utils

import (
	"fmt"
	"syscall"
)

// CheckDiskSpace checks if there is enough available disk space at the given path
// Returns error if available space is less than requiredBytes
func CheckDiskSpace(path string, requiredBytes int64) error {
	availableBytes, err := GetAvailableDiskSpace(path)
	if err != nil {
		return fmt.Errorf("failed to get disk space: %w", err)
	}

	if availableBytes < requiredBytes {
		return fmt.Errorf("insufficient disk space: required %d bytes, available %d bytes", requiredBytes, availableBytes)
	}

	return nil
}

// GetAvailableDiskSpace returns the available disk space in bytes at the given path
func GetAvailableDiskSpace(path string) (int64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, fmt.Errorf("syscall.Statfs failed: %w", err)
	}

	// Available blocks * block size
	availableBytes := int64(stat.Bavail) * int64(stat.Bsize)
	return availableBytes, nil
}
