package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// ASRLogic ASR 服务逻辑
type ASRLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewASRLogic 创建新的 ASR 服务逻辑实例
func NewASRLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *ASRLogic {
	return &ASRLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessASR 处理 ASR 请求
// 步骤：
//  1. 从 Redis 读取 ASR 适配器配置（使用 ConfigManager）
//  2. 从适配器注册表获取对应的 ASR 适配器实例
//  3. 调用适配器的 ASR 方法执行语音识别
//  4. 处理错误并返回说话人列表
func (l *ASRLogic) ProcessASR(ctx context.Context, req *pb.ASRRequest) (*pb.ASRResponse, error) {
	log.Printf("[ASRLogic] Processing ASR request: audio_path=%s", req.AudioPath)

	// 步骤 1: 验证请求参数
	if req.AudioPath == "" {
		return nil, fmt.Errorf("音频文件路径不能为空")
	}

	// 步骤 2: 从 Redis 读取配置
	appConfig, err := l.configManager.GetConfig(ctx)
	if err != nil {
		log.Printf("[ASRLogic] ERROR: Failed to get config: %v", err)
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 步骤 3: 验证 ASR 配置
	if appConfig.ASRProvider == "" {
		return nil, fmt.Errorf("ASR 服务商未配置")
	}
	if appConfig.ASRAPIKey == "" {
		return nil, fmt.Errorf("ASR API 密钥未配置")
	}

	log.Printf("[ASRLogic] Using ASR provider: %s", appConfig.ASRProvider)

	// 步骤 4: 从适配器注册表获取 ASR 适配器
	adapter, err := l.registry.GetASR(appConfig.ASRProvider)
	if err != nil {
		log.Printf("[ASRLogic] ERROR: Failed to get ASR adapter: %v", err)
		return nil, fmt.Errorf("获取 ASR 适配器失败: %w", err)
	}

	// 步骤 5: 调用适配器执行语音识别
	speakers, err := adapter.ASR(req.AudioPath, appConfig.ASRAPIKey, appConfig.ASREndpoint)
	if err != nil {
		log.Printf("[ASRLogic] ERROR: ASR failed: %v", err)
		return nil, fmt.Errorf("语音识别失败: %w", err)
	}

	// 步骤 6: 返回结果
	log.Printf("[ASRLogic] ASR completed successfully: %d speakers found", len(speakers))
	return &pb.ASRResponse{
		Speakers: speakers,
	}, nil
}

