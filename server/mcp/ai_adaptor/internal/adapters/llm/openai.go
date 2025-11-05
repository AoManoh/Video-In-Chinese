package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"video-in-chinese/ai_adaptor/internal/utils"
)

// OpenAILLMAdapter 封装 OpenAI Chat Completions API，实现 LLMAdapter 接口。
//
// 功能说明:
//   - 提供文本润色与脚本优化的统一入口，支持自定义 endpoint 与第三方中转服务。
//
// 设计决策:
//   - 使用 120 秒超时的 http.Client，以适配长输入和复杂生成任务。
//
// 使用示例:
//
//	adapter := NewOpenAILLMAdapter()
//	polished, err := adapter.Polish(text, "professional_tech", "", apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体实例通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于持有 HTTP 客户端。
//
// 错误处理说明:
//   - 由 Polish/Optimize 方法根据 HTTP 状态码分类错误。
//
// 注意事项:
//   - endpoint 可指向代理服务（one-api、gemini-balance 等）以满足不同部署需求。
type OpenAILLMAdapter struct {
	client *http.Client
}

// NewOpenAILLMAdapter 创建 OpenAI LLM 适配器实例并初始化 HTTP 客户端。
//
// 功能说明:
//   - 提供默认超时配置的适配器供业务层直接使用。
//
// 设计决策:
//   - 封装 http.Client，便于后续依赖注入或自定义 Transport。
//
// 使用示例:
//
//	adapter := NewOpenAILLMAdapter()
//
// 返回值说明:
//
//	*OpenAILLMAdapter: 已初始化的适配器实例。
//
// 注意事项:
//   - 若需自定义超时，可替换返回值的 client 字段。
func NewOpenAILLMAdapter() *OpenAILLMAdapter {
	return &OpenAILLMAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // LLM 请求可能需要较长时间
		},
	}
}

// OpenAIChatRequest OpenAI Chat Completions API 请求结构
type OpenAIChatRequest struct {
	Model       string              `json:"model"`       // 模型名称（如 "gpt-4o", "gpt-3.5-turbo"）
	Messages    []OpenAIChatMessage `json:"messages"`    // 对话消息列表
	Temperature float64             `json:"temperature"` // 温度（0.0-2.0）
	MaxTokens   int                 `json:"max_tokens"`  // 最大输出 Token 数
	TopP        float64             `json:"top_p"`       // Top-P 采样
}

// OpenAIChatMessage OpenAI 对话消息
type OpenAIChatMessage struct {
	Role    string `json:"role"`    // 角色（"system", "user", "assistant"）
	Content string `json:"content"` // 消息内容
}

// OpenAIChatResponse OpenAI Chat Completions API 响应结构
type OpenAIChatResponse struct {
	ID      string             `json:"id"`      // 响应 ID
	Object  string             `json:"object"`  // 对象类型（"chat.completion"）
	Created int64              `json:"created"` // 创建时间戳
	Model   string             `json:"model"`   // 使用的模型
	Choices []OpenAIChatChoice `json:"choices"` // 候选结果列表
	Usage   OpenAIChatUsage    `json:"usage"`   // Token 使用情况
}

// OpenAIChatChoice OpenAI 候选结果
type OpenAIChatChoice struct {
	Index        int               `json:"index"`         // 候选索引
	Message      OpenAIChatMessage `json:"message"`       // 生成的消息
	FinishReason string            `json:"finish_reason"` // 完成原因（"stop", "length", "content_filter"）
}

// OpenAIChatUsage OpenAI Token 使用情况
type OpenAIChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // Prompt Token 数
	CompletionTokens int `json:"completion_tokens"` // 生成 Token 数
	TotalTokens      int `json:"total_tokens"`      // 总 Token 数
}

// Polish 执行文本润色并返回润色后的内容。
//
// 功能说明:
//   - 根据视频类型生成系统提示词，组合自定义 Prompt，调用 OpenAI 模型润色文本。
//
// 设计决策:
//   - 复用 callOpenAIAPI 以统一重试与错误处理逻辑。
//
// 使用示例:
//
//	polished, err := adapter.Polish(text, "professional_tech", "", apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待润色文本，不能为空。
//	videoType string: 视频语气标签，决定系统 Prompt 语气。
//	customPrompt string: 可选自定义提示词，空字符串使用默认模版。
//	apiKey string: OpenAI 或兼容代理的鉴权密钥。
//	endpoint string: 可选自定义 API 地址，留空使用默认 https://api.openai.com。
//
// 返回值说明:
//
//	string: 润色后的文本。
//	error: 调用失败或返回内容为空时出错。
//
// 错误处理说明:
//   - 将 HTTP 401/403、429、400、5xx 等错误映射为具备上下文的错误信息。
//
// 注意事项:
//   - 长文本可能触发 Token 限制，调用方应做好截断或重试策略。
func (o *OpenAILLMAdapter) Polish(text, videoType, customPrompt, apiKey, endpoint string) (string, error) {
	log.Printf("[OpenAILLMAdapter] Starting text polishing: video_type=%s", videoType)

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待润色的文本不能为空")
	}

	// 步骤 2: 构建 Prompt
	systemPrompt := buildPolishPrompt(videoType, customPrompt)
	userPrompt := fmt.Sprintf("请润色以下文本：\n\n%s", text)

	// 步骤 3: 调用 OpenAI API
	polishedText, err := o.callOpenAIAPI(systemPrompt, userPrompt, apiKey, endpoint)
	if err != nil {
		return "", fmt.Errorf("调用 OpenAI API 失败: %w", err)
	}

	log.Printf("[OpenAILLMAdapter] Text polishing completed successfully")
	return polishedText, nil
}

