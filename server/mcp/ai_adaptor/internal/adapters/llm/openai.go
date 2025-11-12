package llm

import (
	"context"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"

	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters"
)

// OpenAILLMAdapter 封装 OpenAI Chat Completions API，实现 LLMAdapter 接口。
//
// 功能说明:
//   - 提供文本润色与脚本优化的统一入口，支持自定义 endpoint 与第三方中转服务。
//   - 使用 go-openai 库统一调用所有兼容 OpenAI Chat Completions API 的服务。
//
// 设计决策:
//   - 使用 github.com/sashabaranov/go-openai 库，通过自定义 BaseURL 支持各种代理服务。
//   - 不维护长连接客户端，每次调用时根据配置创建新客户端，避免配置变更时的状态不一致。
//
// 使用示例:
//
//	adapter := NewOpenAILLMAdapter()
//	polished, err := adapter.Polish(text, "professional_tech", "", apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体本身无状态，可安全并发使用。
//
// 错误处理说明:
//   - 由 Polish/Optimize 方法根据 API 响应返回具体错误。
//
// 注意事项:
//   - endpoint 可指向代理服务（one-api、gemini-balance 等）以满足不同部署需求。
type OpenAILLMAdapter struct{}

// NewOpenAILLMAdapter 创建 OpenAI LLM 适配器实例。
//
// 功能说明:
//   - 提供适配器实例供业务层直接使用。
//
// 设计决策:
//   - 不维护长连接客户端，每次调用时根据配置创建新客户端，避免配置变更时的状态不一致。
//
// 使用示例:
//
//	adapter := NewOpenAILLMAdapter()
//
// 返回值说明:
//
//	*OpenAILLMAdapter: 初始化完成的适配器实例。
//
// 注意事项:
//   - 适配器本身无状态，可安全并发使用。
func NewOpenAILLMAdapter() *OpenAILLMAdapter {
	return &OpenAILLMAdapter{}
}

// Polish 执行文本润色并返回润色后的内容。
//
// 功能说明:
//   - 根据视频类型生成系统提示词，组合自定义 Prompt，调用 OpenAI 模型润色文本。
//   - 使用 go-openai 库统一调用所有兼容 OpenAI Chat Completions API 的服务。
//
// 设计决策:
//   - 使用 go-openai 库的 CreateChatCompletion 方法，自动处理 URL 拼接和错误处理。
//   - 温度设置为 0.7，在保证润色质量的同时允许适度的创造性。
//
// 使用示例:
//
//	polished, err := adapter.Polish(text, "professional_tech", "", "gemini-2.5-pro", apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待润色文本，不能为空。
//	videoType string: 视频语气标签，决定系统 Prompt 语气。
//	customPrompt string: 可选自定义提示词，空字符串使用默认模版。
//	modelName string: LLM 模型名称（如 "gpt-4o", "gemini-2.5-pro", "gemini-2.5-flash-lite"）。
//	apiKey string: OpenAI 或兼容代理的鉴权密钥。
//	endpoint string: 可选自定义 API 地址，留空使用默认 https://api.openai.com。
//
// 返回值说明:
//
//	string: 润色后的文本。
//	error: 调用失败或返回内容为空时出错。
//
// 错误处理说明:
//   - 401/403 表示 API 密钥无效。
//   - 429 表示请求限流，需要重试或降级。
//   - 400 表示请求参数错误。
//   - 5xx 表示服务端故障。
//
// 注意事项:
//   - 长文本可能触发 Token 限制，调用方应做好截断或重试策略。
func (o *OpenAILLMAdapter) Polish(text, videoType, customPrompt, modelName, apiKey, endpoint string) (string, error) {
	log.Printf("[OpenAILLMAdapter] Starting text polishing: video_type=%s, model=%s", videoType, modelName)

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待润色的文本不能为空")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API 密钥不能为空")
	}
	if modelName == "" {
		modelName = "gpt-4o" // 默认模型
		log.Printf("[OpenAILLMAdapter] WARNING: No model specified, using default: %s", modelName)
	}

	// 步骤 2: 初始化 OpenAI 客户端
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
		log.Printf("[OpenAILLMAdapter] Using custom endpoint: %s", endpoint)
	}
	client := openai.NewClientWithConfig(config)

	// 步骤 3: 构建 Prompt
	systemPrompt := buildPolishPrompt(videoType, customPrompt)
	userPrompt := fmt.Sprintf("Correct the following ASR transcription:\n\n%s", text)

	log.Printf("[OpenAILLMAdapter] System prompt: %s", systemPrompt)
	log.Printf("[OpenAILLMAdapter] User prompt length: %d", len(userPrompt))

	// 步骤 4: 调用 OpenAI API
	ctx := context.Background()
	request := openai.ChatCompletionRequest{
		Model: modelName, // 使用配置中的模型名称
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
		Temperature: 0.3,  // 中等偏低温度，平衡准确性与表达优化，避免过于机械或过于创意
		MaxTokens:   2048, // 最大输出 Token 数
	}

	log.Printf("[OpenAILLMAdapter] Sending request to API...")
	response, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		log.Printf("[OpenAILLMAdapter] ERROR: API call failed: %v", err)
		return "", fmt.Errorf("调用文本润色 API 失败: %w", err)
	}

	// 步骤 5: 提取润色结果
	if len(response.Choices) == 0 {
		log.Printf("[OpenAILLMAdapter] ERROR: No choices in response")
		return "", fmt.Errorf("API 返回结果为空")
	}

	polishedText := response.Choices[0].Message.Content
	log.Printf("[OpenAILLMAdapter] Text polishing completed successfully: result_length=%d", len(polishedText))

	return polishedText, nil
}

