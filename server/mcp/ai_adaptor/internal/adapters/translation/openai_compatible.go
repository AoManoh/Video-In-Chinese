package translation

import (
	"context"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
)

// OpenAICompatibleTranslationAdapter 封装 OpenAI 兼容格式的翻译 API 调用，实现 TranslationAdapter 接口。
//
// 功能说明:
//   - 使用大语言模型（LLM）进行文本翻译，支持所有兼容 OpenAI Chat Completions API 的服务。
//   - 通过 System Prompt 指导模型进行专业翻译，保证翻译质量和一致性。
//
// 设计决策:
//   - 使用 github.com/sashabaranov/go-openai 库统一调用所有兼容服务（Gemini、通义千问、DeepSeek 等）。
//   - 通过自定义 BaseURL 支持各种代理服务（gemini-balance、one-api、new-api 等）。
//
// 使用示例:
//
//	adapter := NewOpenAICompatibleTranslationAdapter()
//	translated, err := adapter.Translate("Hello", "en", "zh", "professional_tech", apiKey, "https://balance.aomanoh.com/v1")
//
// 参数说明:
//   - 不适用: 结构体通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于维护客户端实例。
//
// 错误处理说明:
//   - Translate 方法会根据 API 响应返回具体错误。
//
// 注意事项:
//   - 调用前需准备兼容 OpenAI 格式的 API Key 和 Endpoint。
//   - 不同服务的 Model 名称可能不同，需要在配置中正确指定。
type OpenAICompatibleTranslationAdapter struct{}

// NewOpenAICompatibleTranslationAdapter 创建 OpenAI 兼容翻译适配器实例。
//
// 功能说明:
//   - 提供适配器实例供业务层直接使用。
//
// 设计决策:
//   - 不维护长连接客户端，每次调用时根据配置创建新客户端，避免配置变更时的状态不一致。
//
// 使用示例:
//
//	adapter := NewOpenAICompatibleTranslationAdapter()
//
// 返回值说明:
//
//	*OpenAICompatibleTranslationAdapter: 初始化完成的适配器实例。
//
// 注意事项:
//   - 适配器本身无状态，可安全并发使用。
func NewOpenAICompatibleTranslationAdapter() *OpenAICompatibleTranslationAdapter {
	return &OpenAICompatibleTranslationAdapter{}
}

// Translate 执行一次文本翻译并返回翻译后的文本。
//
// 功能说明:
//   - 使用大语言模型进行文本翻译，通过 System Prompt 指导模型行为。
//   - 支持根据视频类型调整翻译风格（专业、轻松、教育等）。
//
// 设计决策:
//   - 使用 Chat Completions API 而非专用翻译 API，提供更好的上下文理解和风格控制。
//   - 温度设置为 0.3，在保证翻译准确性的同时允许适度的表达灵活性。
//
// 使用示例:
//
//	translated, err := adapter.Translate("Hello, world!", "en", "zh", "professional_tech", apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待翻译文本，不能为空。
//	sourceLang string: 源语言代码（如 "en"），可为空（模型自动检测）。
//	targetLang string: 目标语言代码（如 "zh"），必填。
//	videoType string: 视频风格标签，用于调整翻译语气（如 "professional_tech", "casual_natural"）。
//	apiKey string: OpenAI 或兼容服务的 API 密钥。
//	endpoint string: 自定义 API 地址（如 "https://balance.aomanoh.com/v1"），空字符串使用默认 OpenAI 地址。
//
// 返回值说明:
//
//	string: 翻译后的文本。
//	error: 请求失败或解析错误时返回。
//
// 错误处理说明:
//   - 401/403 表示 API 密钥无效。
//   - 429 表示请求限流，需要重试或降级。
//   - 400 表示请求参数错误。
//   - 5xx 表示服务端故障。
//
// 注意事项:
//   - 长文本可能触发 Token 限制，调用方应做好分段处理。
//   - 不同服务的模型名称不同，需要在配置中正确指定（如 "gemini-2.5-pro", "qwen-plus"）。
func (o *OpenAICompatibleTranslationAdapter) Translate(text, sourceLang, targetLang, videoType, apiKey, endpoint string) (string, error) {
	log.Printf("[OpenAICompatibleTranslationAdapter] Starting translation: text_length=%d, source=%s, target=%s, videoType=%s",
		len(text), sourceLang, targetLang, videoType)

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待翻译文本不能为空")
	}
	if targetLang == "" {
		return "", fmt.Errorf("目标语言代码不能为空")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API 密钥不能为空")
	}

	// 步骤 2: 初始化 OpenAI 客户端
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
		log.Printf("[OpenAICompatibleTranslationAdapter] Using custom endpoint: %s", endpoint)
	}
	client := openai.NewClientWithConfig(config)

	// 步骤 3: 构建翻译 Prompt
	systemPrompt := o.buildSystemPrompt(targetLang, videoType)
	userPrompt := o.buildUserPrompt(text, sourceLang, targetLang)

	log.Printf("[OpenAICompatibleTranslationAdapter] System prompt: %s", systemPrompt)
	log.Printf("[OpenAICompatibleTranslationAdapter] User prompt length: %d", len(userPrompt))

	// 步骤 4: 调用 OpenAI API
	ctx := context.Background()
	request := openai.ChatCompletionRequest{
		Model: "gemini-2.5-flash", // 默认模型，与润色/优化保持一致
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.3, // 翻译任务使用较低温度，保证准确性和一致性
	}

	log.Printf("[OpenAICompatibleTranslationAdapter] Sending request to API...")
	response, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		log.Printf("[OpenAICompatibleTranslationAdapter] ERROR: API call failed: %v", err)
		return "", fmt.Errorf("调用翻译 API 失败: %w", err)
	}

	// 步骤 5: 提取翻译结果
	if len(response.Choices) == 0 {
		log.Printf("[OpenAICompatibleTranslationAdapter] ERROR: No choices in response")
		return "", fmt.Errorf("API 返回结果为空")
	}

	translatedText := response.Choices[0].Message.Content
	log.Printf("[OpenAICompatibleTranslationAdapter] Translation completed successfully: result_length=%d", len(translatedText))

	return translatedText, nil
}

