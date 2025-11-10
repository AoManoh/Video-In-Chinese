package mediautil

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// MinSpeedRatio is the minimum acceptable speed ratio (0.8x = 20% slower)
	MinSpeedRatio = 0.8

	// MaxSpeedRatio is the maximum acceptable speed ratio (1.2x = 20% faster)
	MaxSpeedRatio = 1.2
)

// AdjustSpeed adjusts the playback speed of an audio file using FFmpeg atempo filter.
//
// The atempo filter changes the speed without altering the pitch.
// Note: atempo supports range [0.5, 2.0], but we restrict to [0.8, 1.2] for quality.
//
// Parameters:
//   - inputPath: path to the input audio file
//   - speedRatio: target speed ratio (0.8 = slower, 1.2 = faster)
//   - outputPath: path to save the adjusted audio
//
// Returns:
//   - error: if speed ratio is out of range or ffmpeg fails
func AdjustSpeed(inputPath string, speedRatio float64, outputPath string) error {
	// Validate speed ratio
	if speedRatio < MinSpeedRatio || speedRatio > MaxSpeedRatio {
		return fmt.Errorf(
			"speed ratio %.3f out of acceptable range [%.2f, %.2f]",
			speedRatio, MinSpeedRatio, MaxSpeedRatio,
		)
	}

	// Special case: if ratio is very close to 1.0, just copy the file
	if speedRatio >= 0.99 && speedRatio <= 1.01 {
		logx.Infof("[AdjustSpeed] Speed ratio %.3f is close to 1.0, copying file directly", speedRatio)
		return CopyAudioFile(inputPath, outputPath)
	}

	logx.Infof("[AdjustSpeed] Adjusting speed: input=%s, ratio=%.3fx, output=%s", inputPath, speedRatio, outputPath)

	// Handle atempo filter limitation: supports [0.5, 2.0] only
	// For ratios outside this range, we need to chain multiple atempo filters
	// But our range [0.8, 1.2] is well within [0.5, 2.0], so single filter is enough
	cmd := NewFFmpegCommand(
		"-i", inputPath,
		"-filter:a", fmt.Sprintf("atempo=%.6f", speedRatio),
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[AdjustSpeed] FFmpeg failed: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to adjust speed: %w", err)
	}

	logx.Infof("[AdjustSpeed] Successfully adjusted speed to %.3fx", speedRatio)
	return nil
}

// CopyAudioFile copies an audio file using FFmpeg to ensure format consistency
func CopyAudioFile(inputPath, outputPath string) error {
	cmd := NewFFmpegCommand(
		"-i", inputPath,
		"-c", "copy",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[CopyAudioFile] FFmpeg copy failed: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to copy audio file: %w", err)
	}

	return nil
}

// VerifyAudioDuration verifies that the actual duration matches the expected duration
// within a tolerance threshold.
//
// Parameters:
//   - audioPath: path to the audio file
//   - expectedDuration: expected duration
//   - tolerance: acceptable deviation (default: 50ms)
//
// Returns:
//   - actualDuration: the actual duration of the audio file
//   - error: if actual duration deviates too much from expected
func VerifyAudioDuration(audioPath string, expectedDuration time.Duration, tolerance time.Duration) (time.Duration, error) {
	if tolerance == 0 {
		tolerance = 50 * time.Millisecond // Default tolerance: 50ms
	}

	actualDuration, err := GetAudioDuration(audioPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get audio duration: %w", err)
	}

	deviation := actualDuration - expectedDuration
	if deviation < 0 {
		deviation = -deviation
	}

	if deviation > tolerance {
		return actualDuration, fmt.Errorf(
			"duration mismatch: expected %.3fs, got %.3fs (deviation: %.3fs > tolerance: %.3fs)",
			expectedDuration.Seconds(),
			actualDuration.Seconds(),
			deviation.Seconds(),
			tolerance.Seconds(),
		)
	}

	return actualDuration, nil
}

// GetAudioDuration retrieves the duration of an audio file using ffprobe
func GetAudioDuration(audioPath string) (time.Duration, error) {
	// Use ffprobe to get audio duration
	// Note: Must use exec.Command directly, not NewFFmpegCommand (which calls ffmpeg, not ffprobe)
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[GetAudioDuration] ffprobe failed: %v, output: %s", err, string(output))
		return 0, fmt.Errorf("failed to get audio duration: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		logx.Errorf("[GetAudioDuration] failed to parse duration '%s': %v", durationStr, err)
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	duration := time.Duration(durationSec * float64(time.Second))
	return duration, nil
}
