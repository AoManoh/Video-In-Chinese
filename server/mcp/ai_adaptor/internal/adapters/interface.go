package adapters

import (
	pb "video-in-chinese/ai_adaptor/proto"
)

// ASRAdapter 定义了统一的语音识别适配器抽象，封装各厂商 API 的调用差异并向业务层提供稳定的 gRPC 行为。
//
// 功能说明:
//   - 负责读取本地音频文件、触发供应商识别流程并返回带时间戳的说话人结果。
//   - 为 `logic` 层提供可插拔的语音识别能力，支持在不同厂商间平滑切换。
//
// 设计决策:
//   - 接口仅暴露 `ASR` 单一方法，将重试、限流、OSS 上传等细节隐藏在具体实现中，简化调用方心智负担。
//   - 使用字符串参数避免暴露文件句柄或自定义结构，降低跨语言调用的复杂度。
//
// 使用示例:
//
//	adapter := registry.MustGetASRAdapter("aliyun")
//	speakers, err := adapter.ASR("./tmp/input.wav", decryptedKey, "")
//	if err != nil {
//	    return err
//	}
//	for _, speaker := range speakers {
//	    log.Printf("speaker=%s sentences=%d", speaker.SpeakerId, len(speaker.Sentences))
//	}
//
// 参数说明:
//
//	audioPath string: 待识别的音频文件路径，应确保存在且可读。
//	apiKey string: 经 ConfigManager 解密后的供应商密钥，通常来源于 Redis。
//	endpoint string: 可选自定义 API 地址，为空时使用实现内置的默认值以支持多区域部署。
//
// 返回值说明:
//
//	[]*pb.Speaker: 识别结果列表，按说话人聚合并包含句子的起止时间。
//	error: 当识别失败、网络异常或解析出错时返回，对应供应商返回的状态码。
//
// 错误处理说明:
//   - 401/403 表示认证失败，调用方应提示配置缺失或密钥过期。
//   - 429 代表限流，建议调用方退避重试或切换供应商。
//   - 5xx 视为供应商故障，可配合 VoiceCache 启用降级策略。
//
// 注意事项:
//   - 对于大文件识别需提前预估耗时并设置调用超时。
//   - 如果实现依赖 OSS 上传，调用环境必须提供完整的对象存储配置。
type ASRAdapter interface {
	// ASR 执行一次离线语音识别任务并返回说话人列表。
	//
	// 功能说明:
	//   - 调用供应商 API 完成音频上传、任务提交与结果解析。
	// 设计决策:
	//   - 具体实现内部管理重试与熔断，接口保持极简签名。
	// 使用示例:
	//   speakers, err := adapter.ASR("./clips/a.wav", apiKey, "")
	// 参数说明:
	//   audioPath string: 输入音频路径。
	//   apiKey string: 供应商密钥。
	//   endpoint string: 可选自定义 API 地址。
	// 返回值说明:
	//   []*pb.Speaker: 说话人切分后的识别结果。
	//   error: 识别失败或网络异常时返回。
	// 错误处理说明:
	//   - 统一封装供应商错误信息，方便上层分类处理。
	// 注意事项:
	//   - 需要确保调用环境具备上传依赖，否则应在业务层提前降级。
	ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error)
}

