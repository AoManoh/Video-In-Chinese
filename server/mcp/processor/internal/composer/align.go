package composer

import (
	"fmt"
	"math"
	"time"

	"video-in-chinese/server/mcp/processor/internal/mediautil"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// SilencePaddingThreshold is the maximum time difference for silence padding (500ms)
	SilencePaddingThreshold = 500 * time.Millisecond

	// MinSpeedRatio is the minimum speed ratio for audio speed adjustment (0.9x)
	MinSpeedRatio = 0.9

	// MaxSpeedRatio is the maximum speed ratio for audio speed adjustment (1.1x)
	MaxSpeedRatio = 1.1
)

// AlignAudio aligns the duration of translated audio to match the original audio.
//
// Strategy:
//  1. If time difference <= 500ms: use silence padding
//  2. If time difference > 500ms: use speed adjustment (0.9x - 1.1x)
//  3. If speed ratio out of range: return error
//
// Parameters:
//   - translatedAudioPath: path to the translated audio file
//   - originalAudioPath: path to the original audio file
//   - outputPath: path to save the aligned audio
//
// Returns:
//   - error: error if alignment fails or speed ratio out of range
func (c *Composer) AlignAudio(translatedAudioPath, originalAudioPath, outputPath string) error {
	// Get audio durations
	translatedDuration, err := GetAudioDuration(translatedAudioPath)
	if err != nil {
		return fmt.Errorf("failed to get translated audio duration: %w", err)
	}

	originalDuration, err := GetAudioDuration(originalAudioPath)
	if err != nil {
		return fmt.Errorf("failed to get original audio duration: %w", err)
	}

	timeDiff := originalDuration - translatedDuration
	logx.Infof("[Composer] Time difference: %v (original: %v, translated: %v)", timeDiff, originalDuration, translatedDuration)

	// Check if durations are already aligned (within threshold)
	if math.Abs(float64(timeDiff)) <= float64(SilencePaddingThreshold) {
		if timeDiff >= 0 {
			// translated shorter: pad a little silence
			return c.padSilence(translatedAudioPath, timeDiff, outputPath)
		}
		// translated slightly longer: keep as-is (no trimming) to avoid unnecessary failure
		return c.copySingleSegment(translatedAudioPath, outputPath)
	}

	// Use speed adjustment
	speedRatio := float64(originalDuration) / float64(translatedDuration)
	if speedRatio < MinSpeedRatio || speedRatio > MaxSpeedRatio {
		logx.Errorf("[Composer] Speed ratio %.2f out of range [%.2f, %.2f]", speedRatio, MinSpeedRatio, MaxSpeedRatio)
		return fmt.Errorf("speed ratio %.2f out of range [%.2f, %.2f]", speedRatio, MinSpeedRatio, MaxSpeedRatio)
	}

	return c.adjustSpeed(translatedAudioPath, speedRatio, outputPath)
}

// padSilence pads silence to the audio to match the target duration.
//
// If timeDiff > 0: add silence at the end
// If timeDiff < 0: trim audio (not implemented, return error)
//
// Parameters:
//   - audioPath: path to the audio file
//   - timeDiff: time difference to pad (positive: add silence, negative: trim)
//   - outputPath: path to save the padded audio
//
// Returns:
//   - error: error if padding fails
func (c *Composer) padSilence(audioPath string, timeDiff time.Duration, outputPath string) error {
	if timeDiff < 0 {
		// Trimming not implemented, use speed adjustment instead
		logx.Infof("[Composer] Negative time difference %v, should use speed adjustment", timeDiff)
		return fmt.Errorf("negative time difference %v, trimming not supported", timeDiff)
	}

	if timeDiff == 0 {
		// No padding needed, copy directly
		return c.copySingleSegment(audioPath, outputPath)
	}

	// Add silence at the end
	silenceDurationSec := float64(timeDiff) / float64(time.Second)

	// ffmpeg -i input.wav -f lavfi -i anullsrc=r=44100:cl=stereo -filter_complex "[0:a][1:a]concat=n=2:v=0:a=1[out]" -map "[out]" -t <total_duration> output.wav
	// Simplified: ffmpeg -i input.wav -af "apad=pad_dur=<silence_duration>" output.wav
	cmd := mediautil.NewFFmpegCommand(
		"-i", audioPath,
		"-af", fmt.Sprintf("apad=pad_dur=%.3f", silenceDurationSec),
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to pad silence: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to pad silence: %w", err)
	}

	logx.Infof("[Composer] Padded %.3fs silence to %s", silenceDurationSec, outputPath)
	return nil
}

// adjustSpeed adjusts the speed of the audio to match the target duration.
//
// Parameters:
//   - audioPath: path to the audio file
//   - speedRatio: speed ratio (e.g., 1.1 means 10% faster)
//   - outputPath: path to save the adjusted audio
//
// Returns:
//   - error: error if speed adjustment fails
func (c *Composer) adjustSpeed(audioPath string, speedRatio float64, outputPath string) error {
	// ffmpeg -i input.wav -filter:a "atempo=<speed_ratio>" output.wav
	// Note: atempo filter only supports 0.5-2.0 range, which is sufficient for our 0.9-1.1 range
	cmd := mediautil.NewFFmpegCommand(
		"-i", audioPath,
		"-filter:a", fmt.Sprintf("atempo=%.3f", speedRatio),
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to adjust speed: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to adjust speed: %w", err)
	}

	logx.Infof("[Composer] Adjusted speed to %.2fx, saved to %s", speedRatio, outputPath)
	return nil
}
