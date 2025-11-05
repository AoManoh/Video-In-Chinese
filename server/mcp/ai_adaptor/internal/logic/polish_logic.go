package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// PolishLogic encapsulates text polishing flows backed by large language model
// adapters. It coordinates configuration toggles, provider selection, and error
// reporting for the polish endpoint.
type PolishLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewPolishLogic constructs a PolishLogic instance bound to the shared adapter
// registry and configuration manager.
func NewPolishLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *PolishLogic {
	return &PolishLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessPolish runs the polishing pipeline for a block of text using the LLM
// provider defined in configuration.
//
// Workflow:
//  1. Validate the request payload.
//  2. Fetch configuration and feature toggles from ConfigManager.
//  3. Short-circuit when polishing is disabled, returning the original text.
//  4. Resolve the LLM adapter and invoke its Polish implementation with the
//     configured video type and optional custom prompt.
//
// Parameters:
//   - ctx: Propagates cancellation and timeouts to downstream components.
//   - req: gRPC payload describing the original text to polish.
//
// Returns:
//   - *pb.PolishResponse carrying the polished text on success.
//   - error describing validation, configuration, registry, or adapter
//     failures.
//
// Design considerations:
//   - Toggle checks allow operators to disable polishing without code changes
//     while keeping observability logs consistent.
//   - Custom prompts are optional; when omitted, defaults keep provider prompts
//     stable across tenants.
//
// Example:
//
//	res, err := l.ProcessPolish(ctx, &pb.PolishRequest{Text: "raw caption"})
//	if err != nil {
//		return err
//	}
//	log.Println(res.PolishedText)
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
