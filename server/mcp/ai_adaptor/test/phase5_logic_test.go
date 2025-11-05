package test

import (
	"context"
	"testing"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/adapters/asr"
	"video-in-chinese/ai_adaptor/internal/adapters/llm"
	"video-in-chinese/ai_adaptor/internal/adapters/translation"
	"video-in-chinese/ai_adaptor/internal/adapters/voice_cloning"
	"video-in-chinese/ai_adaptor/internal/logic"
	"video-in-chinese/ai_adaptor/internal/voice_cache"
	pb "video-in-chinese/ai_adaptor/proto"
)

// TestASRLogicStructure 测试 ASR 服务逻辑结构
func TestASRLogicStructure(t *testing.T) {
	// 创建模拟的依赖
	registry := adapters.NewAdapterRegistry()
	registry.RegisterASR("aliyun", asr.NewAliyunASRAdapter())

	// 注意：这里不创建真实的 ConfigManager，因为需要 Redis
	// 仅测试逻辑结构是否正确

	// 创建 ASR 服务逻辑实例
	asrLogic := logic.NewASRLogic(registry, nil)

	if asrLogic == nil {
		t.Fatal("Failed to create ASRLogic instance")
	}

	t.Log("✓ ASRLogic structure test passed")
}

// TestTranslateLogicStructure 测试翻译服务逻辑结构
func TestTranslateLogicStructure(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterTranslation("google", translation.NewGoogleTranslationAdapter())

	translateLogic := logic.NewTranslateLogic(registry, nil)

	if translateLogic == nil {
		t.Fatal("Failed to create TranslateLogic instance")
	}

	t.Log("✓ TranslateLogic structure test passed")
}

// TestPolishLogicStructure 测试文本润色服务逻辑结构
func TestPolishLogicStructure(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterLLM("gemini", llm.NewGeminiLLMAdapter())

	polishLogic := logic.NewPolishLogic(registry, nil)

	if polishLogic == nil {
		t.Fatal("Failed to create PolishLogic instance")
	}

	t.Log("✓ PolishLogic structure test passed")
}

// TestOptimizeLogicStructure 测试译文优化服务逻辑结构
func TestOptimizeLogicStructure(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterLLM("openai", llm.NewOpenAILLMAdapter())

	optimizeLogic := logic.NewOptimizeLogic(registry, nil)

	if optimizeLogic == nil {
		t.Fatal("Failed to create OptimizeLogic instance")
	}

	t.Log("✓ OptimizeLogic structure test passed")
}

// TestCloneVoiceLogicStructure 测试声音克隆服务逻辑结构
func TestCloneVoiceLogicStructure(t *testing.T) {
	registry := adapters.NewAdapterRegistry()

	// 创建模拟的 VoiceManager（不需要真实的 Redis）
	voiceManager := voice_cache.NewVoiceManager(nil)

	registry.RegisterVoiceCloning("aliyun_cosyvoice", voice_cloning.NewAliyunCosyVoiceAdapter(voiceManager))

	cloneVoiceLogic := logic.NewCloneVoiceLogic(registry, nil)

	if cloneVoiceLogic == nil {
		t.Fatal("Failed to create CloneVoiceLogic instance")
	}

	t.Log("✓ CloneVoiceLogic structure test passed")
}

// TestASRLogicParameterValidation 测试 ASR 服务逻辑参数验证
func TestASRLogicParameterValidation(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterASR("aliyun", asr.NewAliyunASRAdapter())

	asrLogic := logic.NewASRLogic(registry, nil)

	// 测试空音频路径
	req := &pb.ASRRequest{
		AudioPath: "",
	}

	_, err := asrLogic.ProcessASR(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty audio path, got nil")
	}

	if err.Error() != "音频文件路径不能为空" {
		t.Fatalf("Expected '音频文件路径不能为空', got '%s'", err.Error())
	}

	t.Log("✓ ASRLogic parameter validation test passed")
}

// TestTranslateLogicParameterValidation 测试翻译服务逻辑参数验证
func TestTranslateLogicParameterValidation(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterTranslation("google", translation.NewGoogleTranslationAdapter())

	translateLogic := logic.NewTranslateLogic(registry, nil)

	// 测试空文本
	req := &pb.TranslateRequest{
		Text:       "",
		SourceLang: "en",
		TargetLang: "zh",
	}

	_, err := translateLogic.ProcessTranslate(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty text, got nil")
	}

	if err.Error() != "待翻译文本不能为空" {
		t.Fatalf("Expected '待翻译文本不能为空', got '%s'", err.Error())
	}

	// 测试空源语言
	req = &pb.TranslateRequest{
		Text:       "Hello",
		SourceLang: "",
		TargetLang: "zh",
	}

	_, err = translateLogic.ProcessTranslate(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty source language, got nil")
	}

	if err.Error() != "源语言代码不能为空" {
		t.Fatalf("Expected '源语言代码不能为空', got '%s'", err.Error())
	}

	// 测试空目标语言
	req = &pb.TranslateRequest{
		Text:       "Hello",
		SourceLang: "en",
		TargetLang: "",
	}

	_, err = translateLogic.ProcessTranslate(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty target language, got nil")
	}

	if err.Error() != "目标语言代码不能为空" {
		t.Fatalf("Expected '目标语言代码不能为空', got '%s'", err.Error())
	}

	t.Log("✓ TranslateLogic parameter validation test passed")
}

