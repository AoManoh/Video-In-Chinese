package translation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"video-in-chinese/ai_adaptor/internal/utils"
)

// GoogleTranslationAdapter 封装 Google Cloud Translation API 的调用，实现 TranslationAdapter 接口。
//
// 功能说明:
//   - 构造翻译请求、发送到 Google Translation API，并解析返回文本。
//
// 设计决策:
//   - 使用带 60 秒超时的 http.Client，兼顾网络抖动与响应时间。
//
// 使用示例:
//
//	adapter := NewGoogleTranslationAdapter()
//	translated, err := adapter.Translate("Hello", "en", "zh", "default", apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于维护客户端实例。
//
// 错误处理说明:
//   - Translate 方法会根据 HTTP 状态码返回具体错误。
//
// 注意事项:
//   - 调用前需准备 Google Cloud Translation API Key。
type GoogleTranslationAdapter struct {
	client *http.Client
}

// NewGoogleTranslationAdapter 创建 Google 翻译适配器实例并初始化 HTTP 客户端。
//
// 功能说明:
//   - 提供默认超时配置的适配器供业务层直接使用。
//
// 设计决策:
//   - 将 http.Client 封装在结构体中，便于测试替换 Transport。
//
// 使用示例:
//
//	adapter := NewGoogleTranslationAdapter()
//
// 返回值说明:
//
//	*GoogleTranslationAdapter: 初始化完成的适配器实例。
//
// 注意事项:
//   - 若需自定义超时，可在返回值上替换 client。
func NewGoogleTranslationAdapter() *GoogleTranslationAdapter {
	return &GoogleTranslationAdapter{
		client: &http.Client{
			Timeout: 60 * time.Second, // 翻译请求超时时间
		},
	}
}

// GoogleTranslateRequest 描述 Google Translation API 的请求体。
//
// 功能说明:
//   - 指定源语言、目标语言、文本列表与翻译模型。
//
// 注意事项:
//   - Google API 支持批量翻译，Q 可包含多条文本。
type GoogleTranslateRequest struct {
	Q      []string `json:"q"`      // 待翻译的文本列表
	Source string   `json:"source"` // 源语言代码（如 "en"）
	Target string   `json:"target"` // 目标语言代码（如 "zh-CN"）
	Format string   `json:"format"` // 文本格式（"text" 或 "html"）
	Model  string   `json:"model"`  // 翻译模型（"nmt" 或 "base"）
}

// GoogleTranslateResponse 表示 Translation API 的顶层响应结构。
type GoogleTranslateResponse struct {
	Data GoogleTranslateData `json:"data"` // 翻译数据
}

// GoogleTranslateData 承载翻译结果数组。
type GoogleTranslateData struct {
	Translations []GoogleTranslation `json:"translations"` // 翻译结果列表
}

// GoogleTranslation 描述单条翻译结果，包括翻译文本和检测到的源语言。
type GoogleTranslation struct {
	TranslatedText         string `json:"translatedText"`         // 翻译后的文本
	DetectedSourceLanguage string `json:"detectedSourceLanguage"` // 检测到的源语言（如果未指定源语言）
	Model                  string `json:"model"`                  // 使用的翻译模型
}

