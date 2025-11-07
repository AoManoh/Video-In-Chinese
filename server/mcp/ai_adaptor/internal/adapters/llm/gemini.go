package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"video-in-chinese/server/mcp/ai_adaptor/internal/utils"
)

// GeminiLLMAdapter 封装 Google Gemini API，实现 LLMAdapter 接口。
//
// 功能说明:
//   - 提供文本润色与译文优化能力，支持 Gemini 官方接口。
//
// 设计决策:
//   - 采用 120 秒超时的 http.Client，以满足长上下文生成。
//
// 使用示例:
//
//	adapter := NewGeminiLLMAdapter()
//	optimized, err := adapter.Optimize(text, apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体实例通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于持有 HTTP 客户端。
//
// 错误处理说明:
//   - 由 Polish/Optimize 方法统一处理 HTTP 状态码错误。
//
// 注意事项:
//   - endpoint 可指向自定义路由以支持代理或多区域部署。
type GeminiLLMAdapter struct {
	client *http.Client
}

// NewGeminiLLMAdapter 创建 Gemini LLM 适配器实例并初始化 HTTP 客户端。
//
// 功能说明:
//   - 提供默认超时配置，开箱即用连接 Gemini API。
//
// 设计决策:
//   - 封装 http.Client，方便后续替换 Transport 或超时设置。
//
// 使用示例:
//
//	adapter := NewGeminiLLMAdapter()
//
// 返回值说明:
//
//	*GeminiLLMAdapter: 已初始化的适配器实例。
//
// 注意事项:
//   - 若需自定义代理，可替换返回值的 client 字段。
func NewGeminiLLMAdapter() *GeminiLLMAdapter {
	return &GeminiLLMAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // LLM 请求可能需要较长时间
		},
	}
}

// GeminiRequest Gemini API 请求结构
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`                   // 对话内容列表
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"` // 生成配置
	SafetySettings   []GeminiSafetySetting   `json:"safetySettings,omitempty"`   // 安全设置
}

// GeminiContent Gemini 对话内容
type GeminiContent struct {
	Role  string       `json:"role"`  // 角色（"user" 或 "model"）
	Parts []GeminiPart `json:"parts"` // 内容部分列表
}

// GeminiPart Gemini 内容部分
type GeminiPart struct {
	Text string `json:"text"` // 文本内容
}

// GeminiGenerationConfig Gemini 生成配置
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`     // 温度（0.0-1.0）
	TopP            float64 `json:"topP,omitempty"`            // Top-P 采样
	TopK            int     `json:"topK,omitempty"`            // Top-K 采样
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"` // 最大输出 Token 数
}

// GeminiSafetySetting Gemini 安全设置
type GeminiSafetySetting struct {
	Category  string `json:"category"`  // 安全类别
	Threshold string `json:"threshold"` // 阈值
}

// GeminiResponse Gemini API 响应结构
type GeminiResponse struct {
	Candidates     []GeminiCandidate     `json:"candidates"`               // 候选结果列表
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"` // Prompt 反馈
}

// GeminiCandidate Gemini 候选结果
type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`       // 生成的内容
	FinishReason  string               `json:"finishReason"`  // 完成原因
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"` // 安全评级
}

// GeminiSafetyRating Gemini 安全评级
type GeminiSafetyRating struct {
	Category    string `json:"category"`    // 安全类别
	Probability string `json:"probability"` // 概率
}

// GeminiPromptFeedback Gemini Prompt 反馈
type GeminiPromptFeedback struct {
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"` // 安全评级
}

// Polish 调用 Gemini 模型执行文本润色并返回润色后的内容。
//
// 功能说明:
//   - 根据视频类型生成系统提示词，组合自定义 Prompt，调用 Gemini API 生成润色结果。
//
// 设计决策:
//   - 通过 callGeminiAPI 统一处理请求构造、重试与错误分类。
//
// 使用示例:
//
//	polished, err := adapter.Polish(text, "casual_natural", "", apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待润色文本，不能为空。
//	videoType string: 视频语气标签，用于选择默认 Prompt。
//	customPrompt string: 可选自定义提示词，空字符串使用默认模版。
//	apiKey string: Google Cloud API Key。
//	endpoint string: 可选自定义端点，留空使用官方 URL。
//
// 返回值说明:
//
//	string: 润色后的文本。
//	error: 调用失败或返回内容为空时出错。
//
// 错误处理说明:
//   - 将 401/403、429、400、5xx 等错误封装为具上下文的信息。
//
// 注意事项:
//   - Gemini API 需要在 URL 中附带 key，调用方应妥善保管凭证。
func (g *GeminiLLMAdapter) Polish(text, videoType, customPrompt, modelName, apiKey, endpoint string) (string, error) {
	log.Printf("[GeminiLLMAdapter] Starting text polishing: video_type=%s, model=%s", videoType, modelName)

	// 注意：Gemini 适配器忽略 modelName 参数，因为模型在 URL 中指定
	// 如果需要支持不同模型，应在 endpoint 中包含模型名称

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待润色的文本不能为空")
	}

	// 步骤 2: 构建 Prompt
	systemPrompt := buildPolishPrompt(videoType, customPrompt)
	userPrompt := fmt.Sprintf("请润色以下文本：\n\n%s", text)

	// 步骤 3: 调用 Gemini API
	polishedText, err := g.callGeminiAPI(systemPrompt, userPrompt, apiKey, endpoint)
	if err != nil {
		return "", fmt.Errorf("调用 Gemini API 失败: %w", err)
	}

	log.Printf("[GeminiLLMAdapter] Text polishing completed successfully")
	return polishedText, nil
}

