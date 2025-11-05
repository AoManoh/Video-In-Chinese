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

	"video-in-chinese/server/mcp/ai_adaptor/internal/utils"
	pb "video-in-chinese/server/mcp/ai_adaptor/proto"
)

// AzureASRAdapter 封装 Azure Speech Service 的批量转写能力，实现 ASRAdapter 接口。
//
// 功能说明:
//   - 提交批量转写任务、轮询任务状态并解析最终识别结果。
//
// 设计决策:
//   - 采用带 120 秒超时的 http.Client 以兼容长音频识别。
//
// 使用示例:
//
//	adapter := NewAzureASRAdapter()
//	speakers, err := adapter.ASR(audioPath, apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于维护客户端实例。
//
// 错误处理说明:
//   - 具体错误由 ASR 方法返回。
//
// 注意事项:
//   - 调用前需准备 Azure Speech Service 的订阅密钥与端点。
type AzureASRAdapter struct {
	client *http.Client
}

// NewAzureASRAdapter 创建 Azure ASR 适配器实例并初始化 HTTP 客户端。
//
// 功能说明:
//   - 提供默认超时配置的适配器供业务层直接调用。
//
// 设计决策:
//   - 将 http.Client 封装在结构体中，便于后续注入自定义 Transport。
//
// 使用示例:
//
//	adapter := NewAzureASRAdapter()
//
// 返回值说明:
//
//	*AzureASRAdapter: 已初始化的适配器实例。
//
// 注意事项:
//   - 若需自定义超时，可在返回值上替换 client。
func NewAzureASRAdapter() *AzureASRAdapter {
	return &AzureASRAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 语音识别可能需要较长时间
		},
	}
}

// AzureASRRequest 描述 Azure Speech Service 批量转写接口的请求体。
//
// 功能说明:
//   - 指定音频来源、语种配置及说话人分离选项。
//
// 设计决策:
//   - 使用显式字段映射，便于 JSON 序列化与调试。
//
// 注意事项:
//   - ContentURLs 中的音频需对 Azure 可访问。
type AzureASRRequest struct {
	ContentURLs        []string           `json:"contentUrls"`        // 音频文件 URL 列表
	Locale             string             `json:"locale"`             // 语言区域（如 "en-US", "zh-CN"）
	DisplayName        string             `json:"displayName"`        // 转录任务显示名称
	Properties         AzureASRProperties `json:"properties"`         // 转录属性
	DiarizationEnabled bool               `json:"diarizationEnabled"` // 是否启用说话人分离
}

// AzureASRProperties 控制 Azure 批量转写任务的高级属性（标点、敏感词过滤、时间戳）。
//
// 功能说明:
//   - 配置标点模式、敏感词过滤与单词级时间戳。
//
// 注意事项:
//   - WordLevelTimestampsEnabled 会影响输出体积与费用。
type AzureASRProperties struct {
	PunctuationMode            string `json:"punctuationMode"`            // 标点符号模式（"DictatedAndAutomatic"）
	ProfanityFilterMode        string `json:"profanityFilterMode"`        // 脏话过滤模式（"None"）
	WordLevelTimestampsEnabled bool   `json:"wordLevelTimestampsEnabled"` // 是否启用词级别时间戳
}

// AzureASRResponse 表示批量转写任务的创建与状态响应。
//
// 功能说明:
//   - 提供任务状态、自引用链接与结果文件索引。
//
// 注意事项:
//   - Status 可能取 NotStarted、Running、Succeeded、Failed 等值。
type AzureASRResponse struct {
	Self               string        `json:"self"`               // 转录任务 URL
	Status             string        `json:"status"`             // 任务状态（"NotStarted", "Running", "Succeeded", "Failed"）
	CreatedDateTime    string        `json:"createdDateTime"`    // 创建时间
	LastActionDateTime string        `json:"lastActionDateTime"` // 最后操作时间
	Links              AzureASRLinks `json:"links"`              // 相关链接
}

// AzureASRLinks 提供批量转写任务相关的资源链接（如结果文件列表）。
type AzureASRLinks struct {
	Files string `json:"files"` // 结果文件列表 URL
}

// AzureASRFilesResponse 表示结果文件 API 的响应，包含多个文件条目。
type AzureASRFilesResponse struct {
	Values []AzureASRFile `json:"values"` // 文件列表
}

// AzureASRFile 描述批量转写结果中的单个文件条目（类型、名称与下载地址）。
type AzureASRFile struct {
	Kind       string `json:"kind"`       // 文件类型（"Transcription"）
	Name       string `json:"name"`       // 文件名
	ContentURL string `json:"contentUrl"` // 文件下载 URL
}

// AzureTranscriptionResult 表示批量转写完成后的聚合结果。
//
// 功能说明:
//   - CombinedRecognizedPhrases 提供整体摘要。
//   - RecognizedPhrases 提供逐句明细及说话人信息。
type AzureTranscriptionResult struct {
	CombinedRecognizedPhrases []AzureCombinedPhrase   `json:"combinedRecognizedPhrases"` // 合并的识别短语
	RecognizedPhrases         []AzureRecognizedPhrase `json:"recognizedPhrases"`         // 识别的短语列表
}

// AzureCombinedPhrase 表示聚合后的整段识别内容，包含声道与显示文本。
type AzureCombinedPhrase struct {
	Channel int    `json:"channel"` // 音频通道
	Lexical string `json:"lexical"` // 词汇形式
	Display string `json:"display"` // 显示形式
}

