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
)

// OpenAILLMAdapter OpenAI 格式 LLM 适配器
// 实现 LLMAdapter 接口，调用 OpenAI Chat Completions API
// 支持自定义 endpoint，兼容第三方中转服务（如 gemini-balance、one-api、new-api 等）
type OpenAILLMAdapter struct {
	client *http.Client
}

// NewOpenAILLMAdapter 创建新的 OpenAI LLM 适配器
func NewOpenAILLMAdapter() *OpenAILLMAdapter {
	return &OpenAILLMAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // LLM 请求可能需要较长时间
		},
	}
}

// OpenAIChatRequest OpenAI Chat Completions API 请求结构
type OpenAIChatRequest struct {
	Model       string                `json:"model"`       // 模型名称（如 "gpt-4o", "gpt-3.5-turbo"）
	Messages    []OpenAIChatMessage   `json:"messages"`    // 对话消息列表
	Temperature float64               `json:"temperature"` // 温度（0.0-2.0）
	MaxTokens   int                   `json:"max_tokens"`  // 最大输出 Token 数
	TopP        float64               `json:"top_p"`       // Top-P 采样
}

// OpenAIChatMessage OpenAI 对话消息
type OpenAIChatMessage struct {
	Role    string `json:"role"`    // 角色（"system", "user", "assistant"）
	Content string `json:"content"` // 消息内容
}

// OpenAIChatResponse OpenAI Chat Completions API 响应结构
type OpenAIChatResponse struct {
	ID      string                 `json:"id"`      // 响应 ID
	Object  string                 `json:"object"`  // 对象类型（"chat.completion"）
	Created int64                  `json:"created"` // 创建时间戳
	Model   string                 `json:"model"`   // 使用的模型
	Choices []OpenAIChatChoice     `json:"choices"` // 候选结果列表
	Usage   OpenAIChatUsage        `json:"usage"`   // Token 使用情况
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

// Polish 执行文本润色
// 参数:
//   - text: 待处理的文本
//   - videoType: 视频类型（professional_tech, casual_natural, educational_rigorous, default）
//   - customPrompt: 用户自定义 Prompt（可选）
//   - apiKey: 解密后的 API 密钥（OpenAI API Key 或第三方中转服务 API Key）
//   - endpoint: 自定义端点 URL（为空则使用默认端点 https://api.openai.com）
// 返回:
//   - polishedText: 润色后的文本
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 400: Prompt格式错误, 5xx: 外部API服务错误）
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

// Optimize 执行译文优化
// 参数:
//   - text: 待优化的文本
//   - apiKey: 解密后的 API 密钥（OpenAI API Key 或第三方中转服务 API Key）
//   - endpoint: 自定义端点 URL（为空则使用默认端点 https://api.openai.com）
// 返回:
//   - optimizedText: 优化后的文本
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 5xx: 外部API服务错误）
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

// callOpenAIAPI 调用 OpenAI Chat Completions API
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
		if isNonRetryableError(lastErr) {
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

// sendOpenAIRequest 发送 OpenAI HTTP 请求
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

// buildPolishPrompt 构建文本润色 Prompt
func buildPolishPrompt(videoType, customPrompt string) string {
	// 如果用户提供了自定义 Prompt，则使用自定义 Prompt
	if customPrompt != "" {
		return customPrompt
	}

	// 根据视频类型构建默认 Prompt
	basePrompt := "你是一位专业的文本润色专家。请润色以下文本，使其更加流畅、自然、符合表达习惯。保持原意不变，只优化表达方式。"

	switch videoType {
	case "professional_tech":
		return basePrompt + "\n\n特别要求：保持专业术语的准确性，使用正式的技术文档风格。"
	case "casual_natural":
		return basePrompt + "\n\n特别要求：使用轻松、自然的口语化表达，避免过于正式的用词。"
	case "educational_rigorous":
		return basePrompt + "\n\n特别要求：保持严谨的学术风格，确保逻辑清晰、表达准确。"
	default:
		return basePrompt
	}
}

// isNonRetryableError 判断是否为不可重试的错误
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	// 401/403: API 密钥无效
	if contains(errMsg, "API 密钥无效") || contains(errMsg, "HTTP 401") || contains(errMsg, "HTTP 403") {
		return true
	}
	// 429: API 配额不足
	if contains(errMsg, "API 配额不足") || contains(errMsg, "HTTP 429") {
		return true
	}
	// 400: Prompt 格式错误
	if contains(errMsg, "Prompt 格式错误") || contains(errMsg, "HTTP 400") {
		return true
	}
	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

