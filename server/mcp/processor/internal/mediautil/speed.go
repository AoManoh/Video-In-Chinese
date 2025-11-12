package mediautil

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// MinSpeedRatio is the minimum acceptable speed ratio (0.35x ≈ 65% slower)
	// Still conservative enough to avoid unnatural speech after multi-stage atempo processing.
	MinSpeedRatio = 0.35

	// MaxSpeedRatio is the maximum acceptable speed ratio (2.5x ≈ 150% faster)
	// Ratios beyond this tend to introduce artifacts even with chained filters.
	MaxSpeedRatio = 2.5

	speedRatioTolerance = 0.02

	atempoMinRatio = 0.5
	atempoMaxRatio = 2.0
)

// buildAtempoStages decomposes a target ratio into multiple ffmpeg atempo stages,
// each of which must remain within [0.5, 2.0]. This lets us support ratios outside
// the native single-filter range by chaining filters (e.g., 0.5 × 0.9 = 0.45).
func buildAtempoStages(target float64) []float64 {
	remaining := target
	stages := make([]float64, 0, 4)

	for remaining < atempoMinRatio {
		stages = append(stages, atempoMinRatio)
		remaining = remaining / atempoMinRatio
	}

	for remaining > atempoMaxRatio {
		stages = append(stages, atempoMaxRatio)
		remaining = remaining / atempoMaxRatio
	}

	stages = append(stages, remaining)
	return stages
}

// AdjustSpeed adjusts the playback speed of an audio file using FFmpeg atempo filter.
//
// The atempo filter changes the speed without altering the pitch.
// We chain multiple atempo filters (each within [0.5, 2.0]) so that the overall
// ratio can safely cover [0.35, 2.5] without re-synthesizing audio upstream.
//
// Parameters:
//   - inputPath: path to the input audio file
//   - speedRatio: target speed ratio (0.8 = slower, 1.2 = faster)
//   - outputPath: path to save the adjusted audio
//
// Returns:
//   - error: if speed ratio is out of range or ffmpeg fails
func AdjustSpeed(inputPath string, speedRatio float64, outputPath string) error {
	clampedRatio := speedRatio
	// Validate speed ratio with tolerance to absorb floating point errors
	if clampedRatio < MinSpeedRatio-speedRatioTolerance || clampedRatio > MaxSpeedRatio+speedRatioTolerance {
		return fmt.Errorf(
			"speed ratio %.3f out of acceptable range [%.2f, %.2f]",
			speedRatio, MinSpeedRatio, MaxSpeedRatio,
		)
	}

	absInputPath, err := ResolveProjectPath(inputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve input path: %w", err)
	}

	if clampedRatio < MinSpeedRatio {
		logx.Infof("[AdjustSpeed] Speed ratio %.3f below minimum, clamping to %.3f", clampedRatio, MinSpeedRatio)
		clampedRatio = MinSpeedRatio
	}
	if clampedRatio > MaxSpeedRatio {
		logx.Infof("[AdjustSpeed] Speed ratio %.3f above maximum, clamping to %.3f", clampedRatio, MaxSpeedRatio)
		clampedRatio = MaxSpeedRatio
	}

	// Special case: if ratio is very close to 1.0, just copy the file
	if clampedRatio >= 0.99 && clampedRatio <= 1.01 {
		logx.Infof("[AdjustSpeed] Speed ratio %.3f is close to 1.0, copying file directly", clampedRatio)
		return CopyAudioFile(absInputPath, outputPath)
	}

	stages := buildAtempoStages(clampedRatio)
	filterExpr := make([]string, len(stages))
	for i, stage := range stages {
		filterExpr[i] = fmt.Sprintf("atempo=%.6f", stage)
	}
	logx.Infof("[AdjustSpeed] Adjusting speed: input=%s, ratio=%.3fx, stages=%v, output=%s", absInputPath, clampedRatio, stages, outputPath)

	cmd := NewFFmpegCommand(
		"-i", absInputPath,
		"-filter:a", strings.Join(filterExpr, ","),
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[AdjustSpeed] FFmpeg failed: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to adjust speed: %w", err)
	}

	logx.Infof("[AdjustSpeed] Successfully adjusted speed to %.3fx", clampedRatio)
	return nil
}

// CopyAudioFile copies an audio file using FFmpeg to ensure format consistency
func CopyAudioFile(inputPath, outputPath string) error {
	absInputPath, err := ResolveProjectPath(inputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve input path: %w", err)
	}

	cmd := NewFFmpegCommand(
		"-i", absInputPath,
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
	absPath, err := ResolveProjectPath(audioPath)
	if err != nil {
		logx.Errorf("[GetAudioDuration] Failed to resolve audio path %s: %v", audioPath, err)
		return 0, fmt.Errorf("failed to resolve audio path: %w", err)
	}

	// Use ffprobe to get audio duration
	// Note: Must use exec.Command directly, not NewFFmpegCommand (which calls ffmpeg, not ffprobe)
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		absPath,
	)

	// Ensure UTF-8 output on Windows
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "LANG=en_US.UTF-8")

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
