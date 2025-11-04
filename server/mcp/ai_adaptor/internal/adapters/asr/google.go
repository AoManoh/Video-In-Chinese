package asr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	pb "video-in-chinese/ai_adaptor/proto"
)

// GoogleASRAdapter Google 语音识别适配器
// 实现 ASRAdapter 接口，调用 Google Speech-to-Text API
type GoogleASRAdapter struct {
	client *http.Client
}

// NewGoogleASRAdapter 创建新的 Google ASR 适配器
func NewGoogleASRAdapter() *GoogleASRAdapter {
	return &GoogleASRAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 语音识别可能需要较长时间
		},
	}
}

// GoogleASRRequest Google Speech-to-Text API 请求结构
type GoogleASRRequest struct {
	Config GoogleRecognitionConfig `json:"config"` // 识别配置
	Audio  GoogleRecognitionAudio  `json:"audio"`  // 音频数据
}

// GoogleRecognitionConfig Google 识别配置
type GoogleRecognitionConfig struct {
	Encoding                   string                      `json:"encoding"`                   // 音频编码格式（如 "LINEAR16", "FLAC"）
	SampleRateHertz            int                         `json:"sampleRateHertz"`            // 采样率（如 16000）
	LanguageCode               string                      `json:"languageCode"`               // 语言代码（如 "zh-CN", "en-US"）
	EnableAutomaticPunctuation bool                        `json:"enableAutomaticPunctuation"` // 启用自动标点符号
	EnableWordTimeOffsets      bool                        `json:"enableWordTimeOffsets"`      // 启用词级别时间偏移
	DiarizationConfig          *GoogleDiarizationConfig    `json:"diarizationConfig,omitempty"` // 说话人分离配置
	Model                      string                      `json:"model,omitempty"`            // 识别模型（如 "default", "video"）
}

// GoogleDiarizationConfig Google 说话人分离配置
type GoogleDiarizationConfig struct {
	EnableSpeakerDiarization bool `json:"enableSpeakerDiarization"` // 启用说话人分离
	MinSpeakerCount          int  `json:"minSpeakerCount,omitempty"` // 最小说话人数量
	MaxSpeakerCount          int  `json:"maxSpeakerCount,omitempty"` // 最大说话人数量
}

// GoogleRecognitionAudio Google 音频数据
type GoogleRecognitionAudio struct {
	Content string `json:"content,omitempty"` // Base64 编码的音频内容
	URI     string `json:"uri,omitempty"`     // Google Cloud Storage URI（如 "gs://bucket/audio.wav"）
}

// GoogleASRResponse Google Speech-to-Text API 响应结构
type GoogleASRResponse struct {
	Results []GoogleSpeechRecognitionResult `json:"results"` // 识别结果列表
}

// GoogleSpeechRecognitionResult Google 语音识别结果
type GoogleSpeechRecognitionResult struct {
	Alternatives      []GoogleSpeechRecognitionAlternative `json:"alternatives"`      // 识别候选列表
	LanguageCode      string                               `json:"languageCode"`      // 语言代码
	ResultEndTime     string                               `json:"resultEndTime"`     // 结果结束时间（如 "12.345s"）
}

// GoogleSpeechRecognitionAlternative Google 识别候选
type GoogleSpeechRecognitionAlternative struct {
	Transcript string             `json:"transcript"` // 识别文本
	Confidence float64            `json:"confidence"` // 置信度
	Words      []GoogleWordInfo   `json:"words"`      // 词级别信息
}

// GoogleWordInfo Google 词级别信息
type GoogleWordInfo struct {
	StartTime    string `json:"startTime"`    // 开始时间（如 "0.5s"）
	EndTime      string `json:"endTime"`      // 结束时间（如 "1.2s"）
	Word         string `json:"word"`         // 词内容
	SpeakerTag   int    `json:"speakerTag"`   // 说话人标签（如果启用了说话人分离）
}

