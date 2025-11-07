package mediautil

import (
"fmt"

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
// ffmpeg -i input.mp4 -vn -acodec pcm_s16le -ar 16000 -ac 1 -af volume=20dB output.wav
// -vn: no video
// -acodec pcm_s16le: PCM 16-bit little-endian
// -ar 16000: sample rate 16kHz (recommended for ASR)
// -ac 1: mono (1 channel, recommended for ASR)
// -af volume=20dB: boost volume by 20dB for better ASR recognition
cmd := NewFFmpegCommand(
"-i", videoPath,
"-vn",
"-acodec", "pcm_s16le",
"-ar", "16000",
"-ac", "1",
"-af", "volume=20dB",
"-y",
outputPath,
)

output, err := cmd.CombinedOutput()
if err != nil {
logx.Errorf("[Mediautil] Failed to extract audio: %v, output: %s", err, string(output))
return fmt.Errorf("failed to extract audio: %w", err)
}

logx.Infof("[Mediautil] Extracted audio from %s to %s (16kHz mono, +20dB boost for ASR)", videoPath, outputPath)
return nil
}
