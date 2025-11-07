package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters"
	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
	pb "video-in-chinese/server/mcp/ai_adaptor/proto"
)

// OptimizeLogic orchestrates translation optimization flows by combining
// ConfigManager lookups with LLM adapter resolution. Keeping the orchestration
// in a thin logic layer allows transport code to stay simple while new
// providers can be registered without changing handlers.
type OptimizeLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewOptimizeLogic returns an OptimizeLogic bound to the shared adapter
// registry and configuration manager. Callers should reuse the instance so
// ConfigManager's Redis + in-memory cache stays warm across requests.
func NewOptimizeLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *OptimizeLogic {
	return &OptimizeLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessOptimize validates an optimization request, fetches provider settings
// from ConfigManager, resolves the correct LLM adapter, and delegates the
// Optimize call. It also honours the OptimizationEnabled toggle so rollbacks do
// not require code changes.
//
// Workflow:
//  1. Ensure the request supplies non-empty text.
//  2. Load optimization configuration (provider, API key, endpoint, toggle)
//     from ConfigManager which caches Redis results.
//  3. Short-circuit with the original text if optimization is disabled.
//  4. Resolve the adapter from AdapterRegistry and invoke Optimize.
//
// Parameters:
//   - ctx: Carries deadlines/cancellation to the adapter execution.
//   - req: Protobuf payload describing the text to optimize.
//
// Returns:
//   - *pb.OptimizeResponse with the optimized text on success.
//   - error describing validation issues, configuration lookup failures,
//     registry misses, or adapter execution errors.
//
// Example:
//
//	res, err := l.ProcessOptimize(ctx, &pb.OptimizeRequest{Text: "稿件"})
//	if err != nil {
//		return err
//	}
//	log.Println(res.OptimizedText)
func (l *OptimizeLogic) ProcessOptimize(ctx context.Context, req *pb.OptimizeRequest) (*pb.OptimizeResponse, error) {
	log.Printf("[OptimizeLogic] Processing optimize request: text_length=%d", len(req.Text))

	// 步骤 1: 验证请求参数
	if req.Text == "" {
		return nil, fmt.Errorf("待优化文本不能为空")
	}

	// 步骤 2: 从 Redis 读取配置
	appConfig, err := l.configManager.GetConfig(ctx)
	if err != nil {
		log.Printf("[OptimizeLogic] ERROR: Failed to get config: %v", err)
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 步骤 3: 检查译文优化是否启用
	if !appConfig.OptimizationEnabled {
		log.Printf("[OptimizeLogic] WARNING: Optimization is disabled, returning original text")
		return &pb.OptimizeResponse{
			OptimizedText: req.Text, // 返回原文本
		}, nil
	}

	// 步骤 4: 验证译文优化配置
	if appConfig.OptimizationProvider == "" {
		return nil, fmt.Errorf("译文优化服务商未配置")
	}
	if appConfig.OptimizationAPIKey == "" {
		return nil, fmt.Errorf("译文优化 API 密钥未配置")
	}

	log.Printf("[OptimizeLogic] Using optimization provider: %s", appConfig.OptimizationProvider)

	// 步骤 5: 从适配器注册表获取 LLM 适配器
	adapter, err := l.registry.GetLLM(appConfig.OptimizationProvider)
	if err != nil {
		log.Printf("[OptimizeLogic] ERROR: Failed to get LLM adapter: %v", err)
		return nil, fmt.Errorf("获取 LLM 适配器失败: %w", err)
	}

	// 步骤 6: 调用适配器执行译文优化
	// 获取模型名称，如果未配置则使用默认值
	modelName := appConfig.OptimizationModelName
	if modelName == "" {
		modelName = "gemini-2.5-flash" // 默认使用 Gemini Flash（更快）
		log.Printf("[OptimizeLogic] WARNING: No model specified, using default: %s", modelName)
	}

	optimizedText, err := adapter.Optimize(
		req.Text,
		modelName,
		appConfig.OptimizationAPIKey,
		appConfig.OptimizationEndpoint,
	)
	if err != nil {
		log.Printf("[OptimizeLogic] ERROR: Optimization failed: %v", err)
		return nil, fmt.Errorf("译文优化失败: %w", err)
	}

	// 步骤 7: 返回结果
	log.Printf("[OptimizeLogic] Optimization completed successfully: optimized_text_length=%d", len(optimizedText))
	return &pb.OptimizeResponse{
		OptimizedText: optimizedText,
	}, nil
}
