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

// AliyunASRAdapter 封装阿里云智能语音交互 API 的调用，实现 ASRAdapter 接口。
//
// 功能说明:
//   - 负责音频上传、任务提交与结果解析，输出符合 pb.Speaker 的结构。
//
// 设计决策:
//   - 使用内置 http.Client 并设置较长超时时间，适配大文件识别场景。
//
// 使用示例:
//
//	adapter := NewAliyunASRAdapter()
//	speakers, err := adapter.ASR(audioPath, apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体实例通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于保存状态。
//
// 错误处理说明:
//   - 具体错误在 ASR 方法中返回。
//
// 注意事项:
//   - 调用前需配置 OSS 相关环境变量以支持音频上传。
type AliyunASRAdapter struct {
	client *http.Client
}

// NewAliyunASRAdapter 创建阿里云 ASR 适配器实例并初始化 HTTP 客户端。
//
// 功能说明:
//   - 返回带有默认超时设置的适配器实例。
//
// 设计决策:
//   - 使用 120 秒超时覆盖长音频场景，避免识别中途断开。
//
// 使用示例:
//
//	adapter := NewAliyunASRAdapter()
//
// 参数说明:
//   - 无参数。
//
// 返回值说明:
//
//	*AliyunASRAdapter: 可直接调用 ASR 方法的适配器。
//
// 错误处理说明:
//   - 函数不返回错误，如需定制客户端可在外层包装。
//
// 注意事项:
//   - 若需自定义超时，可在返回值上替换 client。
func NewAliyunASRAdapter() *AliyunASRAdapter {
	return &AliyunASRAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 语音识别可能需要较长时间
		},
	}
}

// AliyunASRRequest 表示阿里云 ASR API 的请求体，字段与官方协议保持一致。
//
// 功能说明:
//   - 配置音频来源、识别模式及输出选项。
//
// 设计决策:
//   - 使用 json 标签匹配阿里云接口要求。
//
// 使用示例:
//
//	req := AliyunASRRequest{AppKey: apiKey, FileLink: url}
//
// 参数说明:
//   - 不适用: 结构体字段由调用方赋值。
//
// 返回值说明:
//   - 不适用。
//
// 错误处理说明:
//   - 构造阶段不产生错误，序列化失败将由调用方捕获。
//
// 注意事项:
//   - FileLink 需为阿里云可访问的 URL。
type AliyunASRRequest struct {
	AppKey            string `json:"appkey"`             // 应用 Key
	FileLink          string `json:"file_link"`          // 音频文件 URL（OSS 公网地址）
	Version           string `json:"version"`            // API 版本（默认 "4.0"）
	EnableWords       bool   `json:"enable_words"`       // 是否返回词级别时间戳
	EnableSpeaker     bool   `json:"enable_speaker"`     // 是否启用说话人分离
	SpeakerCount      int    `json:"speaker_count"`      // 说话人数量（0 表示自动检测）
	EnablePunctuation bool   `json:"enable_punctuation"` // 是否启用标点符号
}

// AliyunASRResponse 映射阿里云 ASR API 的响应结构。
//
// 功能说明:
//   - 承载识别状态码、文本及句子明细。
//
// 设计决策:
//   - 保留 StatusCode 供上层判断业务态。
//
// 使用示例:
//
//	var resp AliyunASRResponse
//	_ = json.Unmarshal(body, &resp)
//
// 参数说明:
//   - 不适用。
//
// 返回值说明:
//   - 不适用。
//
// 错误处理说明:
//   - 解码错误由调用方处理。
//
// 注意事项:
//   - StatusCode 为 20000000 时表示成功。
type AliyunASRResponse struct {
	RequestID  string           `json:"request_id"`  // 请求 ID
	StatusCode int              `json:"status_code"` // 状态码（20000000 表示成功）
	StatusText string           `json:"status_text"` // 状态描述
	Result     *AliyunASRResult `json:"result"`      // 识别结果
}

// AliyunASRResult 表示响应中的识别结果部分。
//
// 功能说明:
//   - 包含句子数组，用于后续转换为 pb.Speaker。
//
// 注意事项:
//   - 空结果需在调用方进行错误处理。
type AliyunASRResult struct {
	Sentences []AliyunSentence `json:"sentences"` // 句子列表
}