// Translate 执行一次文本翻译并返回翻译后的文本。
//
// 功能说明:
//   - 对输入文本调用 Google Translation API，按需求选择模型并处理重试。
//
// 设计决策:
//   - 使用 v2 接口的 `nmt` 模型以获得更佳翻译质量。
//   - videoType 参数预留给后续扩展（例如 Glossary），当前不直接使用。
//
// 使用示例:
//
//	translated, err := adapter.Translate("Hello", "en", "zh", "default", apiKey, "")
//
// 参数说明:
//
//	text string: 待翻译文本，不能为空。
//	sourceLang string: 源语言代码，若为空将由 API 自动检测。
//	targetLang string: 目标语言代码，必填。
//	videoType string: 视频语气标签（预留扩展）。
//	apiKey string: Google Cloud API Key。
//	endpoint string: 可选自定义端点，空字符串使用默认 URL。
//
// 返回值说明:
//
//	string: 翻译后的文本。
//	error: 请求失败或解析错误时返回。
//
// 错误处理说明:
//   - HTTP 401/403 表示密钥无效，429 表示配额不足，400 表示语言对不受支持，5xx 表示供应商故障。
//   - JSON 解析失败时，将携带响应体以便调试。
//
// 注意事项:
//   - API 有长度限制，超长文本需调用方拆分处理。
//   - 建议调用方结合重试与熔断策略应对配额限制。
func (g *GoogleTranslationAdapter) Translate(text, sourceLang, targetLang, videoType, apiKey, endpoint string) (string, error) {
	log.Printf("[GoogleTranslationAdapter] Starting translation: source=%s, target=%s, video_type=%s", sourceLang, targetLang, videoType)

	// 步骤 1: 验证输入参数
	if text == "" {
		return "", fmt.Errorf("待翻译的文本不能为空")
	}

	// 步骤 2: 转换语言代码（Google Translation API 使用 BCP-47 格式）
	// 例如：zh → zh-CN, en → en-US
	sourceLanguage := normalizeLanguageCode(sourceLang)
	targetLanguage := normalizeLanguageCode(targetLang)

	// 步骤 3: 构建 API 请求
	// 注意：videoType 参数在 Google Translation API 中不直接使用
	// 可以在未来扩展为使用 Google Cloud Translation Advanced API 的 Glossary 功能
	requestBody := GoogleTranslateRequest{
		Q:      []string{text},
		Source: sourceLanguage,
		Target: targetLanguage,
		Format: "text",
		Model:  "nmt", // 使用神经机器翻译模型（Neural Machine Translation）
	}

	// 步骤 4: 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 步骤 5: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 使用默认端点（Google Cloud Translation API v2）
		apiEndpoint = "https://translation.googleapis.com/language/translate/v2"
	}

	// 步骤 6: 发送 HTTP POST 请求（带重试逻辑）
	var response *GoogleTranslateResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[GoogleTranslationAdapter] Retrying translation request (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = g.sendTranslateRequest(apiEndpoint, requestJSON, apiKey)
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

	// 步骤 7: 提取翻译结果
	if len(response.Data.Translations) == 0 {
		return "", fmt.Errorf("翻译响应中没有结果")
	}

	translatedText := response.Data.Translations[0].TranslatedText

	log.Printf("[GoogleTranslationAdapter] Translation completed successfully")
	return translatedText, nil
}

// sendTranslateRequest 向 Google Translation API 发送请求并解析响应。
//
// 功能说明:
//   - 构造 HTTP POST 请求、校验状态码并解码响应为 GoogleTranslateResponse。
//
// 参数说明:
//
//	endpoint string: API 端点。
//	requestJSON []byte: 序列化后的请求体。
//	apiKey string: Google Cloud API Key。
//
// 返回值说明:
//
//	*GoogleTranslateResponse: 成功时的翻译结果容器。
//	error: 网络请求失败或响应异常时返回。
//
// 注意事项:
//   - 429、401/403、400 与 5xx 等状态码会转化为带响应体的错误。
func (g *GoogleTranslationAdapter) sendTranslateRequest(endpoint string, requestJSON []byte, apiKey string) (*GoogleTranslateResponse, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey) // Google Cloud API 认证方式

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
		return nil, fmt.Errorf("不支持的语言对或请求参数错误 (HTTP 400): %s", string(responseBody))
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("外部 API 服务错误 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP 请求失败 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}

	// 解析 JSON 响应
	var translateResponse GoogleTranslateResponse
	if err := json.Unmarshal(responseBody, &translateResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	return &translateResponse, nil
}

// normalizeLanguageCode 将语言代码标准化为 BCP-47 格式，便于与 Google API 对齐。
//
// 功能说明:
//   - 针对常见简写进行转换，其余语言保持原状。
//
// 注意事项:
//   - 后续如需支持更多地区，可在此扩展映射表。
func normalizeLanguageCode(langCode string) string {
	switch langCode {
	case "zh":
		return "zh-CN" // 简体中文
	case "en":
		return "en-US" // 美式英语
	case "ja":
		return "ja" // 日语
	case "ko":
		return "ko" // 韩语
	case "fr":
		return "fr" // 法语
	case "de":
		return "de" // 德语
	case "es":
		return "es" // 西班牙语
	case "ru":
		return "ru" // 俄语
	default:
		return langCode // 其他语言代码保持不变
	}
}

// contains 检查字符串是否包含子串，兼容简单前后缀匹配场景。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

// containsSubstring 辅助 contains 做逐字符匹配。
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
