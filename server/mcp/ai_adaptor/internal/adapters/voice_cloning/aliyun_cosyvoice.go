package voice_cloning

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"video-in-chinese/server/mcp/ai_adaptor/internal/voice_cache"
)

// AliyunCosyVoiceAdapter 封装阿里云 CosyVoice API，实现 VoiceCloningAdapter 接口。
//
// 功能说明:
//   - 管理音色的注册、缓存命中与语音合成，配合 VoiceManager 降低重复调用。
//
// 设计决策:
//   - 采用 VoiceManager 管理音色缓存，网络细节由适配器集中处理。
//
// 使用示例:
//
//	adapter := NewAliyunCosyVoiceAdapter(voiceManager)
//	audioPath, err := adapter.CloneVoice("speaker-1", script, refAudio, apiKey, "")
//
// 参数说明:
//   - 不适用: 结构体实例通过构造函数创建。
//
// 返回值说明:
//   - 不适用: 结构体用于组合 HTTP 客户端与 VoiceManager。
//
// 错误处理说明:
//   - CloneVoice 会根据 HTTP 状态码和业务结果返回具备上下文的错误。
//
// 注意事项:
//   - 需确保 VoiceManager 已正确初始化并具备 Redis 配置。
type AliyunCosyVoiceAdapter struct {
	client       *http.Client
	voiceManager *voice_cache.VoiceManager
}

// NewAliyunCosyVoiceAdapter 创建 CosyVoice 适配器实例并注入 VoiceManager。
//
// 功能说明:
//   - 提供默认超时配置的 HTTP 客户端，并保存 VoiceManager 引用。
//
// 设计决策:
//   - VoiceManager 负责音色缓存，适配器专注于外部 API 调用。
//
// 使用示例:
//
//	adapter := NewAliyunCosyVoiceAdapter(voiceManager)
//
// 返回值说明:
//
//	*AliyunCosyVoiceAdapter: 已初始化的适配器实例。
//
// 注意事项:
//   - VoiceManager 不能为空，否则无法执行音色注册流程。
func NewAliyunCosyVoiceAdapter(voiceManager *voice_cache.VoiceManager) *AliyunCosyVoiceAdapter {
	return &AliyunCosyVoiceAdapter{
		client: &http.Client{
			Timeout: 120 * time.Second, // 声音克隆可能需要较长时间
		},
		voiceManager: voiceManager,
	}
}

// CloneVoice 执行声音克隆并返回合成后的音频文件路径。
//
// 功能说明:
//   - 通过 VoiceManager 获取或注册音色后调用 CosyVoice 合成音频，失败时自动处理缓存失效。
//
// 设计决策:
//   - 将音色注册与缓存逻辑下沉至 VoiceManager，适配器负责 API 调用与失败重试。
//
// 使用示例:
//
//	audioPath, err := adapter.CloneVoice("speaker-1", text, refAudio, apiKey, endpoint)
//
// 参数说明:
//
//	speakerID string: 说话人标识，用于缓存键。
//	text string: 要合成的文本。
//	referenceAudio string: 参考音频路径，用于音色注册。
//	apiKey string: 阿里云 CosyVoice API 密钥。
//	endpoint string: 可选自定义端点，留空使用默认。
//
// 返回值说明:
//
//	string: 生成音频的本地路径。
//	error: 调用失败或注册音色失败时返回。
//
// 错误处理说明:
//   - 将 401/403、429、404、408、5xx 等错误转换为具上下文的提示，并在音色失效时自动重试注册。
//
// 注意事项:
//   - 需要提前准备参考音频文件，并确保 VoiceManager 有效。
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

