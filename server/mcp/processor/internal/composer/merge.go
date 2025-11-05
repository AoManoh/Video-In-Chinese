package composer

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/zeromicro/go-zero/core/logx"
)

// MergeAudio merges vocals with background music.
//
// If background music exists, merge vocals and background using ffmpeg amix filter.
// If background music doesn't exist, copy vocals directly to output.
//
// Parameters:
//   - vocalsPath: path to the vocals audio file
//   - backgroundPath: path to the background music file (can be empty)
//   - outputPath: path to save the merged audio
//
// Returns:
//   - error: error if merge fails
func (c *Composer) MergeAudio(vocalsPath, backgroundPath, outputPath string) error {
	// Check if vocals file exists
	if _, err := os.Stat(vocalsPath); os.IsNotExist(err) {
		logx.Errorf("[Composer] Vocals file not found: %s", vocalsPath)
		return fmt.Errorf("vocals file not found: %s", vocalsPath)
	}

	// Check if background music exists
	if backgroundPath == "" {
		logx.Infof("[Composer] No background music, copying vocals directly")
		return c.copySingleSegment(vocalsPath, outputPath)
	}

	if _, err := os.Stat(backgroundPath); os.IsNotExist(err) {
		logx.Infof("[Composer] Background music file not found: %s, copying vocals directly", backgroundPath)
		return c.copySingleSegment(vocalsPath, outputPath)
	}

	// Merge vocals and background music
	return c.mergeWithBackground(vocalsPath, backgroundPath, outputPath)
}

// mergeWithBackground merges vocals with background music using ffmpeg amix filter.
//
// Parameters:
//   - vocalsPath: path to the vocals audio file
//   - backgroundPath: path to the background music file
//   - outputPath: path to save the merged audio
//
// Returns:
//   - error: error if merge fails
func (c *Composer) mergeWithBackground(vocalsPath, backgroundPath, outputPath string) error {
	// ffmpeg -i vocals.wav -i background.wav -filter_complex "[0:a][1:a]amix=inputs=2:duration=first:dropout_transition=2" output.wav
	// duration=first: use the duration of the first input (vocals)
	// dropout_transition=2: smooth transition when one input ends
	cmd := exec.Command("ffmpeg",
		"-i", vocalsPath,
		"-i", backgroundPath,
		"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=first:dropout_transition=2",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to merge audio: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to merge audio: %w", err)
	}

	logx.Infof("[Composer] Merged vocals and background music to %s", outputPath)
	return nil
}
