package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters"
	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
	pb "video-in-chinese/server/mcp/ai_adaptor/proto"
)

// TranslateLogic orchestrates translation workflows by combining configuration
// lookups with adapter resolution. It keeps business logic isolated from
// transport layers so providers can be swapped without touching handlers.
type TranslateLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewTranslateLogic wires a TranslateLogic with the shared adapter registry and
// configuration manager. Consumers should reuse the returned instance to take
// advantage of ConfigManager caching.
func NewTranslateLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *TranslateLogic {
	return &TranslateLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessTranslate executes a translation request using the provider defined in
// runtime configuration.
//
// Workflow:
//  1. Validate source text and language codes.
//  2. Load provider credentials and feature toggles from ConfigManager
//     (Redis + in-memory cache).
//  3. Resolve the translation adapter from AdapterRegistry.
//  4. Invoke the adapter and return the translated text.
//
// Parameters:
//   - ctx: Propagates deadlines and cancellation to downstream adapters.
//   - req: gRPC payload describing the text and language pair.
//
// Returns:
//   - *pb.TranslateResponse containing the translated text on success.
//   - error describing validation, configuration, registry, or adapter
//     failures.
//
// Design considerations:
//   - Video type fallbacks ensure callers are not required to pass optional
//     metadata while still enabling provider-specific prompts.
//   - Formatting and error messages remain user-oriented while preserving root
//     causes for observability.
//
// Example:
//
//	res, err := l.ProcessTranslate(ctx, &pb.TranslateRequest{
//		Text:       "你好世界",
//		SourceLang: "zh-CN",
//		TargetLang: "en-US",
//	})
//	if err != nil {
//		return err
//	}
//	log.Printf("translated length: %d", len(res.TranslatedText))
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

	// 使用配置中的模型名称，如果未配置则使用默认值
	modelName := appConfig.TranslationModelName
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite" // 默认模型
		log.Printf("[TranslateLogic] WARNING: translation_model_name not configured, using default: %s", modelName)
	}

	translationCtx := &adapters.TranslationContext{
		DurationSeconds: req.DurationSeconds,
		SpeakerRole:     req.SpeakerRole,
		TargetWordMin:   req.TargetWordMin,
		TargetWordMax:   req.TargetWordMax,
	}

	translatedText, err := adapter.Translate(
		req.Text,
		req.SourceLang,
		req.TargetLang,
		videoType,
		modelName,
		appConfig.TranslationAPIKey,
		appConfig.TranslationEndpoint,
		translationCtx,
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
