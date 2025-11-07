package mediautil

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

// MergeVideoAudio merges a video file with a new audio track using ffmpeg.
//
// This function replaces the original audio track with the new audio track.
//
// Parameters:
//   - videoPath: path to the input video file
//   - audioPath: path to the new audio file
//   - outputPath: path to save the merged video file
//
// Returns:
//   - error: error if merge fails
func MergeVideoAudio(videoPath, audioPath, outputPath string) error {
	// ffmpeg -i input.mp4 -i new_audio.wav -c:v copy -c:a aac -map 0:v:0 -map 1:a:0 output.mp4
	// -c:v copy: copy video stream without re-encoding
	// -c:a aac: encode audio to AAC
	// -map 0:v:0: use video stream from first input
	// -map 1:a:0: use audio stream from second input
	cmd := NewFFmpegCommand(
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Mediautil] Failed to merge video and audio: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to merge video and audio: %w", err)
	}

	logx.Infof("[Mediautil] Merged video %s with audio %s to %s", videoPath, audioPath, outputPath)
	return nil
}