// buildSystemPrompt 根据目标语言和视频类型构建系统提示词。
//
// 功能说明:
//   - 定义模型的角色和翻译要求，确保翻译质量和风格一致性。
//
// 参数说明:
//
//	targetLang string: 目标语言代码。
//	videoType string: 视频风格标签。
//
// 返回值说明:
//
//	string: 系统提示词。
func (o *OpenAICompatibleTranslationAdapter) buildSystemPrompt(targetLang, videoType string) string {
	// 基础角色定义
	basePrompt := "你是一位专业的翻译专家，擅长将各种语言的文本准确、流畅地翻译成目标语言。"

	// 根据视频类型调整翻译风格
	styleGuide := ""
	switch videoType {
	case "professional_tech":
		styleGuide = "请使用专业、准确的技术术语，保持正式的语气。"
	case "casual_natural":
		styleGuide = "请使用轻松、自然的口语化表达，让翻译更贴近日常对话。"
	case "educational_rigorous":
		styleGuide = "请使用严谨、清晰的教育性语言，确保概念表达准确。"
	case "entertainment_lively":
		styleGuide = "请使用生动、活泼的表达方式，让翻译更有趣味性。"
	default:
		styleGuide = "请保持翻译的准确性和流畅性。"
	}

	// 翻译要求
	requirements := fmt.Sprintf(`
翻译要求：
1. 准确传达原文的意思，不要遗漏或添加信息
2. 使用地道的%s表达，符合目标语言的习惯
3. 保持原文的语气和风格
4. %s
5. 只返回翻译结果，不要包含任何解释或说明
`, getLanguageName(targetLang), styleGuide)

	return basePrompt + requirements
}

// buildUserPrompt 构建用户提示词，包含待翻译文本和语言信息。
//
// 功能说明:
//   - 明确指定源语言和目标语言，引导模型进行翻译。
//
// 参数说明:
//
//	text string: 待翻译文本。
//	sourceLang string: 源语言代码。
//	targetLang string: 目标语言代码。
//
// 返回值说明:
//
//	string: 用户提示词。
func (o *OpenAICompatibleTranslationAdapter) buildUserPrompt(text, sourceLang, targetLang string) string {
	if sourceLang != "" {
		return fmt.Sprintf("请将以下%s文本翻译成%s：\n\n%s",
			getLanguageName(sourceLang), getLanguageName(targetLang), text)
	}
	return fmt.Sprintf("请将以下文本翻译成%s：\n\n%s",
		getLanguageName(targetLang), text)
}

// getLanguageName 将语言代码转换为中文名称。
//
// 功能说明:
//   - 提供更友好的语言名称，用于构建提示词。
//
// 参数说明:
//
//	langCode string: 语言代码（如 "en", "zh", "ja"）。
//
// 返回值说明:
//
//	string: 语言的中文名称。
func getLanguageName(langCode string) string {
	languageNames := map[string]string{
		"en":    "英文",
		"zh":    "中文",
		"zh-CN": "简体中文",
		"zh-TW": "繁体中文",
		"ja":    "日文",
		"ko":    "韩文",
		"fr":    "法文",
		"de":    "德文",
		"es":    "西班牙文",
		"ru":    "俄文",
		"ar":    "阿拉伯文",
		"pt":    "葡萄牙文",
		"it":    "意大利文",
	}

	if name, ok := languageNames[langCode]; ok {
		return name
	}
	return langCode // 如果没有映射，直接返回代码
}
