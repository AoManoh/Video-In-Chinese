package composer

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	
	"video-in-chinese/server/mcp/processor/internal/storage"
	
	"github.com/zeromicro/go-zero/core/logx"
)

// AudioSegment represents an audio segment with start time and file path.
//
// This structure is used for concatenating audio segments in chronological order.
type AudioSegment struct {
	StartTime time.Duration // Start time in the original audio
	FilePath  string        // Path to the audio segment file
}

// Composer provides audio composition operations.
//
// This includes concatenating audio segments, aligning audio duration,
// and merging vocals with background music.
type Composer struct {
	pathManager *storage.PathManager
}

// NewComposer creates a new Composer instance.
//
// Parameters:
//   - pathManager: path manager for file path generation
//
// Returns:
//   - *Composer: initialized composer
func NewComposer(pathManager *storage.PathManager) *Composer {
	return &Composer{
		pathManager: pathManager,
	}
}

// GetAudioDuration retrieves the duration of an audio file using ffprobe.
//
// Parameters:
//   - audioPath: path to the audio file
//
// Returns:
//   - duration: audio duration
//   - error: error if ffprobe fails or file doesn't exist
func GetAudioDuration(audioPath string) (time.Duration, error) {
	// ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 <file>
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[Composer] Failed to get audio duration for %s: %v, output: %s", audioPath, err, string(output))
		return 0, fmt.Errorf("failed to get audio duration: %w", err)
	}
	
	durationStr := strings.TrimSpace(string(output))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		logx.Errorf("[Composer] Failed to parse duration %s: %v", durationStr, err)
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	
	duration := time.Duration(durationSec * float64(time.Second))
	logx.Infof("[Composer] Audio duration for %s: %v", audioPath, duration)
	return duration, nil
}

