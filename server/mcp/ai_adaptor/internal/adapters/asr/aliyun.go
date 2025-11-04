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

	"video-in-chinese/ai_adaptor/internal/utils"
	pb "video-in-chinese/ai_adaptor/proto"
)

// AliyunASRAdapter 阿里云语音识别适配器
// 实现 ASRAdapter 接口，调用阿里云智能语音交互 API
type AliyunASRAdapter struct {
	client *http.Client
}

// NewAliyunASRAdapter 创建新的阿里云 ASR 适配器
func NewAliyunASRAdapter() *AliyunASRAdapter {
	return &AliyunASRAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 语音识别可能需要较长时间
		},
	}
}

// AliyunASRRequest 阿里云 ASR API 请求结构
type AliyunASRRequest struct {
	AppKey            string `json:"appkey"`             // 应用 Key
	FileLink          string `json:"file_link"`          // 音频文件 URL（OSS 公网地址）
	Version           string `json:"version"`            // API 版本（默认 "4.0"）
	EnableWords       bool   `json:"enable_words"`       // 是否返回词级别时间戳
	EnableSpeaker     bool   `json:"enable_speaker"`     // 是否启用说话人分离
	SpeakerCount      int    `json:"speaker_count"`      // 说话人数量（0 表示自动检测）
	EnablePunctuation bool   `json:"enable_punctuation"` // 是否启用标点符号
}

// AliyunASRResponse 阿里云 ASR API 响应结构
type AliyunASRResponse struct {
	RequestID  string           `json:"request_id"`  // 请求 ID
	StatusCode int              `json:"status_code"` // 状态码（20000000 表示成功）
	StatusText string           `json:"status_text"` // 状态描述
	Result     *AliyunASRResult `json:"result"`      // 识别结果
}

// AliyunASRResult 阿里云 ASR 识别结果
type AliyunASRResult struct {
	Sentences []AliyunSentence `json:"sentences"` // 句子列表
}

// AliyunSentence 阿里云句子结构
type AliyunSentence struct {
	Text         string `json:"text"`          // 句子文本
	BeginTime    int64  `json:"begin_time"`    // 开始时间（毫秒）
	EndTime      int64  `json:"end_time"`      // 结束时间（毫秒）
	SpeakerID    string `json:"speaker_id"`    // 说话人 ID（如 "0", "1"）
	EmotionValue string `json:"emotion_value"` // 情绪值（可选）
}

// ASR 执行语音识别，返回说话人列表
// 参数:
//   - audioPath: 音频文件的本地路径
//   - apiKey: 解密后的 API 密钥（阿里云 AppKey）
//   - endpoint: 自定义端点 URL（为空则使用默认端点）
//
// 返回:
//   - speakers: 说话人列表，包含句子级时间戳和文本
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 5xx: 外部API服务错误）
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
		log.Printf("[AliyunASRAdapter] WARNING: OSS upload failed, using local path: %v", err)
		fileLink = audioPath // 降级方案：使用本地文件路径
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

// parseASRResponse 解析 ASR 响应，转换为 Speaker 列表
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

// uploadToOSS 上传音频文件到阿里云 OSS
//
// 参数:
//   - audioPath: 本地音频文件路径
//
// 返回:
//   - publicURL: OSS 公开访问 URL
//   - error: 错误信息
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