// Optimize 执行译文优化，提升可读性和表达一致性。
//
// 功能说明:
//   - 以固定系统 Prompt 指导模型优化翻译结果，使语句更流畅自然。
//
// 使用示例:
//
//	optimized, err := adapter.Optimize(text, apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待优化文本，不能为空。
//	apiKey string: OpenAI 或兼容代理的鉴权密钥。
//	endpoint string: 可选自定义 API 地址，留空使用默认 https://api.openai.com。
//
// 返回值说明:
//
//	string: 优化后的文本。
//	error: 调用失败或返回内容为空时出错。
//
// 错误处理说明:
//   - 401/403、429、5xx 等错误会被统一封装为具上下文的错误。
//
// 注意事项:
//   - 输入长度需满足模型上下文限制，调用方可在外层做分页处理。
func (o *OpenAILLMAdapter) Optimize(text, apiKey, endpoint string) (string, error) {
	log.Printf("[OpenAILLMAdapter] Starting translation optimization")

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待优化的文本不能为空")
	}

	// 步骤 2: 构建 Prompt
	systemPrompt := "你是一位专业的翻译优化专家。请优化以下翻译文本，使其更加流畅、自然、符合中文表达习惯。保持原意不变，只优化表达方式。"
	userPrompt := fmt.Sprintf("请优化以下翻译文本：\n\n%s", text)

	// 步骤 3: 调用 OpenAI API
	optimizedText, err := o.callOpenAIAPI(systemPrompt, userPrompt, apiKey, endpoint)
	if err != nil {
		return "", fmt.Errorf("调用 OpenAI API 失败: %w", err)
	}

	log.Printf("[OpenAILLMAdapter] Translation optimization completed successfully")
	return optimizedText, nil
}

// callOpenAIAPI 调用 OpenAI Chat Completions API 并返回生成文本。
//
// 功能说明:
//   - 构造对话请求、拼接 endpoint、执行带重试的请求，并提取首个候选内容。
//
// 设计决策:
//   - 重试逻辑集中于此，Polish/Optimize 共享统一错误处理。
//
// 参数说明:
//
//	systemPrompt string: 系统角色提示语。
//	userPrompt string: 用户输入文本。
//	apiKey string: OpenAI 或代理鉴权密钥。
//	endpoint string: 可选自定义 API 地址。
//
// 返回值说明:
//
//	string: 模型生成的文本。
//	error: 当请求失败或响应为空时返回。
//
// 注意事项:
//   - 对于不可重试错误（401/403、429、400）将立即返回，避免无畏重试。
func (o *OpenAILLMAdapter) callOpenAIAPI(systemPrompt, userPrompt, apiKey, endpoint string) (string, error) {
	// 步骤 1: 构建请求体
	requestBody := OpenAIChatRequest{
		Model: "gpt-4o", // TODO: 从配置中读取模型名称（Phase 4 后期实现）
		Messages: []OpenAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,  // 适中的创造性
		MaxTokens:   2048, // 最大输出 Token 数
		TopP:        0.9,  // Top-P 采样
	}

	// 步骤 2: 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 步骤 3: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 使用默认端点（OpenAI 官方 API）
		apiEndpoint = "https://api.openai.com"
	}

	// 移除末尾的斜杠（如果有）
	apiEndpoint = strings.TrimSuffix(apiEndpoint, "/")

	// 拼接完整的 API 路径
	fullEndpoint := apiEndpoint + "/v1/chat/completions"

	// 步骤 4: 发送 HTTP POST 请求（带重试逻辑）
	var response *OpenAIChatResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[OpenAILLMAdapter] Retrying OpenAI API request (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = o.sendOpenAIRequest(fullEndpoint, requestJSON, apiKey)
		if lastErr == nil {
			break // 请求成功，退出重试循环
		}

		// 检查是否为不可重试的错误（401, 429, 400）
		if utils.IsNonRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	// 步骤 5: 提取生成的文本
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("OpenAI API 响应中没有候选结果")
	}

	generatedText := response.Choices[0].Message.Content

	return generatedText, nil
}

// sendOpenAIRequest 发送 OpenAI HTTP 请求并解析响应为结构体。
//
// 功能说明:
//   - 设置认证头，校验 HTTP 状态码，将成功响应解码为 OpenAIChatResponse。
//
// 参数说明:
//
//	endpoint string: 完整 API 路径。
//	requestJSON []byte: 序列化后的请求体。
//	apiKey string: OpenAI 或代理鉴权密钥。
//
// 返回值说明:
//
//	*OpenAIChatResponse: 成功时返回的响应对象。
//	error: 网络请求失败或响应异常时返回。
//
// 注意事项:
//   - 将 401/403、429、400、5xx 等错误封装为含响应体的错误信息，便于上层排查。
func (o *OpenAILLMAdapter) sendOpenAIRequest(endpoint string, requestJSON []byte, apiKey string) (*OpenAIChatResponse, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey) // OpenAI 认证方式

	// 发送请求
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("API 密钥无效 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("API 配额不足 (HTTP 429): %s", string(responseBody))
	}
	if resp.StatusCode == 400 {
		return nil, fmt.Errorf("Prompt 格式错误或请求参数错误 (HTTP 400): %s", string(responseBody))
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("外部 API 服务错误 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP 请求失败 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}

	// 解析 JSON 响应
	var chatResponse OpenAIChatResponse
	if err := json.Unmarshal(responseBody, &chatResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	return &chatResponse, nil
}