// TranslationAdapter 描述了文本翻译适配器的统一能力，用于屏蔽不同供应商在语种编码、语气控制上的差异。
//
// 功能说明:
//   - 将原始字幕或脚本文本转换为目标语言，支持根据视频类型设置语气。
//   - 供 `logic` 层在翻译流程中选择合适的供应商实现。
//
// 设计决策:
//   - 接口保持同步调用，由具体实现决定是否做缓存或流控。
//
// 使用示例:
//
//	adapter := registry.MustGetTranslationAdapter("deepl")
//	text, err := adapter.Translate(raw, "en", "zh", "educational_rigorous", key, "")
//
// 参数说明:
//
//	text string: 待翻译文本。
//	sourceLang string: 原语言代码，例如 "en"。
//	targetLang string: 目标语言代码，例如 "zh"。
//	videoType string: 视频风格标签，用于驱动 Prompt 或供应商特性。
//	apiKey string: 供应商密钥。
//	endpoint string: 可选 API 地址。
//
// 返回值说明:
//
//	string: 翻译后的文本。
//	error: 请求失败或解析错误时返回。
//
// 错误处理说明:
//   - 400 表示输入语言不受支持，调用方应校验配置。
//   - 429 代表限流，需退避重试或切换供应商。
//   - 5xx 代表供应商故障，可触发降级逻辑。
//
// 注意事项:
//   - 供应商对文本长度有限制，调用方可视情况拆分长段落。
//   - videoType 参数需与配置约定一致，否则可能得到非预期语气。
type TranslationAdapter interface {
	// Translate 执行同步翻译任务并返回目标语言文本。
	//
	// 功能说明:
	//   - 根据视频类型调整翻译语气，保持字幕风格一致。
	// 设计决策:
	//   - 参数明确区分源语言和目标语言，防止调用歧义。
	// 使用示例:
	//   translated, err := adapter.Translate(script, "en", "zh", "default", key, "")
	// 参数说明:
	//   text string: 原始文本。
	//   sourceLang string: 源语言代码。
	//   targetLang string: 目标语言代码。
	//   videoType string: 视频风格标签。
	//   apiKey string: 供应商密钥。
	//   endpoint string: 可选 API 地址。
	// 返回值说明:
	//   string: 翻译结果。
	//   error: 失败时的错误信息。
	// 错误处理说明:
	//   - 将供应商错误透传给调用方以便记录和报警。
	// 注意事项:
	//   - 长文本任务建议调用方拆分并在外层聚合结果。
	Translate(text, sourceLang, targetLang, videoType, apiKey, endpoint string) (string, error)
}

// LLMAdapter 定义了大模型在润色和文本优化场景下的统一接口，方便业务层在不同供应商之间切换而不改动编排逻辑。
//
// 功能说明:
//   - 提供文本润色 (`Polish`) 和脚本优化 (`Optimize`) 两种能力，提升脚本质量与语气一致性。
//
// 设计决策:
//   - 将两个能力拆分到不同方法，允许调用方按需组合，避免注入过多标志位。
//
// 使用示例:
//
//	adapter := registry.MustGetLLMAdapter("openai-gpt4o")
//	polished, err := adapter.Polish(raw, "casual_natural", "", key, "")
//
// 参数说明:
//
//	text string: 输入文本。
//	videoType string: 视频风格标签。
//	customPrompt string: 可选自定义 Prompt，空字符串表示使用默认模版。
//	apiKey string: 模型访问密钥。
//	endpoint string: 可选自定义 API 地址。
//
// 返回值说明:
//
//	string: 模型生成的文本。
//	error: 调用失败或模型报错时返回。
//
// 错误处理说明:
//   - 400 系列错误表示 Prompt 不合法或输入不支持。
//   - 429 代表限流，上层可配置退避策略。
//   - 5xx 代表供应商故障，需要业务层回退或降级。
//
// 注意事项:
//   - 模型调用成本较高，应结合缓存避免重复请求。
//   - 调用前需确认 token 限制，否则可能因长度导致失败。
type LLMAdapter interface {
	// Polish 对文本执行润色，使语气与目标视频类型匹配。
	//
	// 功能说明:
	//   - 根据视频类型调整语气，生成更符合播报场景的文本。
	// 设计决策:
	//   - 支持传入自定义 Prompt，使用者可覆盖默认策略。
	// 使用示例:
	//   output, err := adapter.Polish(text, "professional_tech", "", key, endpoint)
	// 参数说明:
	//   text string: 原始文本。
	//   videoType string: 视频风格标签。
	//   customPrompt string: 可选自定义 Prompt。
	//   apiKey string: 模型密钥。
	//   endpoint string: 可选自定义地址。
	// 返回值说明:
	//   string: 润色后的文本。
	//   error: 模型失败或超时时返回。
	// 错误处理说明:
	//   - 透传供应商错误细节，方便调用方记录与分析。
	// 注意事项:
	//   - 自定义 Prompt 需控制长度以避免超过 token 限制。
	Polish(text, videoType, customPrompt, apiKey, endpoint string) (string, error)

	// Optimize 针对完整脚本执行语义优化，强调结构调整和重点突出。
	//
	// 功能说明:
	//   - 对输入脚本重新组织段落、提醒补充上下文或裁剪冗余内容。
	// 设计决策:
	//   - 保持签名精简，使调用方只需要提供脚本和密钥即可运行。
	// 使用示例:
	//   optimized, err := adapter.Optimize(script, key, "")
	// 参数说明:
	//   text string: 待优化脚本。
	//   apiKey string: 模型密钥。
	//   endpoint string: 可选自定义地址。
	// 返回值说明:
	//   string: 优化后的文本。
	//   error: 模型失败或网络异常时返回。
	// 错误处理说明:
	//   - 将供应商的错误码原样返回，由业务逻辑决定是否降级。
	// 注意事项:
	//   - 建议调用方在外层切片长文本，以免超出模型限制。
	Optimize(text, apiKey, endpoint string) (string, error)
}

