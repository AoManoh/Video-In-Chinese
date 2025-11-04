package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// PolishLogic 文本润色服务逻辑
type PolishLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewPolishLogic 创建新的文本润色服务逻辑实例
func NewPolishLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *PolishLogic {
	return &PolishLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessPolish 处理文本润色请求
// 步骤：
//  1. 从 Redis 读取 LLM 适配器配置（使用 ConfigManager）
//  2. 检查文本润色是否启用
//  3. 从适配器注册表获取对应的 LLM 适配器实例
//  4. 调用适配器的 Polish 方法执行文本润色
//  5. 处理错误并返回润色结果
func (l *PolishLogic) ProcessPolish(ctx context.Context, req *pb.PolishRequest) (*pb.PolishResponse, error) {
	log.Printf("[PolishLogic] Processing polish request: text_length=%d", len(req.Text))

	// 步骤 1: 验证请求参数
	if req.Text == "" {
		return nil, fmt.Errorf("待润色文本不能为空")
	}

	// 步骤 2: 从 Redis 读取配置
	appConfig, err := l.configManager.GetConfig(ctx)
	if err != nil {
		log.Printf("[PolishLogic] ERROR: Failed to get config: %v", err)
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 步骤 3: 检查文本润色是否启用
	if !appConfig.PolishingEnabled {
		log.Printf("[PolishLogic] WARNING: Polishing is disabled, returning original text")
		return &pb.PolishResponse{
			PolishedText: req.Text, // 返回原文本
		}, nil
	}

	// 步骤 4: 验证文本润色配置
	if appConfig.PolishingProvider == "" {
		return nil, fmt.Errorf("文本润色服务商未配置")
	}
	if appConfig.PolishingAPIKey == "" {
		return nil, fmt.Errorf("文本润色 API 密钥未配置")
	}

	log.Printf("[PolishLogic] Using polishing provider: %s", appConfig.PolishingProvider)

	// 步骤 5: 从适配器注册表获取 LLM 适配器
	adapter, err := l.registry.GetLLM(appConfig.PolishingProvider)
	if err != nil {
		log.Printf("[PolishLogic] ERROR: Failed to get LLM adapter: %v", err)
		return nil, fmt.Errorf("获取 LLM 适配器失败: %w", err)
	}

	// 步骤 6: 调用适配器执行文本润色
	// 使用配置中的 video_type 和 custom_prompt
	videoType := appConfig.PolishingVideoType
	if videoType == "" {
		videoType = "default" // 默认视频类型
	}

	customPrompt := appConfig.PolishingCustomPrompt

	polishedText, err := adapter.Polish(
		req.Text,
		videoType,
		customPrompt,
		appConfig.PolishingAPIKey,
		appConfig.PolishingEndpoint,
	)
	if err != nil {
		log.Printf("[PolishLogic] ERROR: Polishing failed: %v", err)
		return nil, fmt.Errorf("文本润色失败: %w", err)
	}

	// 步骤 7: 返回结果
	log.Printf("[PolishLogic] Polishing completed successfully: polished_text_length=%d", len(polishedText))
	return &pb.PolishResponse{
		PolishedText: polishedText,
	}, nil
}