// AzureRecognizedPhrase 表示单条识别结果，包含时间戳、说话人及候选列表。
type AzureRecognizedPhrase struct {
	Channel         int                `json:"channel"`         // 音频通道
	OffsetInTicks   int64              `json:"offsetInTicks"`   // 开始时间（100 纳秒为单位）
	DurationInTicks int64              `json:"durationInTicks"` // 持续时间（100 纳秒为单位）
	NBest           []AzureNBestResult `json:"nBest"`           // N-Best 结果列表
	Speaker         int                `json:"speaker"`         // 说话人 ID（如果启用了说话人分离）
}

// AzureNBestResult 描述单条识别结果的候选列表，包含置信度与不同展示形式。
type AzureNBestResult struct {
	Confidence float64 `json:"confidence"` // 置信度
	Lexical    string  `json:"lexical"`    // 词汇形式
	Display    string  `json:"display"`    // 显示形式
}

// ASR 执行一次离线语音识别，并以说话人列表形式返回结果。
//
// 功能说明:
//   - 上传音频至 Azure Blob（或可访问 URL），创建批量转写任务并解析结果。
//
// 设计决策:
//   - 默认启用说话人分离与标点，便于后续字幕生成。
//   - 内置重试与轮询策略，缓解瞬时网络问题与异步任务延迟。
//
// 使用示例:
//
//	speakers, err := adapter.ASR("./input.wav", apiKey, "")
//
// 参数说明:
//
//	audioPath string: 本地音频文件路径。
//	apiKey string: Azure Speech Service 订阅密钥（需提前解密）。
//	endpoint string: 自定义端点 URL，空字符串使用默认地区端点。
//
// 返回值说明:
//
//	[]*pb.Speaker: 识别结果，按说话人聚合并包含句子时间戳。
//	error: 上传、网络或识别失败时返回。
//
// 错误处理说明:
//   - HTTP 401/403 映射为密钥无效，429 表示限流，5xx 表示供应商故障。
//   - JSON 解析失败会在错误中附带响应体，便于排查。
//
// 注意事项:
//   - 调用前需确保音频已上传至可访问的存储，或提供公开 URL。
//   - 长音频识别耗时较久，建议调用方设置上下文超时。
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

// createTranscription 创建批量转录任务并返回任务查询 URL。
//
// 功能说明:
//   - 构造请求体、发送创建请求并处理重试逻辑。
//
// 设计决策:
//   - 将重试集中于此函数，以便统一处理限流与瞬时错误。
//
// 参数说明:
//
//	endpoint string: Azure 转写 API 端点。
//	audioURL string: 可访问的音频文件 URL。
//	apiKey string: Azure Speech Service 订阅密钥。
//
// 返回值说明:
//
//	string: 创建成功后返回的任务自服务链接。
//	error: 请求失败或重试耗尽时返回。
//
// 注意事项:
//   - 非可重试错误（401、429）会立刻返回，避免无谓重试。
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
		if utils.IsNonRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	return response.Self, nil
}

// sendCreateTranscriptionRequest 向 Azure 转写接口发送创建请求并解析响应。
//
// 功能说明:
//   - 构造 HTTP 请求、校验状态码并将响应解码为 AzureASRResponse。
//
// 参数说明:
//
//	endpoint string: API 端点。
//	requestJSON []byte: 序列化后的请求体。
//	apiKey string: Azure Speech Service 订阅密钥。
//
// 返回值说明:
//
//	*AzureASRResponse: 成功创建任务后的响应。
//	error: HTTP 失败或解码失败时返回。
//
// 注意事项:
//   - 201 状态码表示创建成功，其他状态码会被映射为错误并包含响应体。
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

// pollTranscriptionStatus 轮询批量转写任务状态，直至成功、失败或超时。
//
// 功能说明:
//   - 定期向任务链接发起 GET 请求，解析状态并在成功时返回结果文件链接。
//
// 参数说明:
//
//	transcriptionURL string: 创建任务时返回的自服务链接。
//	apiKey string: Azure Speech Service 订阅密钥。
//
// 返回值说明:
//
//	string: 成功时返回结果文件列表链接。
//	error: 任务失败或轮询超时时返回错误。
//
// 注意事项:
//   - 轮询总时长受 maxPollingTime 限制，默认 5 分钟。
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

// getTranscriptionResult 获取批量转写结果文件并解析为结构化结果。
//
// 功能说明:
//   - 预留实现：未来将下载 JSON 结果并反序列化为 AzureTranscriptionResult。
//
// 注意事项:
//   - Phase 4 后续迭代将补全实现，当前返回占位错误。
func (a *AzureASRAdapter) getTranscriptionResult(filesURL, apiKey string) (*AzureTranscriptionResult, error) {
	// TODO: 实现获取转录结果文件的完整逻辑（Phase 4 后期实现）
	// 临时占位符：返回空结果
	return &AzureTranscriptionResult{}, fmt.Errorf("获取转录结果功能尚未实现（Phase 4 后期实现）")
}

// parseTranscriptionResult 将批量转写结果解析为 pb.Speaker 列表。
//
// 功能说明:
//   - 预留实现：未来会根据 RecognizedPhrases 构建说话人结构。
//
// 注意事项:
//   - Phase 4 后续迭代将补全实现，当前返回占位错误。
func (a *AzureASRAdapter) parseTranscriptionResult(result *AzureTranscriptionResult) ([]*pb.Speaker, error) {
	// TODO: 实现解析转录结果的完整逻辑（Phase 4 后期实现）
	// 临时占位符：返回空列表
	return []*pb.Speaker{}, fmt.Errorf("解析转录结果功能尚未实现（Phase 4 后期实现）")
}