// synthesizeAudio 使用指定音色合成音频。
//
// 功能说明:
//   - 调用 Python 脚本（使用 DashScope SDK）将文本转换为音频。
//   - 根据官方文档，CosyVoice 语音合成仅支持 WebSocket 或 SDK 调用，不支持 HTTP RESTful。
//   - 当前实现通过调用 Python 子进程使用官方 SDK，后续可迁移到 WebSocket 方案。
//
// 参数说明:
//
//	voiceID string: 音色 ID（由 RegisterVoice 返回）。
//	text string: 待合成文本。
//	apiKey string: 阿里云 CosyVoice API 密钥。
//	endpoint string: 保留参数（当前未使用，为兼容性保留）。
//
// 返回值说明:
//
//	[]byte: 合成的音频数据。
//	error: 合成失败时返回。
//
// 注意事项:
//   - 需要安装 Python 3 和 dashscope SDK: pip install dashscope
//   - Python 脚本路径: server/scripts/synthesize_audio.py
func (a *AliyunCosyVoiceAdapter) synthesizeAudio(voiceID, text, apiKey, endpoint string) ([]byte, error) {
	log.Printf("[AliyunCosyVoiceAdapter] Synthesizing audio: voice_id=%s, text_length=%d", voiceID, len(text))

	// 步骤 1: 创建临时文件保存音频
	tempFile, err := os.CreateTemp("", "cosyvoice_*.wav")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath) // 确保清理临时文件

	// 步骤 1.5: 创建临时文件存储文本（避免Windows命令行参数截断问题）
	textFile, err := os.CreateTemp("", "cosyvoice_text_*.txt")
	if err != nil {
		return nil, fmt.Errorf("创建文本临时文件失败: %w", err)
	}
	textPath := textFile.Name()
	if _, err := textFile.WriteString(text); err != nil {
		textFile.Close()
		os.Remove(textPath)
		return nil, fmt.Errorf("写入文本临时文件失败: %w", err)
	}
	textFile.Close()
	defer os.Remove(textPath) // 确保清理临时文件

	// 步骤 2: 调用 Python 脚本
	scriptPath := "server/scripts/synthesize_audio.py"

	// 检查脚本是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Python 脚本不存在: %s", scriptPath)
	}

	// 设置超时（最多等待 60 秒）
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 构建命令（通过临时文件传递文本，避免命令行参数截断）
	// 根据操作系统选择Python命令
	pythonCmd := "python"
	if runtime.GOOS != "windows" {
		// 仅在非Windows系统尝试使用python3
		if _, err := exec.LookPath("python3"); err == nil {
			pythonCmd = "python3"
		}
	}
	log.Printf("[AliyunCosyVoiceAdapter] Using Python command: %s", pythonCmd)
	cmd := exec.CommandContext(ctx, pythonCmd, scriptPath, voiceID, textPath, tempPath)

	// 设置环境变量
	cmd.Env = append(os.Environ(), fmt.Sprintf("DASHSCOPE_API_KEY=%s", apiKey))

	// 捕获标准输出和错误输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 步骤 3: 执行命令
	log.Printf("[AliyunCosyVoiceAdapter] Executing Python script: %s", scriptPath)
	err = cmd.Run()

	// 打印 Python 脚本的输出（用于调试）
	if stdout.Len() > 0 {
		log.Printf("[Python stdout] %s", stdout.String())
	}
	if stderr.Len() > 0 {
		log.Printf("[Python stderr] %s", stderr.String())
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python 脚本执行超时（60秒）")
		}
		return nil, fmt.Errorf("Python 脚本执行失败: %w, stderr: %s", err, stderr.String())
	}

	// 步骤 4: 读取生成的音频文件
	audioData, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("读取音频文件失败: %w", err)
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("生成的音频文件为空")
	}

	log.Printf("[AliyunCosyVoiceAdapter] Audio synthesized successfully: size=%d bytes", len(audioData))
	return audioData, nil
}

// saveAudioFile 将合成的音频数据写入本地文件并返回路径。
//
// 功能说明:
//   - 根据 speakerID 生成文件名，写入输出目录并返回路径。
//
// 参数说明:
//
//	audioData []byte: 待写入的音频数据。
//	speakerID string: 用于拼接文件名的说话人标识。
//
// 返回值说明:
//
//	string: 保存后的音频文件路径。
//	error: 创建目录或写入文件失败时返回。
//
// 注意事项:
//   - 输出目录可通过环境变量 CLONED_VOICE_OUTPUT_DIR 配置。
func (a *AliyunCosyVoiceAdapter) saveAudioFile(audioData []byte, speakerID string) (string, error) {
	// 步骤 1: 创建输出目录
	// 从环境变量读取输出目录，如果未设置则使用默认值
	outputDir := os.Getenv("CLONED_VOICE_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "data/cloned_voices" // 默认输出目录（相对于项目根）
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
