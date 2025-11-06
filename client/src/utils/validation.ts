/**
 * 配置验证工具函数
 * 
 * 提供前端格式验证，避免无效配置提交到后端
 * Phase 1: 仅验证格式，不调用外部 API
 */

/**
 * API 密钥格式验证规则
 */
interface KeyFormatRule {
  prefix?: string
  minLength: number
  maxLength?: number
  pattern?: RegExp
  description: string
}

/**
 * 各服务商的 API 密钥格式规则
 */
const API_KEY_FORMAT_RULES: Record<string, KeyFormatRule> = {
  // OpenAI 系列
  'openai-whisper': {
    prefix: 'sk-',
    minLength: 20,
    description: 'OpenAI API 密钥格式：sk-xxx，至少 20 个字符'
  },
  'openai-gpt4o': {
    prefix: 'sk-',
    minLength: 20,
    description: 'OpenAI API 密钥格式：sk-xxx，至少 20 个字符'
  },
  'openai-compatible': {
    prefix: 'sk-',
    minLength: 10,
    description: '自定义服务 API 密钥格式：通常为 sk-xxx，至少 10 个字符'
  },

  // Google 系列
  'google-gemini': {
    minLength: 39,
    description: 'Google API 密钥格式：39 个字符的字母数字组合'
  },
  'google-speech': {
    minLength: 39,
    description: 'Google Cloud API 密钥格式：39 个字符'
  },

  // 阿里云系列
  'aliyun-asr': {
    minLength: 16,
    description: '阿里云 AccessKey ID 格式：16-30 个字符'
  },
  'aliyun-cosyvoice': {
    minLength: 16,
    description: '阿里云 AccessKey ID 格式：16-30 个字符'
  },

  // Azure 系列
  'azure-speech': {
    minLength: 32,
    maxLength: 32,
    description: 'Azure 订阅密钥格式：32 个字符的十六进制字符串'
  },
  'azure-translator': {
    minLength: 32,
    maxLength: 32,
    description: 'Azure 订阅密钥格式：32 个字符'
  },

  // DeepL
  'deepl': {
    minLength: 39,
    pattern: /^[a-f0-9-]+:fx$/,
    description: 'DeepL API 密钥格式：以 :fx 结尾的 UUID 格式'
  },

  // Claude
  'claude-3.5': {
    prefix: 'sk-ant-',
    minLength: 20,
    description: 'Claude API 密钥格式：sk-ant-xxx'
  },

  // 火山引擎
  'volcengine-translate': {
    minLength: 20,
    description: '火山引擎 API 密钥格式：至少 20 个字符'
  },
  'volcengine-doubao': {
    minLength: 20,
    description: '火山引擎 API 密钥格式：至少 20 个字符'
  },

  // ElevenLabs
  'elevenlabs': {
    minLength: 32,
    description: 'ElevenLabs API 密钥格式：32 个字符'
  }
}

/**
 * 验证 API 密钥格式
 * 
 * @param provider 服务商标识
 * @param apiKey API 密钥
 * @returns { valid: boolean, message: string }
 */
export const validateAPIKeyFormat = (
  provider: string,
  apiKey: string
): { valid: boolean; message: string } => {
  // 如果是脱敏格式（包含 ***），跳过验证
  if (apiKey.includes('***')) {
    return { valid: true, message: '' }
  }

  // 如果密钥为空
  if (!apiKey || apiKey.trim() === '') {
    return { valid: false, message: 'API 密钥不能为空' }
  }

  // 获取该服务商的格式规则
  const rule = API_KEY_FORMAT_RULES[provider]

  // 如果没有特定规则，使用通用规则（至少 10 个字符）
  if (!rule) {
    if (apiKey.length < 10) {
      return { valid: false, message: 'API 密钥长度至少 10 个字符' }
    }
    return { valid: true, message: '' }
  }

  // 验证长度
  if (apiKey.length < rule.minLength) {
    return {
      valid: false,
      message: `${rule.description}（当前长度：${apiKey.length}）`
    }
  }

  if (rule.maxLength && apiKey.length > rule.maxLength) {
    return {
      valid: false,
      message: `API 密钥长度不应超过 ${rule.maxLength} 个字符`
    }
  }

  // 验证前缀
  if (rule.prefix && !apiKey.startsWith(rule.prefix)) {
    return {
      valid: false,
      message: `${rule.description}（应以 ${rule.prefix} 开头）`
    }
  }

  // 验证正则表达式
  if (rule.pattern && !rule.pattern.test(apiKey)) {
    return {
      valid: false,
      message: rule.description
    }
  }

  // 验证通过
  return { valid: true, message: '' }
}

