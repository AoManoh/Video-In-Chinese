package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// OptimizeLogic 译文优化服务逻辑
type OptimizeLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewOptimizeLogic 创建新的译文优化服务逻辑实例
func NewOptimizeLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *OptimizeLogic {
	return &OptimizeLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessOptimize 处理译文优化请求
// 步骤：
//  1. 从 Redis 读取 LLM 适配器配置（使用 ConfigManager）
//  2. 检查译文优化是否启用
//  3. 从适配器注册表获取对应的 LLM 适配器实例
//  4. 调用适配器的 Optimize 方法执行译文优化
//  5. 处理错误并返回优化结果
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
	optimizedText, err := adapter.Optimize(
		req.Text,
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