// ASR 执行语音识别，返回说话人列表
// 参数:
//   - audioPath: 音频文件的本地路径
//   - apiKey: 解密后的 API 密钥（Google Cloud API Key）
//   - endpoint: 自定义端点 URL（为空则使用默认端点）
// 返回:
//   - speakers: 说话人列表，包含句子级时间戳和文本
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 5xx: 外部API服务错误）
func (g *GoogleASRAdapter) ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error) {
	log.Printf("[GoogleASRAdapter] Starting ASR: audio_path=%s", audioPath)

	// 步骤 1: 验证音频文件是否存在
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("音频文件不存在: %s", audioPath)
	}

	// 步骤 2: 读取音频文件并进行 Base64 编码
	// TODO: 对于大文件（>10MB），应上传到 Google Cloud Storage 并使用 URI（Phase 4 后期实现）
	audioContent, err := g.encodeAudioToBase64(audioPath)
	if err != nil {
		return nil, fmt.Errorf("读取音频文件失败: %w", err)
	}

	// 步骤 3: 构建 API 请求
	requestBody := GoogleASRRequest{
		Config: GoogleRecognitionConfig{
			Encoding:                   "LINEAR16", // TODO: 从音频文件自动检测编码格式（Phase 4 后期实现）
			SampleRateHertz:            16000,      // TODO: 从音频文件自动检测采样率（Phase 4 后期实现）
			LanguageCode:               "zh-CN",    // TODO: 从配置中读取语言代码（Phase 4 后期实现）
			EnableAutomaticPunctuation: true,       // 启用自动标点符号
			EnableWordTimeOffsets:      true,       // 启用词级别时间偏移（用于说话人分离）
			DiarizationConfig: &GoogleDiarizationConfig{
				EnableSpeakerDiarization: true, // 启用说话人分离
				MinSpeakerCount:          1,    // 最小说话人数量
				MaxSpeakerCount:          6,    // 最大说话人数量（自动检测）
			},
			Model: "video", // 使用视频模型（更适合视频内容）
		},
		Audio: GoogleRecognitionAudio{
			Content: audioContent, // Base64 编码的音频内容
		},
	}

	// 步骤 4: 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 步骤 5: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 使用默认端点（Google Speech-to-Text API v1 - recognize）
		apiEndpoint = "https://speech.googleapis.com/v1/speech:recognize"
	}

	// 步骤 6: 发送 HTTP POST 请求（带重试逻辑）
	var response *GoogleASRResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[GoogleASRAdapter] Retrying ASR request (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = g.sendASRRequest(apiEndpoint, requestJSON, apiKey)
		if lastErr == nil {
			break // 请求成功，退出重试循环
		}

		// 检查是否为不可重试的错误（401, 429）
		if isNonRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// 步骤 7: 解析响应，转换为 Speaker 列表
	speakers, err := g.parseASRResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析 ASR 响应失败: %w", err)
	}

	log.Printf("[GoogleASRAdapter] ASR completed successfully: %d speakers found", len(speakers))
	return speakers, nil
}

// encodeAudioToBase64 读取音频文件并进行 Base64 编码
func (g *GoogleASRAdapter) encodeAudioToBase64(audioPath string) (string, error) {
	// 读取音频文件
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return "", fmt.Errorf("读取音频文件失败: %w", err)
	}

	// 检查文件大小（Google API 限制：同步识别最大 10MB）
	if len(audioData) > 10*1024*1024 {
		return "", fmt.Errorf("音频文件过大（%d bytes），超过 10MB 限制，请使用异步识别或上传到 Google Cloud Storage", len(audioData))
	}

	// Base64 编码
	encoded := base64.StdEncoding.EncodeToString(audioData)
	return encoded, nil
}

