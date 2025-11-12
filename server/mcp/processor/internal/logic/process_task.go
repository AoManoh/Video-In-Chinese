package logic

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"unicode/utf8"

	"video-in-chinese/server/mcp/processor/internal/composer"
	"video-in-chinese/server/mcp/processor/internal/mediautil"
	"video-in-chinese/server/mcp/processor/internal/svc"
	"video-in-chinese/server/mcp/processor/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

// SegmentWithPath represents a text segment with its audio file path.
type SegmentWithPath struct {
	SpeakerId        string
	Text             string
	TranslatedText   string
	Start            float64
	End              float64
	AudioSegmentPath string
}

const (
	targetWordsPerSecond = 3.6
	targetWordTolerance  = 0.15
)

// processTask processes a single task following the 18-step workflow.
func processTask(ctx context.Context, svcCtx *svc.ServiceContext, task *TaskMessage) {
	taskID := task.TaskID
	logx.Infof("[ProcessTask] Starting task: %s", taskID)

	// Step 1: Update status to PROCESSING
	if err := svcCtx.RedisClient.UpdateTaskStatus(ctx, taskID, "PROCESSING"); err != nil {
		logx.Errorf("[ProcessTask] Failed to update status to PROCESSING: %v", err)
		return
	}

	// Defer error handling
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("Panic: %v", r)
			logx.Errorf("[ProcessTask] Task %s panicked: %s", taskID, errMsg)
			_ = svcCtx.RedisClient.UpdateTaskStatus(ctx, taskID, "FAILED")
			_ = svcCtx.RedisClient.UpdateTaskError(ctx, taskID, errMsg)
		}
	}()

	// Execute workflow
	if err := executeWorkflow(ctx, svcCtx, task); err != nil {
		logx.Errorf("[ProcessTask] Task %s failed: %v", taskID, err)
		_ = svcCtx.RedisClient.UpdateTaskStatus(ctx, taskID, "FAILED")
		_ = svcCtx.RedisClient.UpdateTaskError(ctx, taskID, err.Error())
		return
	}

	// Update status to COMPLETED
	if err := svcCtx.RedisClient.UpdateTaskStatus(ctx, taskID, "COMPLETED"); err != nil {
		logx.Errorf("[ProcessTask] Failed to update status to COMPLETED: %v", err)
		return
	}

	// Cleanup intermediate files after successful task completion
	logx.Infof("[ProcessTask] Cleaning up intermediate files for task: %s", taskID)
	if err := svcCtx.PathManager.CleanupIntermediateFiles(taskID); err != nil {
		// Cleanup failure only logs an error and does not affect task status
		logx.Errorf("[ProcessTask] Failed to cleanup intermediate files: %v", err)
	}

	logx.Infof("[ProcessTask] Task %s completed successfully", taskID)
}