// Optimize 执行译文优化，提升可读性和表达一致性。
//
// 功能说明:
//   - 以固定系统 Prompt 指导模型优化翻译结果，使语句更流畅自然。
//   - 使用 go-openai 库统一调用所有兼容 OpenAI Chat Completions API 的服务。
//
// 设计决策:
//   - 使用 go-openai 库的 CreateChatCompletion 方法，自动处理 URL 拼接和错误处理。
//   - 温度设置为 0.3，在保证优化质量的同时保持翻译的准确性。
//
// 使用示例:
//
//	optimized, err := adapter.Optimize(text, "gemini-2.5-pro", apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待优化文本，不能为空。
//	modelName string: LLM 模型名称（如 "gpt-4o", "gemini-2.5-pro", "gemini-2.5-flash-lite"）。
//	apiKey string: OpenAI 或兼容代理的鉴权密钥。
//	endpoint string: 可选自定义 API 地址，留空使用默认 https://api.openai.com。
//
// 返回值说明:
//
//	string: 优化后的文本。
//	error: 调用失败或返回内容为空时出错。
//
// 错误处理说明:
//   - 401/403 表示 API 密钥无效。
//   - 429 表示请求限流，需要重试或降级。
//   - 400 表示请求参数错误。
//   - 5xx 表示服务端故障。
//
// 注意事项:
//   - 输入长度需满足模型上下文限制，调用方可在外层做分页处理。
func (o *OpenAILLMAdapter) Optimize(text, modelName, apiKey, endpoint string, optCtx *adapters.OptimizationContext) (string, error) {
	log.Printf("[OpenAILLMAdapter] Starting translation optimization: model=%s", modelName)

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待优化的文本不能为空")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API 密钥不能为空")
	}
	if modelName == "" {
		modelName = "gpt-4o" // 默认模型
		log.Printf("[OpenAILLMAdapter] WARNING: No model specified, using default: %s", modelName)
	}

	// 步骤 2: 初始化 OpenAI 客户端
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
		log.Printf("[OpenAILLMAdapter] Using custom endpoint: %s", endpoint)
	}
	client := openai.NewClientWithConfig(config)

	// 步骤 3: 构建 Prompt
	systemPrompt, userPrompt := buildOptimizationPrompts(text, optCtx)

	log.Printf("[OpenAILLMAdapter] System prompt: %s", systemPrompt)
	log.Printf("[OpenAILLMAdapter] User prompt length: %d", len(userPrompt))

	// 步骤 4: 调用 OpenAI API
	reqCtx := context.Background()
	request := openai.ChatCompletionRequest{
		Model: modelName, // 使用配置中的模型名称
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
		Temperature: 0.3, // 翻译优化使用较低温度，保证准确性
	}

	log.Printf("[OpenAILLMAdapter] Sending request to API...")
	response, err := client.CreateChatCompletion(reqCtx, request)
	if err != nil {
		log.Printf("[OpenAILLMAdapter] ERROR: API call failed: %v", err)
		return "", fmt.Errorf("调用译文优化 API 失败: %w", err)
	}

	// 步骤 5: 提取优化结果
	if len(response.Choices) == 0 {
		log.Printf("[OpenAILLMAdapter] ERROR: No choices in response")
		return "", fmt.Errorf("API 返回结果为空")
	}

	optimizedText := response.Choices[0].Message.Content
	log.Printf("[OpenAILLMAdapter] Translation optimization completed successfully: result_length=%d", len(optimizedText))

	return optimizedText, nil
}
