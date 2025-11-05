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

	"video-in-chinese/ai_adaptor/internal/utils"
	pb "video-in-chinese/ai_adaptor/proto"
)

// GoogleASRAdapter 封装 Google Speech-to-Text API 的批量识别能力，实现 ASRAdapter 接口。
//
// 功能说明:
//   - 负责构造识别请求、上传音频并解析返回结果。
//
// 设计决策:
//   - 内置 http.Client 并设置 120 秒超时以兼容长音频识别。
//
// 使用示例:
//
//	adapter := NewGoogleASRAdapter()
//	speakers, err := adapter.ASR(audioPath, apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于维持客户端状态。
//
// 错误处理说明:
//   - 具体错误在 ASR 与内部辅助方法中返回。
//
// 注意事项:
//   - 调用前需准备 Google Cloud Speech API Key 或服务账号凭证。
type GoogleASRAdapter struct {
	client *http.Client
}

// NewGoogleASRAdapter 创建 Google ASR 适配器实例并初始化 HTTP 客户端。
//
// 功能说明:
//   - 提供默认超时配置的适配器实例，供业务层直接调用。
//
// 设计决策:
//   - 将 http.Client 封装在结构体中，便于测试注入。
//
// 使用示例:
//
//	adapter := NewGoogleASRAdapter()
//
// 返回值说明:
//
//	*GoogleASRAdapter: 已初始化的适配器实例。
//
// 注意事项:
//   - 若需自定义超时或代理，可修改返回值的 client 字段。
func NewGoogleASRAdapter() *GoogleASRAdapter {
	return &GoogleASRAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 语音识别可能需要较长时间
		},
	}
}

// GoogleASRRequest 描述 Google Speech-to-Text API 的同步识别请求体。
//
// 功能说明:
//   - 封装识别配置与音频数据。
//
// 注意事项:
//   - Audio 可通过 Base64 内容或 Cloud Storage URI 提供。
type GoogleASRRequest struct {
	Config GoogleRecognitionConfig `json:"config"` // 识别配置
	Audio  GoogleRecognitionAudio  `json:"audio"`  // 音频数据
}

// GoogleRecognitionConfig 定义 Google 识别任务的模型、语种及时间戳等参数。
//
// 功能说明:
//   - 控制编码格式、采样率、语言、标点、说话人分离等选项。
//
// 注意事项:
//   - 启用 DiarizationConfig 时需设置说话人数范围以提升准确率。
type GoogleRecognitionConfig struct {
	Encoding                   string                   `json:"encoding"`                    // 音频编码格式（如 "LINEAR16", "FLAC"）
	SampleRateHertz            int                      `json:"sampleRateHertz"`             // 采样率（如 16000）
	LanguageCode               string                   `json:"languageCode"`                // 语言代码（如 "zh-CN", "en-US"）
	EnableAutomaticPunctuation bool                     `json:"enableAutomaticPunctuation"`  // 启用自动标点符号
	EnableWordTimeOffsets      bool                     `json:"enableWordTimeOffsets"`       // 启用词级别时间偏移
	DiarizationConfig          *GoogleDiarizationConfig `json:"diarizationConfig,omitempty"` // 说话人分离配置
	Model                      string                   `json:"model,omitempty"`             // 识别模型（如 "default", "video"）
}

// GoogleDiarizationConfig 配置说话人分离功能的开关及人数范围。
type GoogleDiarizationConfig struct {
	EnableSpeakerDiarization bool `json:"enableSpeakerDiarization"`  // 启用说话人分离
	MinSpeakerCount          int  `json:"minSpeakerCount,omitempty"` // 最小说话人数量
	MaxSpeakerCount          int  `json:"maxSpeakerCount,omitempty"` // 最大说话人数量
}

// GoogleRecognitionAudio 表示识别音频，可通过 Base64 内容或 GCS URI 提供。
type GoogleRecognitionAudio struct {
	Content string `json:"content,omitempty"` // Base64 编码的音频内容
	URI     string `json:"uri,omitempty"`     // Google Cloud Storage URI（如 "gs://bucket/audio.wav"）
}

// GoogleASRResponse 承载 Google Speech-to-Text API 的识别结果集合。
type GoogleASRResponse struct {
	Results []GoogleSpeechRecognitionResult `json:"results"` // 识别结果列表
}

// GoogleSpeechRecognitionResult 表示一段识别结果，包含候选列表与语言信息。
type GoogleSpeechRecognitionResult struct {
	Alternatives  []GoogleSpeechRecognitionAlternative `json:"alternatives"`  // 识别候选列表
	LanguageCode  string                               `json:"languageCode"`  // 语言代码
	ResultEndTime string                               `json:"resultEndTime"` // 结果结束时间（如 "12.345s"）
}

