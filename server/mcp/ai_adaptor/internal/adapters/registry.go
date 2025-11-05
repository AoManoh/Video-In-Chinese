package adapters

import (
	"fmt"
	"sync"
)

// AdapterRegistry 维护各类适配器的注册表，支持并发安全的查询与注册。
//
// 功能说明:
//   - 保存 ASR、翻译、LLM、声音克隆等适配器实例，并提供按名称检索与列出功能。
//
// 设计决策:
//   - 使用 RWMutex 保证并发安全，读操作在大量查询场景下无需阻塞写操作。
type AdapterRegistry struct {
	asrAdapters          map[string]ASRAdapter
	translationAdapters  map[string]TranslationAdapter
	llmAdapters          map[string]LLMAdapter
	voiceCloningAdapters map[string]VoiceCloningAdapter
	mu                   sync.RWMutex
}

// NewAdapterRegistry 创建并初始化适配器注册表。
//
// 功能说明:
//   - 为每类适配器创建独立的 map 容器。
//
// 返回值说明:
//
//	*AdapterRegistry: 空注册表实例。
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		asrAdapters:          make(map[string]ASRAdapter),
		translationAdapters:  make(map[string]TranslationAdapter),
		llmAdapters:          make(map[string]LLMAdapter),
		voiceCloningAdapters: make(map[string]VoiceCloningAdapter),
	}
}

// RegisterASR 注册 ASR 适配器实例，名称重复时覆盖旧值。
//
// 注意事项:
//   - 写操作加写锁，确保并发安全。
func (r *AdapterRegistry) RegisterASR(name string, adapter ASRAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asrAdapters[name] = adapter
}

// RegisterTranslation 注册翻译适配器实例。
func (r *AdapterRegistry) RegisterTranslation(name string, adapter TranslationAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.translationAdapters[name] = adapter
}

// RegisterLLM 注册 LLM 适配器实例。
func (r *AdapterRegistry) RegisterLLM(name string, adapter LLMAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmAdapters[name] = adapter
}

// RegisterVoiceCloning 注册声音克隆适配器实例。
func (r *AdapterRegistry) RegisterVoiceCloning(name string, adapter VoiceCloningAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.voiceCloningAdapters[name] = adapter
}

// GetASR 按名称获取 ASR 适配器，未注册时返回错误。
func (r *AdapterRegistry) GetASR(name string) (ASRAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.asrAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported ASR provider: %s", name)
	}
	return adapter, nil
}

// GetTranslation 按名称获取翻译适配器，未注册时返回错误。
func (r *AdapterRegistry) GetTranslation(name string) (TranslationAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.translationAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported translation provider: %s", name)
	}
	return adapter, nil
}

// GetLLM 按名称获取 LLM 适配器，未注册时返回错误。
func (r *AdapterRegistry) GetLLM(name string) (LLMAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.llmAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported LLM provider: %s", name)
	}
	return adapter, nil
}

// GetVoiceCloning 按名称获取声音克隆适配器，未注册时返回错误。
func (r *AdapterRegistry) GetVoiceCloning(name string) (VoiceCloningAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.voiceCloningAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported voice cloning provider: %s", name)
	}
	return adapter, nil
}

// ListASRProviders 返回已注册的 ASR 适配器名称列表。
func (r *AdapterRegistry) ListASRProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.asrAdapters))
	for name := range r.asrAdapters {
		providers = append(providers, name)
	}
	return providers
}

// ListTranslationProviders 返回已注册的翻译适配器名称列表。
func (r *AdapterRegistry) ListTranslationProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.translationAdapters))
	for name := range r.translationAdapters {
		providers = append(providers, name)
	}
	return providers
}

// ListLLMProviders 返回已注册的 LLM 适配器名称列表。
func (r *AdapterRegistry) ListLLMProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.llmAdapters))
	for name := range r.llmAdapters {
		providers = append(providers, name)
	}
	return providers
}

// ListVoiceCloningProviders 返回已注册的声音克隆适配器名称列表。
func (r *AdapterRegistry) ListVoiceCloningProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.voiceCloningAdapters))
	for name := range r.voiceCloningAdapters {
		providers = append(providers, name)
	}
	return providers
}