// TestPolishLogicParameterValidation 测试文本润色服务逻辑参数验证
func TestPolishLogicParameterValidation(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterLLM("gemini", llm.NewGeminiLLMAdapter())

	polishLogic := logic.NewPolishLogic(registry, nil)

	// 测试空文本
	req := &pb.PolishRequest{
		Text: "",
	}

	_, err := polishLogic.ProcessPolish(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty text, got nil")
	}

	if err.Error() != "待润色文本不能为空" {
		t.Fatalf("Expected '待润色文本不能为空', got '%s'", err.Error())
	}

	t.Log("✓ PolishLogic parameter validation test passed")
}

// TestOptimizeLogicParameterValidation 测试译文优化服务逻辑参数验证
func TestOptimizeLogicParameterValidation(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	registry.RegisterLLM("openai", llm.NewOpenAILLMAdapter())

	optimizeLogic := logic.NewOptimizeLogic(registry, nil)

	// 测试空文本
	req := &pb.OptimizeRequest{
		Text: "",
	}

	_, err := optimizeLogic.ProcessOptimize(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty text, got nil")
	}

	if err.Error() != "待优化文本不能为空" {
		t.Fatalf("Expected '待优化文本不能为空', got '%s'", err.Error())
	}

	t.Log("✓ OptimizeLogic parameter validation test passed")
}

// TestCloneVoiceLogicParameterValidation 测试声音克隆服务逻辑参数验证
func TestCloneVoiceLogicParameterValidation(t *testing.T) {
	registry := adapters.NewAdapterRegistry()
	voiceManager := voice_cache.NewVoiceManager(nil)
	registry.RegisterVoiceCloning("aliyun_cosyvoice", voice_cloning.NewAliyunCosyVoiceAdapter(voiceManager))

	cloneVoiceLogic := logic.NewCloneVoiceLogic(registry, nil)

	// 测试空说话人 ID
	req := &pb.CloneVoiceRequest{
		SpeakerId:      "",
		Text:           "Hello",
		ReferenceAudio: "/path/to/audio.wav",
	}

	_, err := cloneVoiceLogic.ProcessCloneVoice(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty speaker ID, got nil")
	}

	if err.Error() != "说话人 ID 不能为空" {
		t.Fatalf("Expected '说话人 ID 不能为空', got '%s'", err.Error())
	}

	// 测试空文本
	req = &pb.CloneVoiceRequest{
		SpeakerId:      "speaker1",
		Text:           "",
		ReferenceAudio: "/path/to/audio.wav",
	}

	_, err = cloneVoiceLogic.ProcessCloneVoice(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty text, got nil")
	}

	if err.Error() != "待合成文本不能为空" {
		t.Fatalf("Expected '待合成文本不能为空', got '%s'", err.Error())
	}

	// 测试空参考音频
	req = &pb.CloneVoiceRequest{
		SpeakerId:      "speaker1",
		Text:           "Hello",
		ReferenceAudio: "",
	}

	_, err = cloneVoiceLogic.ProcessCloneVoice(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for empty reference audio, got nil")
	}

	if err.Error() != "参考音频路径不能为空" {
		t.Fatalf("Expected '参考音频路径不能为空', got '%s'", err.Error())
	}

	t.Log("✓ CloneVoiceLogic parameter validation test passed")
}

// TestAdapterRegistryIntegration 测试适配器注册表集成
func TestAdapterRegistryIntegration(t *testing.T) {
	registry := adapters.NewAdapterRegistry()

	// 注册所有适配器
	registry.RegisterASR("aliyun", asr.NewAliyunASRAdapter())
	registry.RegisterASR("azure", asr.NewAzureASRAdapter())
	registry.RegisterASR("google", asr.NewGoogleASRAdapter())
	registry.RegisterTranslation("google", translation.NewGoogleTranslationAdapter())
	registry.RegisterLLM("gemini", llm.NewGeminiLLMAdapter())
	registry.RegisterLLM("openai", llm.NewOpenAILLMAdapter())

	voiceManager := voice_cache.NewVoiceManager(nil)
	registry.RegisterVoiceCloning("aliyun_cosyvoice", voice_cloning.NewAliyunCosyVoiceAdapter(voiceManager))

	// 测试获取 ASR 适配器
	_, err := registry.GetASR("aliyun")
	if err != nil {
		t.Fatalf("Failed to get aliyun ASR adapter: %v", err)
	}

	_, err = registry.GetASR("azure")
	if err != nil {
		t.Fatalf("Failed to get azure ASR adapter: %v", err)
	}

	_, err = registry.GetASR("google")
	if err != nil {
		t.Fatalf("Failed to get google ASR adapter: %v", err)
	}

	// 测试获取翻译适配器
	_, err = registry.GetTranslation("google")
	if err != nil {
		t.Fatalf("Failed to get google translation adapter: %v", err)
	}

	// 测试获取 LLM 适配器
	_, err = registry.GetLLM("gemini")
	if err != nil {
		t.Fatalf("Failed to get gemini LLM adapter: %v", err)
	}

	_, err = registry.GetLLM("openai")
	if err != nil {
		t.Fatalf("Failed to get openai LLM adapter: %v", err)
	}

	// 测试获取声音克隆适配器
	_, err = registry.GetVoiceCloning("aliyun_cosyvoice")
	if err != nil {
		t.Fatalf("Failed to get aliyun_cosyvoice adapter: %v", err)
	}

	// 测试获取不存在的适配器
	_, err = registry.GetASR("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent ASR adapter, got nil")
	}

	t.Log("✓ AdapterRegistry integration test passed")
}