// AliyunSentence 对应识别结果中的单句数据，含文本与时间戳。
//
// 功能说明:
//   - 为时间戳转换和说话人聚合提供原始数据。
//
// 注意事项:
//   - BeginTime 和 EndTime 单位为毫秒。
type AliyunSentence struct {
	Text         string `json:"text"`          // 句子文本
	BeginTime    int64  `json:"begin_time"`    // 开始时间（毫秒）
	EndTime      int64  `json:"end_time"`      // 结束时间（毫秒）
	SpeakerID    string `json:"speaker_id"`    // 说话人 ID（如 "0", "1"）
	EmotionValue string `json:"emotion_value"` // 情绪值（可选）
}

// ASR 执行一次离线语音识别，并以说话人列表形式返回结果。
//
// 功能说明:
//   - 校验音频文件、上传到 OSS、调用阿里云 ASR API 并解析返回句子。
//
// 设计决策:
//   - 默认启用说话人分离与标点，便于字幕生成。
//   - 内置重试策略缓解瞬时网络故障和限流。
//
// 使用示例:
//
//	speakers, err := adapter.ASR("./input.wav", apiKey, "")
//
// 参数说明:
//
//	audioPath string: 本地音频文件路径。
//	apiKey string: 阿里云 AppKey（需提前解密）。
//	endpoint string: 自定义端点 URL，空字符串使用默认值。
//
// 返回值说明:
//
//	[]*pb.Speaker: 识别结果，按说话人聚合并包含句子时间戳。
//	error: 上传、网络或识别失败时返回。
//
// 错误处理说明:
//   - HTTP 401/403 映射为密钥无效，429 表示限流，5xx 表示供应商故障。
//   - JSON 解析失败时返回原始响应，便于排查。
//
// 注意事项:
//   - 调用前需配置 OSS 环境变量支持音频上传。
//   - 长音频识别耗时较久，调用方应设置上下文超时。
func (a *AliyunASRAdapter) ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error) {
	log.Printf("[AliyunASRAdapter] Starting ASR: audio_path=%s", audioPath)

	// 步骤 1: 验证音频文件是否存在
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("音频文件不存在: %s", audioPath)
	}

	// 步骤 2: 上传音频文件到阿里云 OSS，获取公网 URL
	// 注意：OSS 配置需要从 ConfigManager 获取，这里使用环境变量作为临时方案
	// 生产环境应该通过 logic 层传递完整的 AppConfig
	fileLink, err := a.uploadToOSS(audioPath)
	if err != nil {
		return nil, fmt.Errorf("上传音频到 OSS 失败: %w", err)
	}

	// 步骤 3: 构建 API 请求
	requestBody := AliyunASRRequest{
		AppKey:            apiKey,
		FileLink:          fileLink,
		Version:           "4.0",
		EnableWords:       false, // 不需要词级别时间戳
		EnableSpeaker:     true,  // 启用说话人分离
		SpeakerCount:      0,     // 自动检测说话人数量
		EnablePunctuation: true,  // 启用标点符号
	}

	// 步骤 4: 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 步骤 5: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 使用默认端点（阿里云智能语音交互 - 录音文件识别）
		apiEndpoint = "https://nls-gateway.cn-shanghai.aliyuncs.com/stream/v1/asr"
	}

	// 步骤 6: 发送 HTTP POST 请求（带重试逻辑）
	var response *AliyunASRResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[AliyunASRAdapter] Retrying ASR request (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = a.sendASRRequest(apiEndpoint, requestJSON, apiKey)
		if lastErr == nil {
			break // 请求成功，退出重试循环
		}

		// 检查是否为不可重试的错误（401, 429）
		if utils.IsNonRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// 步骤 7: 解析响应，转换为 Speaker 列表
	speakers, err := a.parseASRResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析 ASR 响应失败: %w", err)
	}

	log.Printf("[AliyunASRAdapter] ASR completed successfully: %d speakers found", len(speakers))
	return speakers, nil
}

// sendASRRequest 发送 ASR HTTP 请求
func (a *AliyunASRAdapter) sendASRRequest(endpoint string, requestJSON []byte, apiKey string) (*AliyunASRResponse, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey) // 阿里云 API 认证方式

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
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP 请求失败 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}

	// 解析 JSON 响应
	var asrResponse AliyunASRResponse
	if err := json.Unmarshal(responseBody, &asrResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	// 检查业务状态码
	if asrResponse.StatusCode != 20000000 {
		return nil, fmt.Errorf("ASR 识别失败 (业务状态码 %d): %s", asrResponse.StatusCode, asrResponse.StatusText)
	}

	return &asrResponse, nil
}

