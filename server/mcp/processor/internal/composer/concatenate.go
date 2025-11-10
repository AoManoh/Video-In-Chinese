package composer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"video-in-chinese/server/mcp/processor/internal/mediautil"

	"github.com/zeromicro/go-zero/core/logx"
)

// SegmentWithPath represents a segment with timing information (to avoid circular import)
type SegmentWithPath struct {
	Start float64
	End   float64
}

// concatItem represents an audio file with optional silence gap after it
type concatItem struct {
	AudioPath string
	GapAfter  time.Duration
}

// ConcatenateAudio concatenates audio segments in chronological order.
//
// This function sorts audio segments by start time and concatenates them
// using ffmpeg concat demuxer.
//
// Parameters:
//   - segments: list of audio segments to concatenate
//   - outputPath: path to save the concatenated audio
//
// Returns:
//   - error: error if concatenation fails
func (c *Composer) ConcatenateAudio(segments []AudioSegment, outputPath string) error {
	if len(segments) == 0 {
		logx.Error("[Composer] No audio segments to concatenate")
		return fmt.Errorf("no audio segments to concatenate")
	}

	// Sort segments by start time
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].StartTime < segments[j].StartTime
	})

	logx.Infof("[Composer] Concatenating %d audio segments", len(segments))

	// If only one segment, copy it directly
	if len(segments) == 1 {
		return c.copySingleSegment(segments[0].FilePath, outputPath)
	}

	// Create concat file list
	concatFilePath, err := c.createConcatFile(segments)
	if err != nil {
		return err
	}
	defer os.Remove(concatFilePath)

	// Run ffmpeg concat
	return c.runFFmpegConcat(concatFilePath, outputPath)
}

// copySingleSegment copies a single audio segment to the output path.
//
// Parameters:
//   - inputPath: path to the input audio file
//   - outputPath: path to save the output audio file
//
// Returns:
//   - error: error if copy fails
func (c *Composer) copySingleSegment(inputPath, outputPath string) error {
	// Use ffmpeg to copy (ensures format consistency)
	cmd := mediautil.NewFFmpegCommand(
		"-i", inputPath,
		"-c", "copy",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to copy single segment: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to copy single segment: %w", err)
	}

	logx.Infof("[Composer] Copied single segment to %s", outputPath)
	return nil
}

// createConcatFile creates a concat file list for ffmpeg.
//
// The concat file format:
//
//	file '/path/to/segment1.wav'
//	file '/path/to/segment2.wav'
//
// Parameters:
//   - segments: list of audio segments
//
// Returns:
//   - concatFilePath: path to the created concat file
//   - error: error if file creation fails
func (c *Composer) createConcatFile(segments []AudioSegment) (string, error) {
	// Create temp concat file
	concatFilePath := filepath.Join(os.TempDir(), "concat_list.txt")

	file, err := os.Create(concatFilePath)
	if err != nil {
		logx.Errorf("[Composer] Failed to create concat file: %v", err)
		return "", fmt.Errorf("failed to create concat file: %w", err)
	}
	defer file.Close()

	// Write file paths
	for _, segment := range segments {
		// Convert to absolute path
		absPath, err := filepath.Abs(segment.FilePath)
		if err != nil {
			logx.Errorf("[Composer] Failed to get absolute path for %s: %v", segment.FilePath, err)
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}

		// Write to concat file (use single quotes to handle spaces)
		_, err = fmt.Fprintf(file, "file '%s'\n", absPath)
		if err != nil {
			logx.Errorf("[Composer] Failed to write to concat file: %v", err)
			return "", fmt.Errorf("failed to write to concat file: %w", err)
		}
	}

	logx.Infof("[Composer] Created concat file: %s", concatFilePath)
	return concatFilePath, nil
}

// runFFmpegConcat runs ffmpeg concat demuxer.
//
// Parameters:
//   - concatFilePath: path to the concat file list
//   - outputPath: path to save the concatenated audio
//
// Returns:
//   - error: error if ffmpeg fails
func (c *Composer) runFFmpegConcat(concatFilePath, outputPath string) error {
	// ffmpeg -f concat -safe 0 -i concat_list.txt -c copy output.wav
	cmd := mediautil.NewFFmpegCommand(
		"-f", "concat",
		"-safe", "0",
		"-i", concatFilePath,
		"-c", "copy",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to concatenate audio: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to concatenate audio: %w", err)
	}

	logx.Infof("[Composer] Concatenated audio saved to %s", outputPath)
	return nil
}

