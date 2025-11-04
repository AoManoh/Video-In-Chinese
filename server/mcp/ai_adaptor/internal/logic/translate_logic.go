package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// TranslateLogic 翻译服务逻辑
type TranslateLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewTranslateLogic 创建新的翻译服务逻辑实例
func NewTranslateLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *TranslateLogic {
	return &TranslateLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessTranslate 处理翻译请求
// 步骤：
//  1. 从 Redis 读取翻译适配器配置（使用 ConfigManager）
//  2. 从适配器注册表获取对应的翻译适配器实例
//  3. 调用适配器的 Translate 方法执行翻译
//  4. 处理错误并返回翻译结果
func (l *TranslateLogic) ProcessTranslate(ctx context.Context, req *pb.TranslateRequest) (*pb.TranslateResponse, error) {
	log.Printf("[TranslateLogic] Processing translate request: text_length=%d, source_lang=%s, target_lang=%s",
		len(req.Text), req.SourceLang, req.TargetLang)

	// 步骤 1: 验证请求参数
	if req.Text == "" {
		return nil, fmt.Errorf("待翻译文本不能为空")
	}
	if req.SourceLang == "" {
		return nil, fmt.Errorf("源语言代码不能为空")
	}
	if req.TargetLang == "" {
		return nil, fmt.Errorf("目标语言代码不能为空")
	}

	// 步骤 2: 从 Redis 读取配置
	appConfig, err := l.configManager.GetConfig(ctx)
	if err != nil {
		log.Printf("[TranslateLogic] ERROR: Failed to get config: %v", err)
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 步骤 3: 验证翻译配置
	if appConfig.TranslationProvider == "" {
		return nil, fmt.Errorf("翻译服务商未配置")
	}
	if appConfig.TranslationAPIKey == "" {
		return nil, fmt.Errorf("翻译 API 密钥未配置")
	}

	log.Printf("[TranslateLogic] Using translation provider: %s", appConfig.TranslationProvider)

	// 步骤 4: 从适配器注册表获取翻译适配器
	adapter, err := l.registry.GetTranslation(appConfig.TranslationProvider)
	if err != nil {
		log.Printf("[TranslateLogic] ERROR: Failed to get translation adapter: %v", err)
		return nil, fmt.Errorf("获取翻译适配器失败: %w", err)
	}

	// 步骤 5: 调用适配器执行翻译
	// 使用配置中的 video_type，如果请求中没有指定
	videoType := appConfig.TranslationVideoType
	if videoType == "" {
		videoType = "default" // 默认视频类型
	}

	translatedText, err := adapter.Translate(
		req.Text,
		req.SourceLang,
		req.TargetLang,
		videoType,
		appConfig.TranslationAPIKey,
		appConfig.TranslationEndpoint,
	)
	if err != nil {
		log.Printf("[TranslateLogic] ERROR: Translation failed: %v", err)
		return nil, fmt.Errorf("翻译失败: %w", err)
	}

	// 步骤 6: 返回结果
	log.Printf("[TranslateLogic] Translation completed successfully: translated_text_length=%d", len(translatedText))
	return &pb.TranslateResponse{
		TranslatedText: translatedText,
	}, nil
}