// executeWorkflow executes the 18-step processing workflow.
func executeWorkflow(ctx context.Context, svcCtx *svc.ServiceContext, task *TaskMessage) error {
	taskID := task.TaskID
	originalVideoPath := task.OriginalFilePath

	// Load app settings (from Redis) and derive runtime flags/params
	settings, err := svcCtx.RedisClient.GetAppSettings(ctx)
	if err != nil {
		logx.Infof("[ProcessTask] App settings unavailable, using defaults: %v", err)
	}
	getBool := func(key string, def bool) bool {
		if v, ok := settings[key]; ok {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
		return def
	}
	getStr := func(key, def string) string {
		if v, ok := settings[key]; ok && v != "" {
			return v
		}
		return def
	}
	// Feature toggles
	audioSeparationEnabled := getBool("audio_separation_enabled", false)
	textPolishEnabled := getBool("polishing_enabled", false)
	translationOptimizeEnabled := getBool("optimization_enabled", false)
	// Workflow parameters
	sourceLang := getStr("source_lang", "en")
	targetLang := getStr("target_lang", "zh")
	videoType := getStr("translation_video_type", "general")
	customPrompt := getStr("polishing_custom_prompt", "")

	// Ensure intermediate directory exists
	if err := svcCtx.PathManager.EnsureIntermediateDir(taskID); err != nil {
		return fmt.Errorf("failed to create intermediate directory: %w", err)
	}

	// Step 2: Extract audio from video
	logx.Infof("[ProcessTask] Step 2: Extracting audio from video")
	originalAudioPath := svcCtx.PathManager.GetIntermediatePath(taskID, "original_audio.wav")
	if err := mediautil.ExtractAudio(originalVideoPath, originalAudioPath); err != nil {
		return fmt.Errorf("step 2 failed (extract audio): %w", err)
	}

	// Step 3: (Optional) Separate audio into vocals and background
	var vocalsPath, backgroundPath string

	if audioSeparationEnabled && svcCtx.AudioSeparatorClient != nil {
		logx.Infof("[ProcessTask] Step 3: Separating audio (vocals + background)")

		// Call AudioSeparator gRPC service
		req := &pb.SeparateAudioRequest{
			AudioPath: originalAudioPath,
			OutputDir: svcCtx.PathManager.GetIntermediateDir(taskID),
			Stems:     2, // 2 stems: vocals + accompaniment
		}

		resp, err := svcCtx.AudioSeparatorClient.SeparateAudio(ctx, req)
		if err != nil {
			return fmt.Errorf("step 3 failed (separate audio): %w", err)
		}

		if !resp.Success {
			return fmt.Errorf("step 3 failed (separate audio): %s", resp.ErrorMessage)
		}

		vocalsPath = resp.VocalsPath
		backgroundPath = resp.AccompanimentPath
	} else if audioSeparationEnabled {
		logx.Infof("[ProcessTask] Audio separation enabled but client unavailable; skipping separation")
		vocalsPath = originalAudioPath
		backgroundPath = ""
	} else {
		logx.Infof("[ProcessTask] Step 3: Skipping audio separation (disabled)")
		vocalsPath = originalAudioPath
		backgroundPath = ""
	}

	// Step 4: ASR (Automatic Speech Recognition) with speaker diarization
	logx.Infof("[ProcessTask] Step 4: Running ASR with speaker diarization")
	asrReq := &pb.ASRRequest{
		AudioPath: vocalsPath,
	}

	asrResp, err := svcCtx.AIAdaptorClient.ASR(ctx, asrReq)
	if err != nil {
		return fmt.Errorf("step 4 failed (ASR): %w", err)
	}

	// ASR returns Speakers list (nested structure: Speaker -> Sentences)
	speakers := asrResp.Speakers
	logx.Infof("[ProcessTask] ASR returned %d speakers", len(speakers))

	// Step 7.5: Cut audio segments based on ASR timestamps (Processor's responsibility)
	logx.Infof("[ProcessTask] Step 7.5: Cutting audio segments")
	allSegments := make([]SegmentWithPath, 0)

	for _, speaker := range speakers {
		for i, sentence := range speaker.Sentences {
			// Generate segment file path
			segmentPath := svcCtx.PathManager.GetIntermediatePath(
				taskID,
				fmt.Sprintf("speaker_%s_segment_%d.wav", speaker.SpeakerId, i),
			)

			// Cut audio segment using ffmpeg
			if err := cutAudioSegment(vocalsPath, segmentPath, sentence.StartTime, sentence.EndTime); err != nil {
				return fmt.Errorf("step 7.5 failed (cut segment %d for speaker %s): %w", i, speaker.SpeakerId, err)
			}

			allSegments = append(allSegments, SegmentWithPath{
				SpeakerId:        speaker.SpeakerId,
				Text:             sentence.Text,
				Start:            sentence.StartTime,
				End:              sentence.EndTime,
				AudioSegmentPath: segmentPath,
			})
		}
	}

	logx.Infof("[ProcessTask] Cut %d audio segments", len(allSegments))

	// Step 5: (Optional) Polish text

	if textPolishEnabled {
		logx.Infof("[ProcessTask] Step 5: Polishing text")
		for i := range allSegments {
			polishReq := &pb.PolishRequest{
				Text:         allSegments[i].Text,
				VideoType:    videoType,
				CustomPrompt: customPrompt,
			}

			polishResp, err := svcCtx.AIAdaptorClient.Polish(ctx, polishReq)
			if err != nil {
				return fmt.Errorf("step 5 failed (polish segment %d): %w", i, err)
			}

			allSegments[i].Text = polishResp.PolishedText
		}
	} else {
		logx.Infof("[ProcessTask] Step 5: Skipping text polish (disabled)")
	}

	// Step 6: Translate text
	logx.Infof("[ProcessTask] Step 6: Translating text")
	for i := range allSegments {
		durationSeconds := segmentDurationSeconds(allSegments[i])
		minWords, maxWords := calculateWordBounds(durationSeconds)
		translateReq := &pb.TranslateRequest{
			Text:            allSegments[i].Text,
			SourceLang:      sourceLang,
			TargetLang:      targetLang,
			VideoType:       videoType,
			DurationSeconds: durationSeconds,
			SpeakerRole:     formatSpeakerRole(allSegments[i].SpeakerId),
			TargetWordMin:   minWords,
			TargetWordMax:   maxWords,
		}

		translateResp, err := svcCtx.AIAdaptorClient.Translate(ctx, translateReq)
		if err != nil {
			return fmt.Errorf("step 6 failed (translate segment %d): %w", i, err)
		}

		allSegments[i].TranslatedText = translateResp.TranslatedText
	}

	// Step 7: (Optional) Optimize translation

	if translationOptimizeEnabled {
		logx.Infof("[ProcessTask] Step 7: Optimizing translation")
		for i := range allSegments {
			durationSeconds := segmentDurationSeconds(allSegments[i])
			minWords, maxWords := calculateWordBounds(durationSeconds)
			optimizeReq := &pb.OptimizeRequest{
				Text:                  allSegments[i].TranslatedText,
				TargetDurationSeconds: durationSeconds,
				TargetWordMin:         minWords,
				TargetWordMax:         maxWords,
				SpeakerRole:           formatSpeakerRole(allSegments[i].SpeakerId),
				VideoType:             videoType,
			}

			optimizeResp, err := svcCtx.AIAdaptorClient.Optimize(ctx, optimizeReq)
			if err != nil {
				return fmt.Errorf("step 7 failed (optimize segment %d): %w", i, err)
			}

			allSegments[i].TranslatedText = optimizeResp.OptimizedText
		}
	} else {
		logx.Infof("[ProcessTask] Step 7: Skipping translation optimization (disabled)")
	}

	if err := enforceTranslationLength(ctx, svcCtx, allSegments, videoType); err != nil {
		return err
	}

	// Step 7.8: Merge audio segments per speaker to create reference audio (10-15s)
	logx.Infof("[ProcessTask] Step 7.8: Merging audio segments per speaker for voice cloning reference")
	speakerReferenceAudio, err := mergeSegmentsPerSpeaker(svcCtx, taskID, allSegments)
	if err != nil {
		return fmt.Errorf("step 7.8 failed (merge segments per speaker): %w", err)
	}
	logx.Infof("[ProcessTask] Created reference audio for %d speakers", len(speakerReferenceAudio))

	// Continue workflow: steps 8-13
	return continueWorkflow(ctx, svcCtx, taskID, originalVideoPath, allSegments, speakerReferenceAudio, originalAudioPath, backgroundPath)
}

// continueWorkflow continues the workflow from step 8 to step 13.
func continueWorkflow(ctx context.Context, svcCtx *svc.ServiceContext, taskID, originalVideoPath string,
	allSegments []SegmentWithPath, speakerReferenceAudio map[string]string, originalAudioPath, backgroundPath string) error {

	// Step 8: Clone voice for each speaker
	logx.Infof("[ProcessTask] Step 8: Cloning voice for each speaker")
	clonedSegments := make([]composer.AudioSegment, 0, len(allSegments))

	for i, seg := range allSegments {
		// 使用该说话人的合并参考音频（10-15秒）
		referenceAudio, exists := speakerReferenceAudio[seg.SpeakerId]
		if !exists {
			return fmt.Errorf("step 8 failed: no reference audio found for speaker %s", seg.SpeakerId)
		}

		cloneReq := &pb.CloneVoiceRequest{
			SpeakerId:      seg.SpeakerId,
			Text:           seg.TranslatedText,
			ReferenceAudio: referenceAudio, // 使用合并后的参考音频
		}

		cloneResp, err := svcCtx.AIAdaptorClient.CloneVoice(ctx, cloneReq)
		if err != nil {
			return fmt.Errorf("step 8 failed (clone voice segment %d): %w", i, err)
		}

		resolvedPath, err := mediautil.ResolveProjectPath(cloneResp.AudioPath)
		if err != nil {
			return fmt.Errorf("step 8 failed (resolve cloned audio path for segment %d): %w", i, err)
		}

		// Store cloned audio path temporarily (will be adjusted in Step 8.5)
		clonedSegments = append(clonedSegments, composer.AudioSegment{
			StartTime: time.Duration(seg.Start * float64(time.Second)),
			FilePath:  resolvedPath,
		})
	}

	// Step 8.5: Adjust speed for each segment to match original duration
	logx.Infof("[ProcessTask] Step 8.5: Adjusting speed for each segment to match original duration")

	for i := range clonedSegments {
		// Calculate original duration (from ASR timestamps)
		originalDuration := time.Duration((allSegments[i].End - allSegments[i].Start) * float64(time.Second))

		// Get cloned audio duration
		clonedDuration, err := mediautil.GetAudioDuration(clonedSegments[i].FilePath)
		if err != nil {
			return fmt.Errorf("step 8.5 failed (get duration for segment %d): %w", i, err)
		}

		// Calculate speed ratio
		speedRatio := float64(originalDuration) / float64(clonedDuration)

		logx.Infof("[ProcessTask] Segment %d: original=%.3fs, cloned=%.3fs, ratio=%.3fx",
			i, originalDuration.Seconds(), clonedDuration.Seconds(), speedRatio)

		// Check if speed ratio is within acceptable range
		if speedRatio < mediautil.MinSpeedRatio || speedRatio > mediautil.MaxSpeedRatio {
			return fmt.Errorf(
				"step 8.5 failed: segment %d requires speed ratio %.3fx (out of range [%.2f, %.2f]). "+
					"Original duration: %.3fs, Cloned duration: %.3fs. "+
					"Suggestion: optimize translation length or voice cloning parameters",
				i, speedRatio, mediautil.MinSpeedRatio, mediautil.MaxSpeedRatio,
				originalDuration.Seconds(), clonedDuration.Seconds(),
			)
		}

		// Adjust speed to match original duration
		adjustedPath := svcCtx.PathManager.GetIntermediatePath(
			taskID,
			fmt.Sprintf("adjusted_segment_%d.wav", i),
		)

		if err := mediautil.AdjustSpeed(clonedSegments[i].FilePath, speedRatio, adjustedPath); err != nil {
			return fmt.Errorf("step 8.5 failed (adjust speed for segment %d): %w", i, err)
		}

		// Verify adjusted duration
		verifiedDuration, err := mediautil.VerifyAudioDuration(adjustedPath, originalDuration, 50*time.Millisecond)
		if err != nil {
			logx.Infof("[ProcessTask] Segment %d duration verification warning: %v", i, err)
			// Continue anyway, small deviation is acceptable
		} else {
			logx.Infof("[ProcessTask] Segment %d adjusted successfully: %.3fs (verified)", i, verifiedDuration.Seconds())
		}

		// Update segment path to adjusted version
		clonedSegments[i].FilePath = adjustedPath
	}

	// Step 9: Concatenate audio segments (with original timestamps and gaps)
	logx.Infof("[ProcessTask] Step 9: Concatenating audio segments with original timing")
	concatenatedPath := svcCtx.PathManager.GetIntermediatePath(taskID, "concatenated.wav")

	composerInstance := composer.NewComposer(svcCtx.PathManager)

	// Convert allSegments to composer.SegmentWithPath (only timing info needed)
	originalTimings := make([]composer.SegmentWithPath, len(allSegments))
	for i, seg := range allSegments {
		originalTimings[i] = composer.SegmentWithPath{
			Start: seg.Start,
			End:   seg.End,
		}
	}

	// Pass original segments for timing information
	if err := composerInstance.ConcatenateAudioWithTiming(clonedSegments, originalTimings, concatenatedPath); err != nil {
		return fmt.Errorf("step 9 failed (concatenate audio): %w", err)
	}

	// Step 10 (AlignAudio) is now REMOVED - no longer needed as segments are already aligned
	// The concatenated audio already matches the original timing perfectly

	// Use concatenated audio directly for merging with background
	alignedPath := concatenatedPath

	// Step 11: Merge vocals with background music
	logx.Infof("[ProcessTask] Step 11: Merging vocals with background music")
	mergedAudioPath := svcCtx.PathManager.GetIntermediatePath(taskID, "merged_audio.wav")

	if err := composerInstance.MergeAudio(alignedPath, backgroundPath, mergedAudioPath); err != nil {
		return fmt.Errorf("step 11 failed (merge audio): %w", err)
	}

	// Step 12: Merge video with new audio track
	logx.Infof("[ProcessTask] Step 12: Merging video with new audio track")
	outputVideoPath := svcCtx.PathManager.GetOutputPath(taskID)

	if err := mediautil.MergeVideoAudio(originalVideoPath, mergedAudioPath, outputVideoPath); err != nil {
		return fmt.Errorf("step 12 failed (merge video): %w", err)
	}

	// Step 13: Save result
	logx.Infof("[ProcessTask] Step 13: Saving result")

	if err := svcCtx.RedisClient.SetTaskFields(ctx, taskID, map[string]interface{}{
		"result_file_path": outputVideoPath,
		"updated_at":       time.Now().Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("step 13 failed (save result): %w", err)
	}

	logx.Infof("[ProcessTask] Workflow completed successfully for task: %s", taskID)
	return nil
}

// cutAudioSegment cuts an audio segment from the source audio file using ffmpeg.
func cutAudioSegment(sourceAudio, outputPath string, startTime, endTime float64) error {
	cmd := mediautil.NewFFmpegCommand(
		"-i", sourceAudio,
		"-ss", strconv.FormatFloat(startTime, 'f', 3, 64),
		"-to", strconv.FormatFloat(endTime, 'f', 3, 64),
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-af", "silenceremove=start_periods=1:start_silence=0.1:start_threshold=-50dB,volume=5dB,loudnorm",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[cutAudioSegment] ffmpeg failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg cut audio segment failed: %w", err)
	}

	logx.Infof("[cutAudioSegment] Successfully cut and processed segment: %s (%.3fs - %.3fs)", outputPath, startTime, endTime)
	return nil
}

// mergeSegmentsPerSpeaker merges audio segments for each speaker to create reference audio (at least 12s).
//
// This function groups segments by speaker ID and concatenates enough segments to reach
// at least 12 seconds of audio for voice cloning reference.
//
// Parameters:
//   - svcCtx: service context
//   - taskID: task ID for generating output paths
//   - allSegments: all audio segments with paths
//
// Returns:
//   - map[string]string: map of speaker ID to merged reference audio path
//   - error: error if merge fails
func mergeSegmentsPerSpeaker(svcCtx *svc.ServiceContext, taskID string, allSegments []SegmentWithPath) (map[string]string, error) {
	const targetDuration = 12.0 // 目标时长：至少12秒

	// 步骤 1: 按说话人分组
	speakerSegments := make(map[string][]SegmentWithPath)
	for _, seg := range allSegments {
		speakerSegments[seg.SpeakerId] = append(speakerSegments[seg.SpeakerId], seg)
	}

	logx.Infof("[mergeSegmentsPerSpeaker] Found %d speakers", len(speakerSegments))

	// 步骤 2: 为每个说话人合并音频片段
	speakerReferenceAudio := make(map[string]string)

	for speakerID, segments := range speakerSegments {
		logx.Infof("[mergeSegmentsPerSpeaker] Processing speaker: %s (%d segments)", speakerID, len(segments))

		// 步骤 2.1: 选择足够的片段以达到目标时长
		var selectedSegments []SegmentWithPath
		var totalDuration float64

		for _, seg := range segments {
			selectedSegments = append(selectedSegments, seg)
			totalDuration += (seg.End - seg.Start)

			// 达到目标时长后停止
			if totalDuration >= targetDuration {
				break
			}
		}

		logx.Infof("[mergeSegmentsPerSpeaker] Speaker %s: selected %d segments (total duration: %.2fs)",
			speakerID, len(selectedSegments), totalDuration)

		// 步骤 2.2: 如果只有一个片段且时长足够，直接使用
		if len(selectedSegments) == 1 {
			speakerReferenceAudio[speakerID] = selectedSegments[0].AudioSegmentPath
			logx.Infof("[mergeSegmentsPerSpeaker] Speaker %s: using single segment as reference", speakerID)
			continue
		}

		// 步骤 2.3: 合并多个片段
		mergedPath := svcCtx.PathManager.GetIntermediatePath(
			taskID,
			fmt.Sprintf("speaker_%s_reference.wav", speakerID),
		)

		if err := concatenateAudioSegments(selectedSegments, mergedPath); err != nil {
			return nil, fmt.Errorf("failed to merge segments for speaker %s: %w", speakerID, err)
		}

		speakerReferenceAudio[speakerID] = mergedPath
		logx.Infof("[mergeSegmentsPerSpeaker] Speaker %s: merged reference audio created at %s", speakerID, mergedPath)
	}

	return speakerReferenceAudio, nil
}

// concatenateAudioSegments concatenates multiple audio segments into a single file using ffmpeg.
//
// Parameters:
//   - segments: audio segments to concatenate
//   - outputPath: path to save the concatenated audio
//
// Returns:
//   - error: error if concatenation fails
func concatenateAudioSegments(segments []SegmentWithPath, outputPath string) error {
	if len(segments) == 0 {
		return fmt.Errorf("no segments to concatenate")
	}

	// 步骤 1: 创建 concat 文件列表
	concatFilePath := outputPath + ".concat.txt"
	file, err := os.Create(concatFilePath)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}
	defer os.Remove(concatFilePath) // 清理临时文件

	// 步骤 2: 写入文件路径
	for _, seg := range segments {
		absPath, err := filepath.Abs(seg.AudioSegmentPath)
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to get absolute path for %s: %w", seg.AudioSegmentPath, err)
		}
		fmt.Fprintf(file, "file '%s'\n", absPath)
	}
	file.Close()

	// 步骤 3: 使用 ffmpeg concat 合并音频
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
		logx.Errorf("[concatenateAudioSegments] ffmpeg failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg concatenate failed: %w", err)
	}

	logx.Infof("[concatenateAudioSegments] Successfully concatenated %d segments to %s", len(segments), outputPath)
	return nil
}

