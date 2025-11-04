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

// GoogleTranslationAdapter Google 翻译适配器
// 实现 TranslationAdapter 接口，调用 Google Cloud Translation API
type GoogleTranslationAdapter struct {
	client *http.Client
}

// NewGoogleTranslationAdapter 创建新的 Google 翻译适配器
func NewGoogleTranslationAdapter() *GoogleTranslationAdapter {
	return &GoogleTranslationAdapter{
		client: &http.Client{
			Timeout: 60 * time.Second, // 翻译请求超时时间
		},
	}
}

// GoogleTranslateRequest Google Translation API 请求结构
type GoogleTranslateRequest struct {
	Q      []string `json:"q"`      // 待翻译的文本列表
	Source string   `json:"source"` // 源语言代码（如 "en"）
	Target string   `json:"target"` // 目标语言代码（如 "zh-CN"）
	Format string   `json:"format"` // 文本格式（"text" 或 "html"）
	Model  string   `json:"model"`  // 翻译模型（"nmt" 或 "base"）
}

// GoogleTranslateResponse Google Translation API 响应结构
type GoogleTranslateResponse struct {
	Data GoogleTranslateData `json:"data"` // 翻译数据
}

// GoogleTranslateData Google 翻译数据
type GoogleTranslateData struct {
	Translations []GoogleTranslation `json:"translations"` // 翻译结果列表
}

// GoogleTranslation Google 翻译结果
type GoogleTranslation struct {
	TranslatedText         string `json:"translatedText"`         // 翻译后的文本
	DetectedSourceLanguage string `json:"detectedSourceLanguage"` // 检测到的源语言（如果未指定源语言）
	Model                  string `json:"model"`                  // 使用的翻译模型
}

// Translate 执行文本翻译
// 参数:
//   - text: 待翻译的文本
//   - sourceLang: 源语言代码（如 "en"）
//   - targetLang: 目标语言代码（如 "zh"）
//   - videoType: 视频类型（professional_tech, casual_natural, educational_rigorous, default）
//   - apiKey: 解密后的 API 密钥（Google Cloud API Key）
//   - endpoint: 自定义端点 URL（为空则使用默认端点）
//
// 返回:
//   - translatedText: 翻译后的文本
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 400: 不支持的语言对, 5xx: 外部API服务错误）
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

// sendTranslateRequest 发送翻译 HTTP 请求
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

// normalizeLanguageCode 标准化语言代码为 BCP-47 格式
// 例如：zh → zh-CN, en → en-US
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
