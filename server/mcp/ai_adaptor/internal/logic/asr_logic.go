package logic

import (
	"context"
	"fmt"
	"log"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// ASRLogic coordinates automatic speech recognition flows by loading runtime
// settings from ConfigManager, resolving adapters via AdapterRegistry, and
// delegating execution to the provider implementation. It stays stateless so
// caching and retry policies remain inside lower layers.
type ASRLogic struct {
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
}

// NewASRLogic builds an ASRLogic bound to the shared adapter registry and
// configuration manager. Callers should reuse the returned instance so the
// Redis-backed ConfigManager cache remains hot across requests.
func NewASRLogic(registry *adapters.AdapterRegistry, configManager *config.ConfigManager) *ASRLogic {
	return &ASRLogic{
		registry:      registry,
		configManager: configManager,
	}
}

// ProcessASR handles an automatic speech recognition request end to end.
//
// Workflow:
//  1. Validate the protobuf payload to ensure an audio path is provided.
//  2. Load ASR provider credentials from the Redis-backed ConfigManager cache.
//  3. Resolve the concrete adapter from AdapterRegistry based on configuration.
//  4. Invoke the adapter's ASR method and translate results into protobuf
//     speakers.
//
// Parameters:
//   - ctx: Propagates cancellation and deadline signals to the adapter call.
//   - req: Incoming gRPC request containing the audio asset that must be
//     transcribed.
//
// Returns:
//   - *pb.ASRResponse with the detected speakers when transcription succeeds.
//   - error describing validation errors, configuration lookup failures,
//     missing adapters, or provider execution issues.
//
// Design considerations:
//   - The logic layer is intentionally stateless; caching and throttling live in
//     ConfigManager and adapter implementations to keep business rules testable.
//   - Errors are wrapped with fmt.Errorf so transport layers retain root causes
//     while exposing localized messages to clients.
//
// Example:
//
//	res, err := l.ProcessASR(ctx, &pb.ASRRequest{AudioPath: "/tmp/demo.wav"})
//	if err != nil {
//		return err
//	}
//	log.Printf("detected speakers: %d", len(res.Speakers))
func (l *ASRLogic) ProcessASR(ctx context.Context, req *pb.ASRRequest) (*pb.ASRResponse, error) {
	log.Printf("[ASRLogic] Processing ASR request: audio_path=%s", req.AudioPath)

	// 步骤 1: 验证请求参数
	if req.AudioPath == "" {
		return nil, fmt.Errorf("音频文件路径不能为空")
	}

	// 步骤 2: 从 Redis 读取配置
	appConfig, err := l.configManager.GetConfig(ctx)
	if err != nil {
		log.Printf("[ASRLogic] ERROR: Failed to get config: %v", err)
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}

	// 步骤 3: 验证 ASR 配置
	if appConfig.ASRProvider == "" {
		return nil, fmt.Errorf("ASR 服务商未配置")
	}
	if appConfig.ASRAPIKey == "" {
		return nil, fmt.Errorf("ASR API 密钥未配置")
	}

	log.Printf("[ASRLogic] Using ASR provider: %s", appConfig.ASRProvider)

	// 步骤 4: 从适配器注册表获取 ASR 适配器
	adapter, err := l.registry.GetASR(appConfig.ASRProvider)
	if err != nil {
		log.Printf("[ASRLogic] ERROR: Failed to get ASR adapter: %v", err)
		return nil, fmt.Errorf("获取 ASR 适配器失败: %w", err)
	}

	// 步骤 5: 调用适配器执行语音识别
	speakers, err := adapter.ASR(req.AudioPath, appConfig.ASRAPIKey, appConfig.ASREndpoint)
	if err != nil {
		log.Printf("[ASRLogic] ERROR: ASR failed: %v", err)
		return nil, fmt.Errorf("语音识别失败: %w", err)
	}

	// 步骤 6: 返回结果
	log.Printf("[ASRLogic] ASR completed successfully: %d speakers found", len(speakers))
	return &pb.ASRResponse{
		Speakers: speakers,
	}, nil
}
