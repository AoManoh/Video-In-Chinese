package mediautil

import (
	"fmt"
	"os/exec"
	
	"github.com/zeromicro/go-zero/core/logx"
)

// ExtractAudio extracts audio from a video file using ffmpeg.
//
// Parameters:
//   - videoPath: path to the input video file
//   - outputPath: path to save the extracted audio file
//
// Returns:
//   - error: error if extraction fails
func ExtractAudio(videoPath, outputPath string) error {
	// ffmpeg -i input.mp4 -vn -acodec pcm_s16le -ar 44100 -ac 2 output.wav
	// -vn: no video
	// -acodec pcm_s16le: PCM 16-bit little-endian
	// -ar 44100: sample rate 44.1kHz
	// -ac 2: stereo (2 channels)
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "44100",
		"-ac", "2",
		"-y",
		outputPath,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Mediautil] Failed to extract audio: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to extract audio: %w", err)
	}
	
	logx.Infof("[Mediautil] Extracted audio from %s to %s", videoPath, outputPath)
	return nil
}