// sendASRRequest 发送 ASR HTTP 请求
func (g *GoogleASRAdapter) sendASRRequest(endpoint string, requestJSON []byte, apiKey string) (*GoogleASRResponse, error) {
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
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("外部 API 服务错误 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP 请求失败 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}

	// 解析 JSON 响应
	var asrResponse GoogleASRResponse
	if err := json.Unmarshal(responseBody, &asrResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	return &asrResponse, nil
}

// parseASRResponse 解析 ASR 响应，转换为 Speaker 列表
func (g *GoogleASRAdapter) parseASRResponse(response *GoogleASRResponse) ([]*pb.Speaker, error) {
	if len(response.Results) == 0 {
		return nil, fmt.Errorf("ASR 响应中没有识别结果")
	}

	// 按说话人标签分组词
	speakerWordsMap := make(map[int][]GoogleWordInfo)

	for _, result := range response.Results {
		if len(result.Alternatives) == 0 {
			continue
		}

		// 使用第一个候选（置信度最高）
		alternative := result.Alternatives[0]

		// 按说话人标签分组词
		for _, word := range alternative.Words {
			speakerTag := word.SpeakerTag
			if speakerTag == 0 {
				speakerTag = 1 // 默认说话人标签为 1
			}
			speakerWordsMap[speakerTag] = append(speakerWordsMap[speakerTag], word)
		}
	}

	// 转换为 Speaker 列表
	var speakers []*pb.Speaker

	for speakerTag, words := range speakerWordsMap {
		// 构建说话人 ID
		speakerID := fmt.Sprintf("speaker_%d", speakerTag)

		// 将词合并为句子（简单策略：按时间间隔分句）
		sentences := g.mergeWordsIntoSentences(words)

		speaker := &pb.Speaker{
			SpeakerId: speakerID,
			Sentences: sentences,
		}
		speakers = append(speakers, speaker)
	}

	return speakers, nil
}

// mergeWordsIntoSentences 将词合并为句子
// 简单策略：如果两个词之间的时间间隔超过 1 秒，则认为是新句子
func (g *GoogleASRAdapter) mergeWordsIntoSentences(words []GoogleWordInfo) []*pb.Sentence {
	if len(words) == 0 {
		return nil
	}

	var sentences []*pb.Sentence
	var currentSentence []string
	var sentenceStartTime float64
	var lastEndTime float64

	for i, word := range words {
		// 解析时间戳（格式：如 "1.5s"）
		startTime := parseGoogleTimestamp(word.StartTime)
		endTime := parseGoogleTimestamp(word.EndTime)

		// 第一个词，初始化句子
		if i == 0 {
			currentSentence = []string{word.Word}
			sentenceStartTime = startTime
			lastEndTime = endTime
			continue
		}

		// 检查时间间隔
		timeGap := startTime - lastEndTime

		// 如果时间间隔超过 1 秒，或者是最后一个词，则结束当前句子
		if timeGap > 1.0 || i == len(words)-1 {
			// 如果是最后一个词且时间间隔不超过 1 秒，则加入当前句子
			if i == len(words)-1 && timeGap <= 1.0 {
				currentSentence = append(currentSentence, word.Word)
				lastEndTime = endTime
			}

			// 构建句子
			sentenceText := ""
			for j, w := range currentSentence {
				if j > 0 {
					sentenceText += " "
				}
				sentenceText += w
			}

			sentence := &pb.Sentence{
				Text:      sentenceText,
				StartTime: sentenceStartTime,
				EndTime:   lastEndTime,
			}
			sentences = append(sentences, sentence)

			// 如果时间间隔超过 1 秒，则开始新句子
			if timeGap > 1.0 {
				currentSentence = []string{word.Word}
				sentenceStartTime = startTime
				lastEndTime = endTime
			}
		} else {
			// 继续当前句子
			currentSentence = append(currentSentence, word.Word)
			lastEndTime = endTime
		}
	}

	return sentences
}

// parseGoogleTimestamp 解析 Google 时间戳格式（如 "1.5s"）为秒（float64）
func parseGoogleTimestamp(timestamp string) float64 {
	if timestamp == "" {
		return 0.0
	}

	// 移除末尾的 "s"
	if len(timestamp) > 0 && timestamp[len(timestamp)-1] == 's' {
		timestamp = timestamp[:len(timestamp)-1]
	}

	// 解析为 float64
	var seconds float64
	fmt.Sscanf(timestamp, "%f", &seconds)
	return seconds
}

