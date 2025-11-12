package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters"
	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
	pb "video-in-chinese/server/mcp/ai_adaptor/proto"
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
		// 降级策略：配置获取失败时，润色服务降级到原文本
		log.Printf("[PolishLogic] WARNING: Failed to get config (will fallback to original text): %v", err)
		log.Printf("[PolishLogic] Degraded: Returning original text due to config failure")
		return &pb.PolishResponse{
			PolishedText: req.Text, // 降级到原文本
		}, nil
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
		log.Printf("[PolishLogic] WARNING: Polishing provider not configured, returning original text")
		return &pb.PolishResponse{
			PolishedText: req.Text, // 降级到原文本
		}, nil
	}
	if appConfig.PolishingAPIKey == "" {
		log.Printf("[PolishLogic] WARNING: Polishing API key not configured, returning original text")
		return &pb.PolishResponse{
			PolishedText: req.Text, // 降级到原文本
		}, nil
	}

	log.Printf("[PolishLogic] Using polishing provider: %s", appConfig.PolishingProvider)

	// 步骤 5: 从适配器注册表获取 LLM 适配器
	adapter, err := l.registry.GetLLM(appConfig.PolishingProvider)
	if err != nil {
		// 降级策略：适配器获取失败时，降级到原文本
		log.Printf("[PolishLogic] WARNING: Failed to get LLM adapter (will fallback to original text): %v", err)
		log.Printf("[PolishLogic] Degraded: Returning original text due to adapter resolution failure")
		return &pb.PolishResponse{
			PolishedText: req.Text, // 降级到原文本
		}, nil
	}

	// 步骤 6: 调用适配器执行文本润色（带重试机制）
	// 使用配置中的 video_type 和 custom_prompt
	videoType := appConfig.PolishingVideoType
	if videoType == "" {
		videoType = "default" // 默认视频类型
	}

	customPrompt := appConfig.PolishingCustomPrompt

	// 获取模型名称，如果未配置则使用默认值
	modelName := appConfig.PolishingModelName
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite" // 默认使用 Gemini Flash（更快）
		log.Printf("[PolishLogic] WARNING: No model specified, using default: %s", modelName)
	}

	// 重试机制：最多尝试 3 次
	const maxRetries = 3
	var polishedText string
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		polishedText, err = adapter.Polish(
			req.Text,
			videoType,
			customPrompt,
			modelName,
			appConfig.PolishingAPIKey,
			appConfig.PolishingEndpoint,
		)

		// 检查是否成功且返回非空结果
		if err == nil && polishedText != "" {
			log.Printf("[PolishLogic] Polishing succeeded on attempt %d: polished_text_length=%d", attempt, len(polishedText))
			return &pb.PolishResponse{
				PolishedText: polishedText,
			}, nil
		}

		// 记录失败原因
		if err != nil {
			lastErr = err
			log.Printf("[PolishLogic] WARNING: Polishing attempt %d/%d failed with error: %v", attempt, maxRetries, err)
		} else {
			lastErr = fmt.Errorf("返回空字符串")
			log.Printf("[PolishLogic] WARNING: Polishing attempt %d/%d returned empty string", attempt, maxRetries)
		}

		// 如果不是最后一次尝试，等待后重试
		if attempt < maxRetries {
			log.Printf("[PolishLogic] Retrying polishing (attempt %d/%d)...", attempt+1, maxRetries)
		}
	}

	// 步骤 7: 所有重试都失败，降级策略：返回原文本
	log.Printf("[PolishLogic] WARNING: All %d polishing attempts failed (last error: %v), falling back to original text", maxRetries, lastErr)
	return &pb.PolishResponse{
		PolishedText: req.Text, // 降级：返回原文本
	}, nil
}