// VoiceCloningAdapter 抽象了语音克隆服务能力，使业务层可以重用 VoiceCache 并在供应商间自由切换。
//
// 功能说明:
//   - 根据参考音频创建目标语音，返回生成文件路径或标识。
//   - 与 `voice_cache.VoiceManager` 协同实现语音复用及状态管理。
//
// 设计决策:
//   - 强制使用 `speakerID` 作为缓存键，保证 Clone 与缓存读取之间的一致性。
//
// 使用示例:
//
//	adapter := registry.MustGetVoiceCloningAdapter("aliyun_cosyvoice")
//	path, err := adapter.CloneVoice("speaker-42", script, "./ref.wav", key, "")
//
// 参数说明:
//
//	speakerID string: VoiceCache 使用的主键。
//	text string: 需要合成的文本。
//	referenceAudio string: 参考音频路径。
//	apiKey string: 供应商密钥。
//	endpoint string: 可选 API 地址。
//
// 返回值说明:
//
//	string: 生成音频的路径或 URL。
//	error: 调用失败或供应商报错时返回。
//
// 错误处理说明:
//   - 404 表示供应商侧未找到语音，需要重新注册。
//   - 408 代表语音注册超时，可结合 VoiceManager 的重试策略。
//   - 429/5xx 需要触发业务降级或回退逻辑。
//
// 注意事项:
//   - referenceAudio 可能需要先上传到对象存储，需提前准备配置。
//   - 生成的语音受供应商许可协议约束，注意合规模型使用政策。
type VoiceCloningAdapter interface {
	// CloneVoice 基于参考音频生成新语音，并返回生成文件路径或标识。
	//
	// 功能说明:
	//   - 调用供应商语音克隆 API 并返回可复用的语音资源。
	// 设计决策:
	//   - 使用 speakerID 作为缓存键，与 VoiceManager 协同实现幂等。
	// 使用示例:
	//   path, err := adapter.CloneVoice("speaker-42", text, refAudio, key, endpoint)
	// 参数说明:
	//   speakerID string: VoiceCache 的主键。
	//   text string: 需要合成的文本。
	//   referenceAudio string: 参考音频路径。
	//   apiKey string: 供应商密钥。
	//   endpoint string: 可选自定义地址。
	// 返回值说明:
	//   string: 生成音频的路径或 URL。
	//   error: 克隆失败或网络异常时返回。
	// 错误处理说明:
	//   - 透传供应商错误，便于上层判断是否重试或降级。
	// 注意事项:
	//   - 需保证参考音频存储在供应商可访问的位置。
	CloneVoice(speakerID, text, referenceAudio, apiKey, endpoint string) (string, error)
}
