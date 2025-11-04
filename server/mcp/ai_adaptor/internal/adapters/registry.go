package adapters

import (
	"fmt"
	"sync"
)

// AdapterRegistry 适配器注册表，管理所有已注册的适配器实例
type AdapterRegistry struct {
	asrAdapters          map[string]ASRAdapter
	translationAdapters  map[string]TranslationAdapter
	llmAdapters          map[string]LLMAdapter
	voiceCloningAdapters map[string]VoiceCloningAdapter
	mu                   sync.RWMutex
}

// NewAdapterRegistry 创建新的适配器注册表
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		asrAdapters:          make(map[string]ASRAdapter),
		translationAdapters:  make(map[string]TranslationAdapter),
		llmAdapters:          make(map[string]LLMAdapter),
		voiceCloningAdapters: make(map[string]VoiceCloningAdapter),
	}
}

// RegisterASR 注册 ASR 适配器
func (r *AdapterRegistry) RegisterASR(name string, adapter ASRAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asrAdapters[name] = adapter
}

// RegisterTranslation 注册翻译适配器
func (r *AdapterRegistry) RegisterTranslation(name string, adapter TranslationAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.translationAdapters[name] = adapter
}

// RegisterLLM 注册 LLM 适配器
func (r *AdapterRegistry) RegisterLLM(name string, adapter LLMAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmAdapters[name] = adapter
}

// RegisterVoiceCloning 注册声音克隆适配器
func (r *AdapterRegistry) RegisterVoiceCloning(name string, adapter VoiceCloningAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.voiceCloningAdapters[name] = adapter
}

// GetASR 获取 ASR 适配器
func (r *AdapterRegistry) GetASR(name string) (ASRAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, ok := r.asrAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported ASR provider: %s", name)
	}
	return adapter, nil
}

// GetTranslation 获取翻译适配器
func (r *AdapterRegistry) GetTranslation(name string) (TranslationAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, ok := r.translationAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported translation provider: %s", name)
	}
	return adapter, nil
}

// GetLLM 获取 LLM 适配器
func (r *AdapterRegistry) GetLLM(name string) (LLMAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, ok := r.llmAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported LLM provider: %s", name)
	}
	return adapter, nil
}

// GetVoiceCloning 获取声音克隆适配器
func (r *AdapterRegistry) GetVoiceCloning(name string) (VoiceCloningAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, ok := r.voiceCloningAdapters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported voice cloning provider: %s", name)
	}
	return adapter, nil
}

// ListASRProviders 列出所有已注册的 ASR 提供商
func (r *AdapterRegistry) ListASRProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]string, 0, len(r.asrAdapters))
	for name := range r.asrAdapters {
		providers = append(providers, name)
	}
	return providers
}

// ListTranslationProviders 列出所有已注册的翻译提供商
func (r *AdapterRegistry) ListTranslationProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]string, 0, len(r.translationAdapters))
	for name := range r.translationAdapters {
		providers = append(providers, name)
	}
	return providers
}

// ListLLMProviders 列出所有已注册的 LLM 提供商
func (r *AdapterRegistry) ListLLMProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]string, 0, len(r.llmAdapters))
	for name := range r.llmAdapters {
		providers = append(providers, name)
	}
	return providers
}

// ListVoiceCloningProviders 列出所有已注册的声音克隆提供商
func (r *AdapterRegistry) ListVoiceCloningProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	providers := make([]string, 0, len(r.voiceCloningAdapters))
	for name := range r.voiceCloningAdapters {
		providers = append(providers, name)
	}
	return providers
}

