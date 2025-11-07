// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package settings

import (
	"context"
	"fmt"
	"strconv"

	"video-in-chinese/server/app/gateway/internal/svc"
	"video-in-chinese/server/app/gateway/internal/types"
	"video-in-chinese/server/app/gateway/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get current application settings
func NewGetSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSettingsLogic {
	return &GetSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSettingsLogic) GetSettings() (resp *types.GetSettingsResponse, err error) {
	// Step 1: Read configuration from Redis (HGETALL app:settings)
	settings, err := l.svcCtx.RedisClient.Hgetall("app:settings")
	if err != nil {
		l.Errorf("[GetSettings] Failed to read from Redis: %v", err)
		return nil, fmt.Errorf("Redis unavailable")
	}

	// Step 2: If configuration does not exist, return default configuration
	if len(settings) == 0 {
		l.Infof("[GetSettings] No configuration found, returning default")
		return &types.GetSettingsResponse{
			Version:                         0,
			IsConfigured:                    false,
			ProcessingMode:                  "standard",
			AudioSeparationEnabled:          false,
			PolishingEnabled:                false,
			OptimizationEnabled:             false,
			VoiceCloningAutoSelectReference: true,
		}, nil
	}

	// Step 3: Decrypt all API Keys (AES-256-GCM)
	secret := l.svcCtx.Config.ApiKeyEncryptionSecret
	decryptedSettings := make(map[string]string)
	for key, value := range settings {
		// Only decrypt fields ending with "_api_key"
		if len(key) > 8 && key[len(key)-8:] == "_api_key" && value != "" {
			decrypted, err := utils.DecryptAPIKey(value, secret)
			if err != nil {
				l.Errorf("[GetSettings] Failed to decrypt %s: %v", key, err)
				return nil, fmt.Errorf("decryption failed")
			}
			decryptedSettings[key] = decrypted
		} else {
			decryptedSettings[key] = value
		}
	}

	// Step 4: Determine IsConfigured status
	// Check if at least ASR, Translation, VoiceCloning API Keys are configured
	isConfigured := decryptedSettings["asr_api_key"] != "" &&
		decryptedSettings["translation_api_key"] != "" &&
		decryptedSettings["voice_cloning_api_key"] != ""

	// Step 5: Construct and return response
	version, _ := strconv.ParseInt(decryptedSettings["version"], 10, 64)
	audioSeparationEnabled, _ := strconv.ParseBool(decryptedSettings["audio_separation_enabled"])
	polishingEnabled, _ := strconv.ParseBool(decryptedSettings["polishing_enabled"])
	optimizationEnabled, _ := strconv.ParseBool(decryptedSettings["optimization_enabled"])
	voiceCloningAutoSelectReference, _ := strconv.ParseBool(decryptedSettings["voice_cloning_auto_select_reference"])

	resp = &types.GetSettingsResponse{
		Version:                         version,
		IsConfigured:                    isConfigured,
		ProcessingMode:                  decryptedSettings["processing_mode"],
		AsrProvider:                     decryptedSettings["asr_provider"],
		AsrApiKey:                       decryptedSettings["asr_api_key"],
		AsrEndpoint:                     decryptedSettings["asr_endpoint"],
		AudioSeparationEnabled:          audioSeparationEnabled,
		PolishingEnabled:                polishingEnabled,
		PolishingProvider:               decryptedSettings["polishing_provider"],
		PolishingApiKey:                 decryptedSettings["polishing_api_key"],
		PolishingEndpoint:               decryptedSettings["polishing_endpoint"],
		PolishingCustomPrompt:           decryptedSettings["polishing_custom_prompt"],
		PolishingVideoType:              decryptedSettings["polishing_video_type"],
		TranslationProvider:             decryptedSettings["translation_provider"],
		TranslationApiKey:               decryptedSettings["translation_api_key"],
		TranslationEndpoint:             decryptedSettings["translation_endpoint"],
		TranslationVideoType:            decryptedSettings["translation_video_type"],
		OptimizationEnabled:             optimizationEnabled,
		OptimizationProvider:            decryptedSettings["optimization_provider"],
		OptimizationApiKey:              decryptedSettings["optimization_api_key"],
		OptimizationEndpoint:            decryptedSettings["optimization_endpoint"],
		VoiceCloningProvider:            decryptedSettings["voice_cloning_provider"],
		VoiceCloningApiKey:              decryptedSettings["voice_cloning_api_key"],
		VoiceCloningEndpoint:            decryptedSettings["voice_cloning_endpoint"],
		VoiceCloningAutoSelectReference: voiceCloningAutoSelectReference,
		S2stProvider:                    decryptedSettings["s2st_provider"],
		S2stApiKey:                      decryptedSettings["s2st_api_key"],
	}

	l.Infof("[GetSettings] Successfully retrieved settings, version=%d, isConfigured=%v", version, isConfigured)
	return resp, nil
}