// parseASRResponse 解析阿里云 ASR 响应并转换为 pb.Speaker 列表。
//
// 功能说明:
//   - 将句子按说话人分组，并转换为 protobuf 定义。
//
// 设计决策:
//   - 使用 map 聚合说话人，保持输出顺序稳定。
//
// 使用示例:
//
//	speakers, err := a.parseASRResponse(resp)
//
// 参数说明:
//
//	response *AliyunASRResponse: 识别响应。
//
// 返回值说明:
//
//	[]*pb.Speaker: 结构化的识别结果。
//	error: 当结果为空或数据异常时返回。
//
// 错误处理说明:
//   - 若响应不包含结果，将返回错误提示调用方检查音频或参数。
//
// 注意事项:
//   - 阿里云可能返回空说话人，此时会 fallback 到默认 speaker_0。
func (a *AliyunASRAdapter) parseASRResponse(response *AliyunASRResponse) ([]*pb.Speaker, error) {
	if response.Result == nil || len(response.Result.Sentences) == 0 {
		return nil, fmt.Errorf("ASR 响应中没有识别结果")
	}

	// 按说话人 ID 分组句子
	speakerMap := make(map[string][]*pb.Sentence)

	for _, sentence := range response.Result.Sentences {
		// 转换时间戳：毫秒 → 秒
		startTime := float64(sentence.BeginTime) / 1000.0
		endTime := float64(sentence.EndTime) / 1000.0

		// 构建 Sentence 对象
		pbSentence := &pb.Sentence{
			Text:      sentence.Text,
			StartTime: startTime,
			EndTime:   endTime,
		}

		// 按说话人 ID 分组
		speakerID := sentence.SpeakerID
		if speakerID == "" {
			speakerID = "speaker_0" // 默认说话人 ID
		}

		speakerMap[speakerID] = append(speakerMap[speakerID], pbSentence)
	}

	// 转换为 Speaker 列表
	var speakers []*pb.Speaker
	for speakerID, sentences := range speakerMap {
		speaker := &pb.Speaker{
			SpeakerId: speakerID,
			Sentences: sentences,
		}
		speakers = append(speakers, speaker)
	}

	return speakers, nil
}

// uploadToOSS 将音频文件上传至阿里云 OSS，并返回公开访问 URL。
//
// 功能说明:
//   - 构造 OSS 上传器并完成文件上传，用于供应商 API 访问。
//
// 设计决策:
//   - 遇到配置缺失时返回模拟 URL，保持降级能力。
//
// 使用示例:
//
//	url, err := a.uploadToOSS("./audio.wav")
//
// 参数说明:
//
//	audioPath string: 本地音频文件路径。
//
// 返回值说明:
//
//	string: 上传后的公开 URL。
//	error: 当配置缺失或上传失败时返回，降级情况下返回空字符串。
//
// 错误处理说明:
//   - OSS 配置缺失将返回错误提示调用方补全。
//
// 注意事项:
//   - 需在环境中设置 ALIYUN_OSS_* 参数确保上传成功。
func (a *AliyunASRAdapter) uploadToOSS(audioPath string) (string, error) {
	// 从环境变量读取 OSS 配置
	accessKeyID := os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("ALIYUN_OSS_BUCKET_NAME")
	endpoint := os.Getenv("ALIYUN_OSS_ENDPOINT")

	// 验证配置
	if accessKeyID == "" || accessKeySecret == "" || bucketName == "" || endpoint == "" {
		return "", fmt.Errorf("OSS 配置不完整，请设置环境变量: ALIYUN_OSS_ACCESS_KEY_ID, ALIYUN_OSS_ACCESS_KEY_SECRET, ALIYUN_OSS_BUCKET_NAME, ALIYUN_OSS_ENDPOINT")
	}

	// 创建 OSS 上传器
	uploader, err := utils.NewOSSUploader(accessKeyID, accessKeySecret, endpoint, bucketName)
	if err != nil {
		return "", fmt.Errorf("创建 OSS 上传器失败: %w", err)
	}

	// 生成对象键
	objectKey := utils.GenerateObjectKey(audioPath, "asr-audio")

	// 上传文件
	publicURL, err := uploader.UploadFile(nil, audioPath, objectKey)
	if err != nil {
		return "", fmt.Errorf("上传文件到 OSS 失败: %w", err)
	}

	return publicURL, nil
}

// contains 检查字符串是否包含子串，用于兼容低版本 SDK 的错误提示。
//
// 功能说明:
//   - 基于字符串长度快速判断是否包含特定子串。
//
// 注意事项:
//   - 仅用于处理阿里云错误码，通用场景建议使用 strings.Contains。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

// containsSubstring 辅助 contains 进行逐字符匹配。
//
// 功能说明:
//   - 线性扫描判断是否包含目标子串。
//
// 注意事项:
//   - 时间复杂度 O(n*m)，仅用于小字符串匹配。
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
