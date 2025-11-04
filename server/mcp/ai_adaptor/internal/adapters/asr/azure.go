package asr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	pb "video-in-chinese/ai_adaptor/proto"
)

// AzureASRAdapter Azure 语音识别适配器
// 实现 ASRAdapter 接口，调用 Azure Speech Service API
type AzureASRAdapter struct {
	client *http.Client
}

// NewAzureASRAdapter 创建新的 Azure ASR 适配器
func NewAzureASRAdapter() *AzureASRAdapter {
	return &AzureASRAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 语音识别可能需要较长时间
		},
	}
}

// AzureASRRequest Azure Speech API 请求结构（Batch Transcription）
type AzureASRRequest struct {
	ContentURLs        []string          `json:"contentUrls"`        // 音频文件 URL 列表
	Locale             string            `json:"locale"`             // 语言区域（如 "en-US", "zh-CN"）
	DisplayName        string            `json:"displayName"`        // 转录任务显示名称
	Properties         AzureASRProperties `json:"properties"`         // 转录属性
	DiarizationEnabled bool              `json:"diarizationEnabled"` // 是否启用说话人分离
}

// AzureASRProperties Azure Speech API 转录属性
type AzureASRProperties struct {
	PunctuationMode        string `json:"punctuationMode"`        // 标点符号模式（"DictatedAndAutomatic"）
	ProfanityFilterMode    string `json:"profanityFilterMode"`    // 脏话过滤模式（"None"）
	WordLevelTimestampsEnabled bool `json:"wordLevelTimestampsEnabled"` // 是否启用词级别时间戳
}

// AzureASRResponse Azure Speech API 响应结构（Batch Transcription）
type AzureASRResponse struct {
	Self              string                 `json:"self"`              // 转录任务 URL
	Status            string                 `json:"status"`            // 任务状态（"NotStarted", "Running", "Succeeded", "Failed"）
	CreatedDateTime   string                 `json:"createdDateTime"`   // 创建时间
	LastActionDateTime string                `json:"lastActionDateTime"` // 最后操作时间
	Links             AzureASRLinks          `json:"links"`             // 相关链接
}

// AzureASRLinks Azure Speech API 链接结构
type AzureASRLinks struct {
	Files string `json:"files"` // 结果文件列表 URL
}

// AzureASRFilesResponse Azure Speech API 文件列表响应
type AzureASRFilesResponse struct {
	Values []AzureASRFile `json:"values"` // 文件列表
}

// AzureASRFile Azure Speech API 文件结构
type AzureASRFile struct {
	Kind        string `json:"kind"`        // 文件类型（"Transcription"）
	Name        string `json:"name"`        // 文件名
	ContentURL  string `json:"contentUrl"`  // 文件下载 URL
}

// AzureTranscriptionResult Azure Speech API 转录结果
type AzureTranscriptionResult struct {
	CombinedRecognizedPhrases []AzureCombinedPhrase `json:"combinedRecognizedPhrases"` // 合并的识别短语
	RecognizedPhrases         []AzureRecognizedPhrase `json:"recognizedPhrases"`       // 识别的短语列表
}

// AzureCombinedPhrase Azure 合并短语
type AzureCombinedPhrase struct {
	Channel int    `json:"channel"` // 音频通道
	Lexical string `json:"lexical"` // 词汇形式
	Display string `json:"display"` // 显示形式
}

// AzureRecognizedPhrase Azure 识别短语
type AzureRecognizedPhrase struct {
	Channel        int                `json:"channel"`        // 音频通道
	OffsetInTicks  int64              `json:"offsetInTicks"`  // 开始时间（100 纳秒为单位）
	DurationInTicks int64             `json:"durationInTicks"` // 持续时间（100 纳秒为单位）
	NBest          []AzureNBestResult `json:"nBest"`          // N-Best 结果列表
	Speaker        int                `json:"speaker"`        // 说话人 ID（如果启用了说话人分离）
}

// AzureNBestResult Azure N-Best 结果
type AzureNBestResult struct {
	Confidence float64 `json:"confidence"` // 置信度
	Lexical    string  `json:"lexical"`    // 词汇形式
	Display    string  `json:"display"`    // 显示形式
}

// ASR 执行语音识别，返回说话人列表
// 参数:
//   - audioPath: 音频文件的本地路径
//   - apiKey: 解密后的 API 密钥（Azure Speech Service 订阅密钥）
//   - endpoint: 自定义端点 URL（为空则使用默认端点）
// 返回:
//   - speakers: 说话人列表，包含句子级时间戳和文本
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 5xx: 外部API服务错误）
func (a *AzureASRAdapter) ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error) {
	log.Printf("[AzureASRAdapter] Starting ASR: audio_path=%s", audioPath)

	// 步骤 1: 验证音频文件是否存在
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("音频文件不存在: %s", audioPath)
	}

	// 步骤 2: 上传音频文件到临时存储（Azure Blob Storage 或其他公网可访问的存储）
	// TODO: 实现音频文件上传到 Azure Blob Storage，获取公网 URL（Phase 4 后期实现）
	// 临时方案：使用本地文件路径（实际生产环境需要上传到 Azure Blob Storage）
	audioURL := audioPath // 临时占位符，Phase 4 后期实现 Azure Blob Storage 上传

	// 步骤 3: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 使用默认端点（Azure Speech Service - Batch Transcription）
		// 注意：需要替换 {region} 为实际的 Azure 区域（如 "eastus", "westus2"）
		// TODO: 从配置或 API 密钥中提取区域信息（Phase 4 后期实现）
		apiEndpoint = "https://eastus.api.cognitive.microsoft.com/speechtotext/v3.1/transcriptions"
	}

	// 步骤 4: 创建批量转录任务
	transcriptionURL, err := a.createTranscription(apiEndpoint, audioURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("创建转录任务失败: %w", err)
	}

	// 步骤 5: 轮询转录任务状态（等待完成）
	filesURL, err := a.pollTranscriptionStatus(transcriptionURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("轮询转录任务状态失败: %w", err)
	}

	// 步骤 6: 获取转录结果文件
	transcriptionResult, err := a.getTranscriptionResult(filesURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("获取转录结果失败: %w", err)
	}

	// 步骤 7: 解析转录结果，转换为 Speaker 列表
	speakers, err := a.parseTranscriptionResult(transcriptionResult)
	if err != nil {
		return nil, fmt.Errorf("解析转录结果失败: %w", err)
	}

	log.Printf("[AzureASRAdapter] ASR completed successfully: %d speakers found", len(speakers))
	return speakers, nil
}

