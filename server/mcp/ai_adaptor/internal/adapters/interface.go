package adapters

import (
	pb "video-in-chinese/ai_adaptor/proto"
)

// ASRAdapter 定义 ASR (语音识别) 适配器接口
type ASRAdapter interface {
	// ASR 执行语音识别，返回说话人列表
	// 参数:
	//   - audioPath: 音频文件的本地路径
	//   - apiKey: 解密后的 API 密钥
	//   - endpoint: 自定义端点 URL（为空则使用默认端点）
	// 返回:
	//   - speakers: 说话人列表，包含句子级时间戳和文本
	//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 5xx: 外部API服务错误）
	ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error)
}

// TranslationAdapter 定义翻译适配器接口
type TranslationAdapter interface {
	// Translate 执行文本翻译
	// 参数:
	//   - text: 待翻译的文本
	//   - sourceLang: 源语言代码（如 "en"）
	//   - targetLang: 目标语言代码（如 "zh"）
	//   - videoType: 视频类型（professional_tech, casual_natural, educational_rigorous, default）
	//   - apiKey: 解密后的 API 密钥
	//   - endpoint: 自定义端点 URL（为空则使用默认端点）
	// 返回:
	//   - translatedText: 翻译后的文本
	//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 400: 不支持的语言对, 5xx: 外部API服务错误）
	Translate(text, sourceLang, targetLang, videoType, apiKey, endpoint string) (string, error)
}

// LLMAdapter 定义 LLM (大语言模型) 适配器接口
type LLMAdapter interface {
	// Polish 执行文本润色
	// 参数:
	//   - text: 待处理的文本
	//   - videoType: 视频类型（professional_tech, casual_natural, educational_rigorous, default）
	//   - customPrompt: 用户自定义 Prompt（可选）
	//   - apiKey: 解密后的 API 密钥
	//   - endpoint: 自定义端点 URL（为空则使用默认端点）
	// 返回:
	//   - polishedText: 润色后的文本
	//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 400: Prompt格式错误, 5xx: 外部API服务错误）
	Polish(text, videoType, customPrompt, apiKey, endpoint string) (string, error)

	// Optimize 执行译文优化
	// 参数:
	//   - text: 待优化的文本
	//   - apiKey: 解密后的 API 密钥
	//   - endpoint: 自定义端点 URL（为空则使用默认端点）
	// 返回:
	//   - optimizedText: 优化后的文本
	//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 5xx: 外部API服务错误）
	Optimize(text, apiKey, endpoint string) (string, error)
}

// VoiceCloningAdapter 定义声音克隆适配器接口
type VoiceCloningAdapter interface {
	// CloneVoice 执行声音克隆
	// 参数:
	//   - speakerID: 说话人 ID（用于缓存）
	//   - text: 要合成的文本
	//   - referenceAudio: 参考音频路径
	//   - apiKey: 解密后的 API 密钥
	//   - endpoint: 自定义端点 URL（为空则使用默认端点）
	// 返回:
	//   - audioPath: 合成的音频路径
	//   - error: 错误信息（401: API密钥无效, 429: API配额不足, 404: 音色不存在, 408: 音色注册超时, 5xx: 外部API服务错误）
	// 注意: 音色管理逻辑（缓存检查、音色注册、轮询）在适配器内部实现
	CloneVoice(speakerID, text, referenceAudio, apiKey, endpoint string) (string, error)
}