// GoogleSpeechRecognitionAlternative 描述单个识别候选及其置信度与词级信息。
type GoogleSpeechRecognitionAlternative struct {
	Transcript string           `json:"transcript"` // 识别文本
	Confidence float64          `json:"confidence"` // 置信度
	Words      []GoogleWordInfo `json:"words"`      // 词级别信息
}

// GoogleWordInfo 描述词级别的时间戳与说话人标签，用于构建句子。
type GoogleWordInfo struct {
	StartTime  string `json:"startTime"`  // 开始时间（如 "0.5s"）
	EndTime    string `json:"endTime"`    // 结束时间（如 "1.2s"）
	Word       string `json:"word"`       // 词内容
	SpeakerTag int    `json:"speakerTag"` // 说话人标签（如果启用了说话人分离）
}

// ASR 执行一次离线语音识别，并以说话人列表形式返回结果。
//
// 功能说明:
//   - 读取本地音频、进行 Base64 编码、调用 Google Speech-to-Text API，并解析返回结果。
//
// 设计决策:
//   - 采用同步识别接口，避免额外的异步任务管理。
//   - 提前校验音频文件存在性，减少 API 调用失败。
//
// 使用示例:
//
//	speakers, err := adapter.ASR("./audio.wav", apiKey, "")
//
// 参数说明:
//
//	audioPath string: 本地音频文件路径。
//	apiKey string: Google Cloud API Key，用于访问 Speech-to-Text。
//	endpoint string: 自定义端点 URL，空字符串使用默认 `https://speech.googleapis.com/v1/speech:recognize`。
//
// 返回值说明:
//
//	[]*pb.Speaker: 识别结果，按说话人聚合并包含句子时间戳。
//	error: 当音频缺失、网络异常或识别失败时返回。
//
// 错误处理说明:
//   - HTTP 401/403 映射为认证失败；429 表示配额不足；5xx 表示服务故障。
//   - JSON 解析失败会包含响应体以便排查。
//
// 注意事项:
//   - 大于 10MB 的音频建议改用 GCS URI 上传，待后续实现 TODO。
//   - 调用方应在上下文中配置超时和重试策略。
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
		if utils.IsNonRetryableError(lastErr) {
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

// encodeAudioToBase64 读取音频文件并返回 Base64 编码字符串。
//
// 功能说明:
//   - 将本地音频加载到内存后转换为 Base64，供 Speech-to-Text 同步接口使用。
//
// 设计决策:
//   - 当前一次性读取文件，后续可根据 TODO 优化为流式或 GCS 上传。
//
// 参数说明:
//
//	audioPath string: 本地音频文件路径。
//
// 返回值说明:
//
//	string: Base64 编码后的音频内容。
//	error: 读取文件失败时返回。
//
// 注意事项:
//   - 对于超大文件，应改用云存储路径，避免内存占用。
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

// sendASRRequest 向 Google Speech-to-Text API 发送同步识别请求并解析响应。
//
// 功能说明:
//   - 构造 HTTP POST 请求、校验状态码并解码响应为 GoogleASRResponse。
//
// 参数说明:
//
//	endpoint string: 识别 API 端点。
//	requestJSON []byte: 序列化后的请求体。
//	apiKey string: Google Cloud API Key。
//
// 返回值说明:
//
//	*GoogleASRResponse: 成功时的识别结果。
//	error: 网络请求失败或响应异常时返回。
//
// 注意事项:
//   - 429 与 5xx 状态码会转化为错误，上层可据此实现退避重试。
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

// parseASRResponse 解析 Google 识别响应并转换为 pb.Speaker 列表。
//
// 功能说明:
//   - 选择置信度最高的候选并按说话人标签聚合词级信息。
//
// 参数说明:
//
//	response *GoogleASRResponse: 识别结果。
//
// 返回值说明:
//
//	[]*pb.Speaker: 聚合后的说话人文本列表。
//	error: 当响应为空或数据异常时返回。
//
// 注意事项:
//   - 当缺少说话人标签时会默认归入 speaker_1。
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

// mergeWordsIntoSentences 按时间顺序将词级结果聚合为句子。
//
// 功能说明:
//   - 以 1 秒间隔作为句子切分阈值，并保留起止时间。
//
// 注意事项:
//   - 逻辑为启发式实现，未来可根据标点或更精细规则优化。
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

// parseGoogleTimestamp 将 Google 返回的时间戳（如 "1.5s"）解析为秒数。
//
// 功能说明:
//   - 去除尾部单位并转换为 float64，供句子聚合使用。
//
// 注意事项:
//   - 空字符串返回 0；解析失败时默认为 0。
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