// ConcatenateAudioWithTiming concatenates audio segments preserving original timing and gaps.
//
// This function is the core of Scheme B: each segment is already speed-adjusted to match
// its original duration. This function simply concatenates them with proper gaps inserted
// to preserve the original timing structure.
//
// Parameters:
//   - adjustedSegments: audio segments that have been speed-adjusted to match original duration
//   - originalSegments: original timing information (Start/End timestamps)
//   - outputPath: path to save the concatenated audio
//
// Returns:
//   - error: error if concatenation fails or invalid timing detected
func (c *Composer) ConcatenateAudioWithTiming(
	adjustedSegments []AudioSegment,
	originalSegments []SegmentWithPath,
	outputPath string,
) error {
	if len(adjustedSegments) == 0 {
		logx.Error("[Composer] No audio segments to concatenate")
		return fmt.Errorf("no audio segments to concatenate")
	}

	if len(adjustedSegments) != len(originalSegments) {
		return fmt.Errorf(
			"segment count mismatch: adjustedSegments=%d, originalSegments=%d",
			len(adjustedSegments), len(originalSegments),
		)
	}

	logx.Infof("[Composer] Concatenating %d segments with original timing", len(adjustedSegments))

	// Build concat list with gaps
	concatItems := make([]concatItem, 0, len(adjustedSegments))

	for i := range adjustedSegments {
		// Calculate gap until next segment
		var gap time.Duration
		if i < len(adjustedSegments)-1 {
			currentEnd := originalSegments[i].End
			nextStart := originalSegments[i+1].Start
			gapSeconds := nextStart - currentEnd

			// Validate: gap must be non-negative
			if gapSeconds < 0 {
				return fmt.Errorf(
					"invalid ASR timing: segment %d ends at %.3fs but segment %d starts at %.3fs (overlap of %.3fs). "+
					"This indicates ASR timestamp error",
					i, currentEnd, i+1, nextStart, -gapSeconds,
				)
			}

			gap = time.Duration(gapSeconds * float64(time.Second))
			logx.Infof("[Composer] Segment %d: gap after = %.3fs", i, gap.Seconds())
		}

		concatItems = append(concatItems, concatItem{
			AudioPath: adjustedSegments[i].FilePath,
			GapAfter:  gap,
		})
	}

	// If only one segment with no gap, just copy it
	if len(concatItems) == 1 && concatItems[0].GapAfter == 0 {
		return c.copySingleSegment(concatItems[0].AudioPath, outputPath)
	}

	// Create concat file with silence gaps
	return c.concatWithGaps(concatItems, outputPath)
}

// concatWithGaps concatenates audio files with silence gaps inserted between them
func (c *Composer) concatWithGaps(items []concatItem, outputPath string) error {
	// Create temporary directory for intermediate files
	tmpDir := filepath.Join(os.TempDir(), "audio_concat")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create concat list file
	concatListPath := filepath.Join(tmpDir, "concat_list.txt")
	file, err := os.Create(concatListPath)
	if err != nil {
		return fmt.Errorf("failed to create concat list: %w", err)
	}
	defer file.Close()

	// Write each audio file and gap (as silence)
	for i, item := range items {
		// Add audio file
		absPath, err := filepath.Abs(item.AudioPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", item.AudioPath, err)
		}
		fmt.Fprintf(file, "file '%s'\n", absPath)

		// Add silence gap if needed
		if item.GapAfter > 0 {
			silencePath := filepath.Join(tmpDir, fmt.Sprintf("silence_%d.wav", i))
			if err := c.generateSilence(item.GapAfter, silencePath); err != nil {
				return fmt.Errorf("failed to generate silence for segment %d: %w", i, err)
			}
			absSilencePath, _ := filepath.Abs(silencePath)
			fmt.Fprintf(file, "file '%s'\n", absSilencePath)
		}
	}

	file.Close()

	// Run FFmpeg concat
	return c.runFFmpegConcat(concatListPath, outputPath)
}

// generateSilence generates a silence audio file with the specified duration
func (c *Composer) generateSilence(duration time.Duration, outputPath string) error {
	durationSec := duration.Seconds()
	
	// ffmpeg -f lavfi -i anullsrc=r=44100:cl=stereo -t <duration> <output>
	cmd := mediautil.NewFFmpegCommand(
		"-f", "lavfi",
		"-i", "anullsrc=r=44100:cl=stereo",
		"-t", fmt.Sprintf("%.3f", durationSec),
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to generate silence: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to generate silence: %w", err)
	}

	logx.Infof("[Composer] Generated %.3fs silence at %s", durationSec, outputPath)
	return nil
}
