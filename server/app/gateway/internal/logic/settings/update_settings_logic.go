// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package settings

import (
	"context"
	"fmt"

	"video-in-chinese/server/app/gateway/internal/svc"
	"video-in-chinese/server/app/gateway/internal/types"
	"video-in-chinese/server/app/gateway/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update application settings
func NewUpdateSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSettingsLogic {
	return &UpdateSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSettingsLogic) UpdateSettings(req *types.UpdateSettingsRequest) (resp *types.UpdateSettingsResponse, err error) {
	// Step 1: Parse request body (already done by goctl)
	l.Infof("[UpdateSettings] Received update request, version=%d", req.Version)

	// Step 2: Process masked API Keys (if contains ***, keep original value)
	secret := l.svcCtx.Config.ApiKeyEncryptionSecret
	fieldsToUpdate := make(map[string]string)

	// Helper function to add field if not empty
	addField := func(key, value string) {
		if value != "" {
			fieldsToUpdate[key] = value
		}
	}

	// Helper function to encrypt API Key if not masked
	encryptAPIKey := func(key, value string) error {
		if value == "" {
			return nil
		}
		if utils.IsMaskedAPIKey(value) {
			// Skip masked values (keep original)
			l.Infof("[UpdateSettings] Skipping masked field: %s", key)
			return nil
		}
		encrypted, err := utils.EncryptAPIKey(value, secret)
		if err != nil {
			l.Errorf("[UpdateSettings] Failed to encrypt %s: %v", key, err)
			return fmt.Errorf("encryption failed")
		}
		fieldsToUpdate[key] = encrypted
		return nil
	}

	// Process all fields
	addField("processing_mode", req.ProcessingMode)

	// ASR
	addField("asr_provider", req.AsrProvider)
	if err := encryptAPIKey("asr_api_key", req.AsrApiKey); err != nil {
		return nil, err
	}
	addField("asr_endpoint", req.AsrEndpoint)

	// Audio Separation
	if req.AudioSeparationEnabled != nil {
		fieldsToUpdate["audio_separation_enabled"] = fmt.Sprintf("%v", *req.AudioSeparationEnabled)
	}

	// Polishing
	if req.PolishingEnabled != nil {
		fieldsToUpdate["polishing_enabled"] = fmt.Sprintf("%v", *req.PolishingEnabled)
	}
	addField("polishing_provider", req.PolishingProvider)
	if err := encryptAPIKey("polishing_api_key", req.PolishingApiKey); err != nil {
		return nil, err
	}
	addField("polishing_custom_prompt", req.PolishingCustomPrompt)
	addField("polishing_video_type", req.PolishingVideoType)

	// Translation
	addField("translation_provider", req.TranslationProvider)
	if err := encryptAPIKey("translation_api_key", req.TranslationApiKey); err != nil {
		return nil, err
	}
	addField("translation_endpoint", req.TranslationEndpoint)
	addField("translation_video_type", req.TranslationVideoType)

	// Optimization
	if req.OptimizationEnabled != nil {
		fieldsToUpdate["optimization_enabled"] = fmt.Sprintf("%v", *req.OptimizationEnabled)
	}
	addField("optimization_provider", req.OptimizationProvider)
	if err := encryptAPIKey("optimization_api_key", req.OptimizationApiKey); err != nil {
		return nil, err
	}

	// Voice Cloning
	addField("voice_cloning_provider", req.VoiceCloningProvider)
	if err := encryptAPIKey("voice_cloning_api_key", req.VoiceCloningApiKey); err != nil {
		return nil, err
	}
	addField("voice_cloning_endpoint", req.VoiceCloningEndpoint)
	if req.VoiceCloningAutoSelectReference != nil {
		fieldsToUpdate["voice_cloning_auto_select_reference"] = fmt.Sprintf("%v", *req.VoiceCloningAutoSelectReference)
	}

	// S2ST
	addField("s2st_provider", req.S2stProvider)
	if err := encryptAPIKey("s2st_api_key", req.S2stApiKey); err != nil {
		return nil, err
	}

	// Step 3: Use Lua script to atomically update Redis (optimistic lock)
	// Prepare arguments: [expectedVersion, field1, value1, field2, value2, ...]
	args := []interface{}{req.Version}
	for key, value := range fieldsToUpdate {
		args = append(args, key, value)
	}

	// Execute Lua script
	result, err := l.svcCtx.RedisClient.Eval(utils.UpdateSettingsScript, []string{"app:settings"}, args...)
	if err != nil {
		l.Errorf("[UpdateSettings] Failed to execute Lua script: %v", err)
		return nil, fmt.Errorf("Redis unavailable")
	}

	// Parse result
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 2 {
		l.Errorf("[UpdateSettings] Invalid Lua script result: %v", result)
		return nil, fmt.Errorf("internal error")
	}

	statusCode, _ := resultArray[0].(int64)
	newVersion, _ := resultArray[1].(int64)

	// Step 4: Check optimistic lock result
	if statusCode == -1 {
		l.Infof("[UpdateSettings] Version conflict: expected=%d, current=%d", req.Version, newVersion)
		return nil, fmt.Errorf("configuration has been modified by another user, please refresh and try again (current version: %d)", newVersion)
	}

	// Step 5: Return update result
	resp = &types.UpdateSettingsResponse{
		Version: newVersion,
		Message: "Configuration updated successfully",
	}

	l.Infof("[UpdateSettings] Successfully updated settings, newVersion=%d, fieldsUpdated=%d", newVersion, len(fieldsToUpdate))
	return resp, nil
}
