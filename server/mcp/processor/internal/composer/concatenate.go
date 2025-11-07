package composer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"video-in-chinese/server/mcp/processor/internal/mediautil"

	"github.com/zeromicro/go-zero/core/logx"
)

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
