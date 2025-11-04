package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// CloneVoiceLogic 声音克隆服务逻辑
type CloneVoiceLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewCloneVoiceLogic 创建新的声音克隆服务逻辑实例
func NewCloneVoiceLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *CloneVoiceLogic {
	return &CloneVoiceLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessCloneVoice 处理声音克隆请求
// 步骤：
//  1. 从 Redis 读取声音克隆适配器配置（使用 ConfigManager）
//  2. 从适配器注册表获取对应的声音克隆适配器实例
//  3. 调用适配器的 CloneVoice 方法执行声音克隆
//  4. 处理错误并返回音频文件路径
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