// Optimize 调用 Gemini 模型对文本执行语义优化。
//
// 功能说明:
//   - 通过固定系统 Prompt 指导模型对翻译结果做整体润色与逻辑整理。
//
// 使用示例:
//
//	optimized, err := adapter.Optimize(text, apiKey, endpoint)
//
// 参数说明:
//
//	text string: 待优化文本，不能为空。
//	apiKey string: Google Cloud API Key。
//	endpoint string: 可选自定义端点，留空使用官方 URL。
//
// 返回值说明:
//
//	string: 优化后的文本。
//	error: 调用失败或返回内容为空时出错。
//
// 错误处理说明:
//   - 401/403、429、5xx 等错误会被归类并返回，方便上层处理。
//
// 注意事项:
//   - 任务需满足 Gemini 模型的 Token 限制，建议调用方控制输入长度。
func (g *GeminiLLMAdapter) Optimize(text, modelName, apiKey, endpoint string) (string, error) {
	log.Printf("[GeminiLLMAdapter] Starting translation optimization: model=%s", modelName)

	// 注意：Gemini 适配器忽略 modelName 参数，因为模型在 URL 中指定
	// 如果需要支持不同模型，应在 endpoint 中包含模型名称

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待优化的文本不能为空")
	}

	// 步骤 2: 构建 Prompt
	systemPrompt := "你是一位专业的翻译优化专家。请优化以下翻译文本，使其更加流畅、自然、符合中文表达习惯。保持原意不变，只优化表达方式。"
	userPrompt := fmt.Sprintf("请优化以下翻译文本：\n\n%s", text)

	// 步骤 3: 调用 Gemini API
	optimizedText, err := g.callGeminiAPI(systemPrompt, userPrompt, apiKey, endpoint)
	if err != nil {
		return "", fmt.Errorf("调用 Gemini API 失败: %w", err)
	}

	log.Printf("[GeminiLLMAdapter] Translation optimization completed successfully")
	return optimizedText, nil
}

// callGeminiAPI 调用 Gemini API 并返回首个候选文本。
//
// 功能说明:
//   - 构造请求体、补全端点、执行带重试的请求，并提取候选文本。
//
// 参数说明:
//
//	systemPrompt string: 系统提示语。
//	userPrompt string: 用户输入文本。
//	apiKey string: Google Cloud API Key。
//	endpoint string: 可选自定义端点。
//
// 返回值说明:
//
//	string: Gemini 模型返回的文本。
//	error: 当请求失败或响应为空时返回。
//
// 注意事项:
//   - URL 可能需包含 `key` 参数，调用方需确保凭证安全。
func (g *GeminiLLMAdapter) callGeminiAPI(systemPrompt, userPrompt, apiKey, endpoint string) (string, error) {
	// 步骤 1: 构建请求体
	requestBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{Text: systemPrompt + "\n\n" + userPrompt},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     0.3,  // 中等偏低温度，平衡准确性与表达优化，避免过于机械或过于创意
			TopP:            0.9,  // Top-P 采样
			TopK:            40,   // Top-K 采样
			MaxOutputTokens: 2048, // 最大输出 Token 数
		},
		SafetySettings: []GeminiSafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	// 步骤 2: 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 步骤 3: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 使用默认端点（Gemini 1.5 Flash 模型）
		// 注意：需要在 URL 中包含 API Key
		apiEndpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
	}

	// 步骤 4: 发送 HTTP POST 请求（带重试逻辑）
	var response *GeminiResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[GeminiLLMAdapter] Retrying Gemini API request (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = g.sendGeminiRequest(apiEndpoint, requestJSON)
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
	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("Gemini API 响应中没有候选结果")
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini API 响应中没有生成的文本")
	}

	generatedText := candidate.Content.Parts[0].Text

	return generatedText, nil
}

// sendGeminiRequest 发送 Gemini HTTP 请求并解析响应。
//
// 功能说明:
//   - 构造 POST 请求、设置 JSON 头部、检查状态码并解码响应。
//
// 参数说明:
//
//	endpoint string: 完整 Gemini API URL，应包含 key 参数。
//	requestJSON []byte: 序列化后的请求体。
//
// 返回值说明:
//
//	*GeminiResponse: 成功时的响应对象。
//	error: HTTP 请求失败或响应异常时返回。
//
// 注意事项:
//   - 对 401/403、429、400、5xx 错误提供详细提示以便上层处理。
func (g *GeminiLLMAdapter) sendGeminiRequest(endpoint string, requestJSON []byte) (*GeminiResponse, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := g.client.Do(req)
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
	var geminiResponse GeminiResponse
	if err := json.Unmarshal(responseBody, &geminiResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	return &geminiResponse, nil
}

// buildPolishPrompt 根据视频类型构建文本润色 Prompt，支持自定义覆盖。
//
// 功能说明:
//   - 若提供 customPrompt 则直接返回，否则按视频类型返回默认模版。
func buildPolishPrompt(videoType, customPrompt string) string {
	// 如果用户提供了自定义 Prompt，则使用自定义 Prompt
	if customPrompt != "" {
		return customPrompt
	}

	// 根据视频类型构建默认 Prompt
	basePrompt := "你是一位专业的文本润色专家。请润色以下文本，使其更加流畅、自然、符合表达习惯。保持原意不变，只优化表达方式。\n\n重要：请直接返回润色后的文本，不要添加任何解释、说明或多个方案。"

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

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

// containsSubstring 辅助函数：检查字符串是否包含子串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
