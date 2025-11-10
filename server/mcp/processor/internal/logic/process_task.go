package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
		translateReq := &pb.TranslateRequest{
			Text:       allSegments[i].Text,
			SourceLang: sourceLang,
			TargetLang: targetLang,
			VideoType:  videoType,
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
			optimizeReq := &pb.OptimizeRequest{
				Text: allSegments[i].TranslatedText,
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

	// Continue workflow: steps 8-13
	return continueWorkflow(ctx, svcCtx, taskID, originalVideoPath, allSegments, originalAudioPath, backgroundPath)
}

// continueWorkflow continues the workflow from step 8 to step 13.
func continueWorkflow(ctx context.Context, svcCtx *svc.ServiceContext, taskID, originalVideoPath string,
	allSegments []SegmentWithPath, originalAudioPath, backgroundPath string) error {

	// Step 8: Clone voice for each speaker
	logx.Infof("[ProcessTask] Step 8: Cloning voice for each speaker")
	clonedSegments := make([]composer.AudioSegment, 0, len(allSegments))

	for i, seg := range allSegments {
		cloneReq := &pb.CloneVoiceRequest{
			SpeakerId:      seg.SpeakerId,
			Text:           seg.TranslatedText,
			ReferenceAudio: seg.AudioSegmentPath,
		}

		cloneResp, err := svcCtx.AIAdaptorClient.CloneVoice(ctx, cloneReq)
		if err != nil {
			return fmt.Errorf("step 8 failed (clone voice segment %d): %w", i, err)
		}

		// Store cloned audio path temporarily (will be adjusted in Step 8.5)
		clonedSegments = append(clonedSegments, composer.AudioSegment{
			StartTime: time.Duration(seg.Start * float64(time.Second)),
			FilePath:  cloneResp.AudioPath,
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
		"-c", "copy",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logx.Errorf("[cutAudioSegment] ffmpeg failed: %v, output: %s", err, string(output))
		return fmt.Errorf("ffmpeg cut audio segment failed: %w", err)
	}

	logx.Infof("[cutAudioSegment] Successfully cut segment: %s (%.3fs - %.3fs)", outputPath, startTime, endTime)
	return nil
}
