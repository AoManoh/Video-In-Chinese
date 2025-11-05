//go:build windows
// +build windows

package utils

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpace = kernel32.NewProc("GetDiskFreeSpaceExW")
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
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("failed to convert path to UTF16: %w", err)
	}

	ret, _, err := getDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if ret == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx failed: %w", err)
	}

	return freeBytesAvailable, nil
}

