package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// CloneVoiceLogic orchestrates voice cloning requests by combining
// configuration management with adapter resolution. It keeps the voice cloning
// workflow isolated from transport handlers so providers can be swapped with
// minimal surface area.
type CloneVoiceLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewCloneVoiceLogic builds a CloneVoiceLogic instance using the shared adapter
// registry and configuration manager.
func NewCloneVoiceLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *CloneVoiceLogic {
	return &CloneVoiceLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessCloneVoice manages an end-to-end voice cloning workflow.
//
// Workflow:
//  1. Validate required request fields.
//  2. Load cloning provider credentials from ConfigManager (Redis + cache).
//  3. Resolve the matching voice-cloning adapter via AdapterRegistry.
//  4. Invoke CloneVoice on the adapter and return the generated audio path.
//
// Parameters:
//   - ctx: Propagates cancellation and deadlines to the adapter invocation.
//   - req: Protobuf payload with speaker metadata, reference audio, and text.
//
// Returns:
//   - *pb.CloneVoiceResponse on success.
//   - error for validation issues, configuration lookup problems, missing
//     adapters, or provider execution errors.
//
// Design considerations:
//   - AdapterRegistry decouples provider-specific behaviour so adding vendors
//     remains a registry operation.
//   - Errors are wrapped to preserve root causes for observability.
//
// Example:
//
//	res, err := l.ProcessCloneVoice(ctx, &pb.CloneVoiceRequest{
//		SpeakerId:       "speaker-001",
//		Text:            "示例台词",
//		ReferenceAudio:  "oss://bucket/reference.wav",
//	})
//	if err != nil {
//		return err
//	}
//	log.Println(res.AudioPath)
func (l *CloneVoiceLogic) ProcessCloneVoice(ctx context.Context, req *pb.CloneVoiceRequest) (*pb.CloneVoiceResponse, error) {
	log.Printf("[CloneVoiceLogic] Processing clone voice request: speaker_id=%s, text_length=%d",
		req.SpeakerId, len(req.Text))

	// 步骤 1: 验证请求参数
	if req.SpeakerId == "" {
		return nil, fmt.Errorf("说话人 ID 不能为空")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("待合成文本不能为空")
	}
	if req.ReferenceAudio == "" {
		return nil, fmt.Errorf("参考音频路径不能为空")
	}

	// 步骤 2: 从 Redis 读取配置
	appConfig, err := l.configManager.GetConfig(ctx)
	if err != nil {
		log.Printf("[CloneVoiceLogic] ERROR: Failed to get config: %v", err)
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 步骤 3: 验证声音克隆配置
	if appConfig.VoiceCloningProvider == "" {
		return nil, fmt.Errorf("声音克隆服务商未配置")
	}
	if appConfig.VoiceCloningAPIKey == "" {
		return nil, fmt.Errorf("声音克隆 API 密钥未配置")
	}

	log.Printf("[CloneVoiceLogic] Using voice cloning provider: %s", appConfig.VoiceCloningProvider)

	// 步骤 4: 从适配器注册表获取声音克隆适配器
	adapter, err := l.registry.GetVoiceCloning(appConfig.VoiceCloningProvider)
	if err != nil {
		log.Printf("[CloneVoiceLogic] ERROR: Failed to get voice cloning adapter: %v", err)
		return nil, fmt.Errorf("获取声音克隆适配器失败: %w", err)
	}

	// 步骤 5: 调用适配器执行声音克隆
	audioPath, err := adapter.CloneVoice(
		req.SpeakerId,
		req.Text,
		req.ReferenceAudio,
		appConfig.VoiceCloningAPIKey,
		appConfig.VoiceCloningEndpoint,
	)
	if err != nil {
		log.Printf("[CloneVoiceLogic] ERROR: Voice cloning failed: %v", err)
		return nil, fmt.Errorf("声音克隆失败: %w", err)
	}

	// 步骤 6: 返回结果
	log.Printf("[CloneVoiceLogic] Voice cloning completed successfully: audio_path=%s", audioPath)
	return &pb.CloneVoiceResponse{
		AudioPath: audioPath,
	}, nil
}