/**
 * 验证自定义端点格式
 * 
 * @param endpoint 端点 URL
 * @returns { valid: boolean, message: string }
 */
export const validateEndpointFormat = (
  endpoint: string
): { valid: boolean; message: string } => {
  // 端点为空是允许的（使用默认端点）
  if (!endpoint || endpoint.trim() === '') {
    return { valid: true, message: '' }
  }

  // 必须是 HTTP 或 HTTPS URL
  if (!endpoint.startsWith('http://') && !endpoint.startsWith('https://')) {
    return {
      valid: false,
      message: '端点地址必须以 http:// 或 https:// 开头'
    }
  }

  // 强烈建议使用 HTTPS
  if (endpoint.startsWith('http://') && !endpoint.includes('localhost')) {
    return {
      valid: true,
      message: '⚠️ 建议使用 HTTPS 保护 API 密钥安全'
    }
  }

  // 基本 URL 格式验证
  try {
    const url = new URL(endpoint)

    // 检查主机名是否有效
    if (!url.hostname) {
      return {
        valid: false,
        message: '端点地址格式不正确'
      }
    }

    // 成功
    return { valid: true, message: '' }
  } catch (error) {
    return {
      valid: false,
      message: '端点地址格式不正确，请检查 URL 格式'
    }
  }
}

/**
 * 验证配置是否完整（必填字段）
 * 
 * @param form 表单数据
 * @returns { valid: boolean, missing: string[] }
 */
export const validateRequiredConfig = (form: {
  asr_provider?: string
  asr_api_key?: string
  translation_provider?: string
  translation_api_key?: string
  voice_cloning_provider?: string
  voice_cloning_api_key?: string
  s2st_provider?: string
  s2st_api_key?: string
}): { valid: boolean; missing: string[] } => {
  const missing: string[] = []

  // 检查必填的三个服务
  if (!form.asr_provider) missing.push('ASR 服务商')
  if (!form.asr_api_key || form.asr_api_key.includes('***')) {
    missing.push('ASR API 密钥')
  }

  if (!form.translation_provider) missing.push('翻译服务商')
  if (!form.translation_api_key || form.translation_api_key.includes('***')) {
    missing.push('翻译 API 密钥')
  }

  if (!form.voice_cloning_provider) missing.push('声音克隆服务商')
  if (!form.voice_cloning_api_key || form.voice_cloning_api_key.includes('***')) {
    missing.push('声音克隆 API 密钥')
  }

  return {
    valid: missing.length === 0,
    missing
  }
}

/**
 * 获取服务商的配置建议
 * 
 * @param provider 服务商标识
 * @returns 配置建议文本
 */
export const getProviderConfigTips = (provider: string): string => {
  const tips: Record<string, string> = {
    'openai-whisper':
      '💡 获取方式：访问 https://platform.openai.com/api-keys 创建 API 密钥',
    'openai-gpt4o':
      '💡 获取方式：访问 https://platform.openai.com/api-keys 创建 API 密钥',
    'openai-compatible':
      '💡 使用代理服务（如 gemini-balance、one-api）时，请填写代理服务提供的密钥',
    'google-gemini':
      '💡 获取方式：访问 https://makersuite.google.com/app/apikey 创建 API 密钥',
    'google-speech':
      '💡 获取方式：访问 Google Cloud Console 创建 API 密钥',
    'aliyun-asr':
      '💡 获取方式：访问阿里云控制台，创建 AccessKey ID 和 AccessKey Secret',
    'aliyun-cosyvoice':
      '💡 获取方式：访问阿里云控制台，创建 AccessKey ID 和 AccessKey Secret',
    'azure-speech':
      '💡 获取方式：访问 Azure Portal，在语音服务中查看密钥',
    'azure-translator':
      '💡 获取方式：访问 Azure Portal，在翻译服务中查看密钥',
    'deepl': '💡 获取方式：访问 https://www.deepl.com/pro-api 注册并获取 API 密钥',
    'claude-3.5':
      '💡 获取方式：访问 https://console.anthropic.com/ 创建 API 密钥',
    'elevenlabs':
      '💡 获取方式：访问 https://elevenlabs.io/app/settings 查看 API 密钥'
  }

  return tips[provider] || '💡 请参考服务商文档获取 API 密钥'
}

/**
 * 获取常见的配置错误原因和解决建议
 * 
 * @param errorMessage 错误信息
 * @returns 解决建议
 */