func segmentDurationSeconds(seg SegmentWithPath) float64 {
	return seg.End - seg.Start
}

func calculateWordBounds(durationSeconds float64) (uint32, uint32) {
	if durationSeconds <= 0 {
		return 1, 1
	}
	target := durationSeconds * targetWordsPerSecond
	minFloat := target * (1 - targetWordTolerance)
	maxFloat := target * (1 + targetWordTolerance)
	minVal := uint32(math.Max(math.Floor(minFloat), 1))
	maxVal := uint32(math.Max(math.Ceil(maxFloat), float64(minVal+1)))
	if maxVal < minVal {
		maxVal = minVal
	}
	return minVal, maxVal
}

func formatSpeakerRole(speakerID string) string {
	if speakerID == "" {
		return "default-speaker"
	}
	return speakerID
}

func enforceTranslationLength(ctx context.Context, svcCtx *svc.ServiceContext, segments []SegmentWithPath, videoType string) error {
	for i := range segments {
		duration := segmentDurationSeconds(segments[i])
		minWords, maxWords := calculateWordBounds(duration)
		length := utf8.RuneCountInString(segments[i].TranslatedText)
		if uint32(length) >= minWords && uint32(length) <= maxWords {
			continue
		}

		logx.Infof("[ProcessTask] Segment %d length out of bounds: len=%d, expected=%d-%d", i, length, minWords, maxWords)
		optimizeReq := &pb.OptimizeRequest{
			Text:                  segments[i].TranslatedText,
			TargetDurationSeconds: duration,
			TargetWordMin:         minWords,
			TargetWordMax:         maxWords,
			SpeakerRole:           formatSpeakerRole(segments[i].SpeakerId),
			VideoType:             videoType,
		}

		optimizeResp, err := svcCtx.AIAdaptorClient.Optimize(ctx, optimizeReq)
		if err != nil {
			logx.Infof("[ProcessTask] Segment %d length adjustment failed: %v", i, err)
			continue
		}

		segments[i].TranslatedText = optimizeResp.OptimizedText
		newLen := utf8.RuneCountInString(optimizeResp.OptimizedText)
		logx.Infof("[ProcessTask] Segment %d length adjusted: new_len=%d", i, newLen)
	}

	return nil
}
