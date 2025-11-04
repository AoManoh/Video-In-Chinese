package voice_cloning

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"video-in-chinese/ai_adaptor/internal/utils"
	"video-in-chinese/ai_adaptor/internal/voice_cache"
)

// AliyunCosyVoiceAdapter 阿里云 CosyVoice 声音克隆适配器
// 实现 VoiceCloningAdapter 接口，调用阿里云 CosyVoice API
// 集成 VoiceManager 实现音色注册、缓存、轮询
type AliyunCosyVoiceAdapter struct {
	client       *http.Client
	voiceManager *voice_cache.VoiceManager
}

// NewAliyunCosyVoiceAdapter 创建新的阿里云 CosyVoice 适配器
func NewAliyunCosyVoiceAdapter(voiceManager *voice_cache.VoiceManager) *AliyunCosyVoiceAdapter {
	return &AliyunCosyVoiceAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 声音克隆可能需要较长时间
		},
		voiceManager: voiceManager,
	}
}

// AliyunSynthesizeRequest 阿里云音频合成请求结构
type AliyunSynthesizeRequest struct {
	VoiceID string `json:"voice_id"` // 音色 ID
	Text    string `json:"text"`     // 要合成的文本
	Format  string `json:"format"`   // 音频格式（如 "wav", "mp3"）
}

// AliyunSynthesizeResponse 阿里云音频合成响应结构
type AliyunSynthesizeResponse struct {
	StatusCode int    `json:"status_code"` // 业务状态码（20000000 表示成功）
	Message    string `json:"message"`     // 响应消息
	AudioData  string `json:"audio_data"`  // Base64 编码的音频数据
}

// CloneVoice 执行声音克隆
// 参数:
//   - speakerID: 说话人 ID（用于缓存）
//   - text: 要合成的文本
//   - referenceAudio: 参考音频路径
//   - apiKey: 解密后的 API 密钥
//   - endpoint: 自定义端点 URL（为空则使用默认端点）
//
// 返回:
//   - audioPath: 合成的音频路径
//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 404: 音色不存在, 408: 音色注册超时, 5xx: 外部API服务错误）
//
// 注意: 音色管理逻辑（缓存检查、音色注册、轮询）由 VoiceManager 实现
func (a *AliyunCosyVoiceAdapter) CloneVoice(speakerID, text, referenceAudio, apiKey, endpoint string) (string, error) {
	log.Printf("[AliyunCosyVoiceAdapter] Starting voice cloning: speaker_id=%s", speakerID)

	// 步骤 1: 验证输入参数
	if speakerID == "" {
		return "", fmt.Errorf("说话人 ID 不能为空")
	}
	if text == "" {
		return "", fmt.Errorf("要合成的文本不能为空")
	}
	if referenceAudio == "" {
		return "", fmt.Errorf("参考音频路径不能为空")
	}

	// 步骤 2: 获取或注册音色（使用 VoiceManager）
	ctx := context.Background()
	voiceID, err := a.voiceManager.GetOrRegisterVoice(ctx, speakerID, referenceAudio, apiKey, endpoint)
	if err != nil {
		return "", fmt.Errorf("获取或注册音色失败: %w", err)
	}

	log.Printf("[AliyunCosyVoiceAdapter] Voice ID obtained: voice_id=%s", voiceID)

	// 步骤 3: 调用阿里云 API 合成音频
	audioData, err := a.synthesizeAudio(voiceID, text, apiKey, endpoint)
	if err != nil {
		// 检查是否为音色不存在错误（404）
		if contains(err.Error(), "音色不存在") || contains(err.Error(), "HTTP 404") {
			log.Printf("[AliyunCosyVoiceAdapter] Voice not found, invalidating cache and retrying: voice_id=%s", voiceID)

			// 音色失效，清除缓存并重新注册
			newVoiceID, retryErr := a.voiceManager.HandleVoiceNotFound(ctx, speakerID, referenceAudio, apiKey, endpoint)
			if retryErr != nil {
				return "", fmt.Errorf("音色失效处理失败: %w", retryErr)
			}

			// 使用重新注册后的音色 ID
			voiceID = newVoiceID

			// 重新合成音频
			audioData, err = a.synthesizeAudio(voiceID, text, apiKey, endpoint)
			if err != nil {
				return "", fmt.Errorf("重新合成音频失败: %w", err)
			}
		} else {
			return "", fmt.Errorf("合成音频失败: %w", err)
		}
	}

	// 步骤 4: 保存音频文件
	audioPath, err := a.saveAudioFile(audioData, speakerID)
	if err != nil {
		return "", fmt.Errorf("保存音频文件失败: %w", err)
	}

	log.Printf("[AliyunCosyVoiceAdapter] Voice cloning completed successfully: audio_path=%s", audioPath)
	return audioPath, nil
}