export const getConfigErrorSuggestion = (errorMessage: string): string => {
  const suggestions: Record<string, string> = {
    '401': '🔧 API 密钥无效或已过期，请在配置页面检查并更新密钥',
    '403': '🔧 API 密钥权限不足，请确认密钥拥有必要的权限',
    '429': '🔧 API 配额不足或请求频率过高，请检查账户配额或稍后重试',
    'API 密钥无效': '🔧 请前往配置页面更新 API 密钥',
    'API 配额不足': '🔧 请检查外部 API 账户余额并升级套餐',
    '配置错误': '🔧 请前往配置页面检查必填项是否完整',
    '解密失败': '🔧 配置数据可能损坏，请重新保存配置'
  }

  for (const [key, suggestion] of Object.entries(suggestions)) {
    if (errorMessage.includes(key)) {
      return suggestion
    }
  }

  return '🔧 请检查配置是否正确，必要时重新保存配置'
}

/**
 * 综合配置验证（用于保存前检查）
 * 
 * @param form 表单数据
 * @returns { valid: boolean, errors: string[] }
 */
export const validateConfiguration = (form: any): { valid: boolean; errors: string[] } => {
  const errors: string[] = []

  // 1. 验证必填字段
  const requiredCheck = validateRequiredConfig(form)
  if (!requiredCheck.valid) {
    requiredCheck.missing.forEach(field => {
      errors.push(`❌ ${field}未配置`)
    })
  }

  // 2. 验证 ASR API 密钥格式
  if (form.asr_provider && form.asr_api_key && !form.asr_api_key.includes('***')) {
    const keyValidation = validateAPIKeyFormat(form.asr_provider, form.asr_api_key)
    if (!keyValidation.valid) {
      errors.push(`❌ ASR API 密钥格式错误：${keyValidation.message}`)
    }
  }

  // 3. 验证翻译 API 密钥格式
  if (
    form.translation_provider &&
    form.translation_api_key &&
    !form.translation_api_key.includes('***')
  ) {
    const keyValidation = validateAPIKeyFormat(form.translation_provider, form.translation_api_key)
    if (!keyValidation.valid) {
      errors.push(`❌ 翻译 API 密钥格式错误：${keyValidation.message}`)
    }
  }

  // 4. 验证声音克隆 API 密钥格式
  if (
    form.voice_cloning_provider &&
    form.voice_cloning_api_key &&
    !form.voice_cloning_api_key.includes('***')
  ) {
    const keyValidation = validateAPIKeyFormat(
      form.voice_cloning_provider,
      form.voice_cloning_api_key
    )
    if (!keyValidation.valid) {
      errors.push(`❌ 声音克隆 API 密钥格式错误：${keyValidation.message}`)
    }
  }

  // 5. 验证文本润色 API 密钥格式（如果启用）
  if (
    form.polishing_enabled &&
    form.polishing_provider &&
    form.polishing_api_key &&
    !form.polishing_api_key.includes('***')
  ) {
    const keyValidation = validateAPIKeyFormat(form.polishing_provider, form.polishing_api_key)
    if (!keyValidation.valid) {
      errors.push(`⚠️ 文本润色 API 密钥格式错误：${keyValidation.message}`)
    }
  }

  // 6. 验证译文优化 API 密钥格式（如果启用）
  if (
    form.optimization_enabled &&
    form.optimization_provider &&
    form.optimization_api_key &&
    !form.optimization_api_key.includes('***')
  ) {
    const keyValidation = validateAPIKeyFormat(
      form.optimization_provider,
      form.optimization_api_key
    )
    if (!keyValidation.valid) {
      errors.push(`⚠️ 译文优化 API 密钥格式错误：${keyValidation.message}`)
    }
  }

  // 7. 验证自定义端点格式
  const endpoints = [
    { field: 'ASR', value: form.asr_endpoint },
    { field: '翻译', value: form.translation_endpoint },
    { field: '声音克隆', value: form.voice_cloning_endpoint },
    { field: '文本润色', value: form.polishing_endpoint },
    { field: '译文优化', value: form.optimization_endpoint }
  ]

  endpoints.forEach(({ field, value }) => {
    if (value && value.trim()) {
      const endpointValidation = validateEndpointFormat(value)
      if (!endpointValidation.valid) {
        errors.push(`❌ ${field}自定义端点格式错误：${endpointValidation.message}`)
      } else if (endpointValidation.message) {
        // 警告信息（如使用 HTTP）
        errors.push(`⚠️ ${field}：${endpointValidation.message}`)
      }
    }
  })

  return {
    valid: errors.length === 0,
    errors
  }
}

/**
 * 获取 API 密钥配置提示（兼容旧版本）
 * 
 * @param provider 服务商标识
 * @returns 配置提示
 */
export const getAPIKeyHint = (provider: string): string => {
  return getProviderConfigTips(provider)
}