// createTranscription 创建批量转录任务
func (a *AzureASRAdapter) createTranscription(endpoint, audioURL, apiKey string) (string, error) {
	// 构建请求体
	requestBody := AzureASRRequest{
		ContentURLs: []string{audioURL},
		Locale:      "zh-CN", // TODO: 从配置中读取语言区域（Phase 4 后期实现）
		DisplayName: "Video Translation ASR Task",
		Properties: AzureASRProperties{
			PunctuationMode:            "DictatedAndAutomatic",
			ProfanityFilterMode:        "None",
			WordLevelTimestampsEnabled: false, // 不需要词级别时间戳
		},
		DiarizationEnabled: true, // 启用说话人分离
	}

	// 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 发送 HTTP POST 请求（带重试逻辑）
	var response *AzureASRResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[AzureASRAdapter] Retrying create transcription (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = a.sendCreateTranscriptionRequest(endpoint, requestJSON, apiKey)
		if lastErr == nil {
			break // 请求成功，退出重试循环
		}

		// 检查是否为不可重试的错误（401, 429）
		if isNonRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	return response.Self, nil
}

// sendCreateTranscriptionRequest 发送创建转录任务的 HTTP 请求
func (a *AzureASRAdapter) sendCreateTranscriptionRequest(endpoint string, requestJSON []byte, apiKey string) (*AzureASRResponse, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey) // Azure Speech Service 认证方式

	// 发送请求
	resp, err := a.client.Do(req)
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
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("外部 API 服务错误 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	if resp.StatusCode != 201 { // Azure Batch Transcription 创建成功返回 201
		return nil, fmt.Errorf("HTTP 请求失败 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}

	// 解析 JSON 响应
	var asrResponse AzureASRResponse
	if err := json.Unmarshal(responseBody, &asrResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	return &asrResponse, nil
}

// pollTranscriptionStatus 轮询转录任务状态，等待完成
func (a *AzureASRAdapter) pollTranscriptionStatus(transcriptionURL, apiKey string) (string, error) {
	log.Printf("[AzureASRAdapter] Polling transcription status: %s", transcriptionURL)

	// 轮询参数
	maxPollingTime := 300 * time.Second // 最大轮询时间 5 分钟
	pollingInterval := 5 * time.Second  // 轮询间隔 5 秒
	startTime := time.Now()

	for {
		// 检查是否超时
		if time.Since(startTime) > maxPollingTime {
			return "", fmt.Errorf("转录任务轮询超时（超过 %v）", maxPollingTime)
		}

		// 发送 GET 请求获取任务状态
		req, err := http.NewRequest("GET", transcriptionURL, nil)
		if err != nil {
			return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
		}
		req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

		resp, err := a.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("发送 HTTP 请求失败: %w", err)
		}

		responseBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("读取响应体失败: %w", err)
		}

		// 解析响应
		var statusResponse AzureASRResponse
		if err := json.Unmarshal(responseBody, &statusResponse); err != nil {
			return "", fmt.Errorf("解析 JSON 响应失败: %w", err)
		}

		// 检查任务状态
		switch statusResponse.Status {
		case "Succeeded":
			log.Printf("[AzureASRAdapter] Transcription succeeded")
			return statusResponse.Links.Files, nil
		case "Failed":
			return "", fmt.Errorf("转录任务失败")
		case "Running", "NotStarted":
			log.Printf("[AzureASRAdapter] Transcription status: %s, waiting...", statusResponse.Status)
			time.Sleep(pollingInterval)
		default:
			return "", fmt.Errorf("未知的转录任务状态: %s", statusResponse.Status)
		}
	}
}

// getTranscriptionResult 获取转录结果文件
func (a *AzureASRAdapter) getTranscriptionResult(filesURL, apiKey string) (*AzureTranscriptionResult, error) {
	// TODO: 实现获取转录结果文件的完整逻辑（Phase 4 后期实现）
	// 临时占位符：返回空结果
	return &AzureTranscriptionResult{}, fmt.Errorf("获取转录结果功能尚未实现（Phase 4 后期实现）")
}

// parseTranscriptionResult 解析转录结果，转换为 Speaker 列表
func (a *AzureASRAdapter) parseTranscriptionResult(result *AzureTranscriptionResult) ([]*pb.Speaker, error) {
	// TODO: 实现解析转录结果的完整逻辑（Phase 4 后期实现）
	// 临时占位符：返回空列表
	return []*pb.Speaker{}, fmt.Errorf("解析转录结果功能尚未实现（Phase 4 后期实现）")
}