// synthesizeAudio 调用阿里云 API 合成音频
func (a *AliyunCosyVoiceAdapter) synthesizeAudio(voiceID, text, apiKey, endpoint string) ([]byte, error) {
	// 步骤 1: 构建请求体
	requestBody := AliyunSynthesizeRequest{
		VoiceID: voiceID,
		Text:    text,
		Format:  "wav", // 使用 WAV 格式
	}

	// 步骤 2: 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 步骤 3: 确定 API 端点
	apiEndpoint := endpoint
	if apiEndpoint == "" {
		// 从环境变量读取默认端点，如果未设置则使用阿里云官方端点
		apiEndpoint = os.Getenv("ALIYUN_COSYVOICE_ENDPOINT")
		if apiEndpoint == "" {
			apiEndpoint = "https://nls-gateway.cn-shanghai.aliyuncs.com/cosyvoice/v1/synthesize"
		}
	}

	// 步骤 4: 发送 HTTP POST 请求（带重试逻辑）
	var response *AliyunSynthesizeResponse
	var lastErr error

	for retryCount := 0; retryCount <= 3; retryCount++ {
		if retryCount > 0 {
			log.Printf("[AliyunCosyVoiceAdapter] Retrying synthesize request (attempt %d/3)", retryCount)
			time.Sleep(2 * time.Second) // 重试间隔 2 秒
		}

		response, lastErr = a.sendSynthesizeRequest(apiEndpoint, requestJSON, apiKey)
		if lastErr == nil {
			break // 请求成功，退出重试循环
		}

		// 检查是否为不可重试的错误（401, 429, 404）
		if utils.IsNonRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// 步骤 5: 解码 Base64 音频数据
	audioData, err := base64.StdEncoding.DecodeString(response.AudioData)
	if err != nil {
		return nil, fmt.Errorf("解码音频数据失败: %w", err)
	}

	log.Printf("[AliyunCosyVoiceAdapter] Audio synthesized successfully: size=%d bytes", len(audioData))
	return audioData, nil
}

// sendSynthesizeRequest 发送音频合成 HTTP 请求
func (a *AliyunCosyVoiceAdapter) sendSynthesizeRequest(endpoint string, requestJSON []byte, apiKey string) (*AliyunSynthesizeResponse, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey) // 阿里云认证方式

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
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("音色不存在 (HTTP 404): %s", string(responseBody))
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("外部 API 服务错误 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP 请求失败 (HTTP %d): %s", resp.StatusCode, string(responseBody))
	}

	// 解析 JSON 响应
	var synthesizeResponse AliyunSynthesizeResponse
	if err := json.Unmarshal(responseBody, &synthesizeResponse); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 响应体: %s", err, string(responseBody))
	}

	// 检查业务状态码
	if synthesizeResponse.StatusCode != 20000000 {
		return nil, fmt.Errorf("音频合成失败 (业务状态码 %d): %s", synthesizeResponse.StatusCode, synthesizeResponse.Message)
	}

	return &synthesizeResponse, nil
}

// saveAudioFile 保存音频文件到本地
func (a *AliyunCosyVoiceAdapter) saveAudioFile(audioData []byte, speakerID string) (string, error) {
	// 步骤 1: 创建输出目录
	// 从环境变量读取输出目录，如果未设置则使用默认值
	outputDir := os.Getenv("CLONED_VOICE_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "./output/cloned_voices" // 默认输出目录
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 步骤 2: 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.wav", speakerID, timestamp)
	audioPath := filepath.Join(outputDir, filename)

	// 步骤 3: 写入文件
	if err := os.WriteFile(audioPath, audioData, 0644); err != nil {
		return "", fmt.Errorf("写入音频文件失败: %w", err)
	}

	log.Printf("[AliyunCosyVoiceAdapter] Audio file saved: %s", audioPath)
	return audioPath, nil
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
