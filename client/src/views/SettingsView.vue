<template>
  <div class="settings-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-title">
        <h2>服务配置</h2>
        <p class="header-subtitle">配置 AI 服务以启用视频翻译功能</p>
      </div>
    </div>

    <!-- 初始化提示 -->
    <el-alert
      v-if="!settings?.is_configured"
      type="warning"
      :closable="false"
      show-icon
      class="mb-20"
    >
      <template #title>
        <span style="font-weight: 600">🚀 快速开始</span>
      </template>
      <p style="margin: 8px 0 0 0">
        请完成 <el-tag type="danger" size="small">必填</el-tag> 标记的三项配置（ASR、翻译、声音克隆），即可开始使用视频翻译功能
      </p>
    </el-alert>

    <!-- 配置表单 -->
    <el-form
      ref="formRef"
      v-loading="loading"
      :model="form"
      :rules="validationRules"
      label-width="140px"
      label-position="right"
    >
      <!-- 核心配置 -->
      <el-card shadow="never" class="config-section">
        <template #header>
          <div class="section-header">
            <span>核心配置</span>
            <el-tag type="danger" size="small">必填</el-tag>
          </div>
        </template>

        <!-- ASR 语音识别 -->
        <div class="config-group">
          <div class="group-title">
            <span>ASR 语音识别</span>
            <el-tooltip content="将视频中的外语语音自动转换成文字" placement="top">
              <el-icon class="help-icon"><QuestionFilled /></el-icon>
            </el-tooltip>
          </div>
          
          <el-form-item label="服务商" prop="asr_provider">
            <el-select v-model="form.asr_provider" placeholder="请选择">
              <el-option label="OpenAI Whisper（推荐）" value="openai-whisper" />
              <el-option label="阿里云语音识别" value="aliyun-asr" />
              <el-option label="Azure Speech" value="azure-speech" />
              <el-option label="Google Cloud Speech" value="google-speech" />
            </el-select>
          </el-form-item>
          
          <el-form-item label="API 密钥" prop="asr_api_key">
            <el-input
              v-model="form.asr_api_key"
              type="password"
              placeholder="至少10个字符"
              show-password
            />
          <el-text type="info" size="small" class="field-hint">
            保存后会显示完整密钥，请注意妥善保管，必要时可随时更新
          </el-text>
          </el-form-item>

          <el-form-item label="自定义端点">
            <el-input v-model="form.asr_endpoint" placeholder="例如: https://api.your-proxy.com">
              <template #append>
                <el-tooltip placement="top">
                  <template #content>
                    <div style="max-width: 300px">
                      <p style="margin: 0 0 8px 0; font-weight: 600;">自定义端点用途：</p>
                      <p style="margin: 0 0 8px 0;">• 使用第三方代理服务</p>
                      <p style="margin: 0 0 8px 0;">• 使用企业内部的 API 网关</p>
                      <p style="margin: 0; color: #909399; font-size: 12px;">留空则使用官方默认端点</p>
                    </div>
                  </template>
                  <el-icon><QuestionFilled /></el-icon>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>
        </div>

        <!-- 翻译服务 -->
        <div class="config-group">
          <div class="group-title">
            <span>翻译服务</span>
            <el-tooltip content="将识别的外语文字翻译成中文" placement="top">
              <el-icon class="help-icon"><QuestionFilled /></el-icon>
            </el-tooltip>
          </div>
          
          <el-form-item label="服务商" prop="translation_provider">
            <el-select v-model="form.translation_provider" placeholder="请选择">
              <el-option label="Google Gemini（推荐）" value="google-gemini" />
              <el-option label="自定义 OpenAI 格式 API" value="openai-compatible" />
              <el-option label="DeepL" value="deepl" />
              <el-option label="Azure Translator" value="azure-translator" />
              <el-option label="火山引擎翻译" value="volcengine-translate" />
            </el-select>
          </el-form-item>
          
          <el-form-item label="API 密钥" prop="translation_api_key">
            <el-input
              v-model="form.translation_api_key"
              type="password"
              placeholder="至少10个字符"
              show-password
            />
          <el-text type="info" size="small" class="field-hint">
            保存后会显示完整密钥，请注意妥善保管，必要时可随时更新
          </el-text>
          </el-form-item>

          <el-form-item label="自定义端点">
            <el-input v-model="form.translation_endpoint" placeholder="例如: https://gemini-balance.xxx.com">
              <template #append>
                <el-tooltip placement="top">
                  <template #content>
                    <div style="max-width: 300px">
                      <p style="margin: 0 0 8px 0; font-weight: 600;">自定义端点用途：</p>
                      <p style="margin: 0 0 8px 0;">• 使用第三方代理服务（如 gemini-balance、one-api 等）</p>
                      <p style="margin: 0 0 8px 0;">• 使用企业内部的 API 网关</p>
                      <p style="margin: 0 0 8px 0;">• 配置自建的 OpenAI 兼容服务</p>
                      <p style="margin: 0; color: #909399; font-size: 12px;">留空则使用官方默认端点</p>
                    </div>
                  </template>
                  <el-icon><QuestionFilled /></el-icon>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item label="翻译风格">
            <el-select v-model="form.translation_video_type" placeholder="可选">
              <el-option label="专业科技" value="professional_tech" />
              <el-option label="口语自然" value="casual_natural" />
              <el-option label="教育严谨" value="educational_rigorous" />
            </el-select>
          </el-form-item>
        </div>

        <!-- 声音克隆 -->
        <div class="config-group">
          <div class="group-title">
            <span>声音克隆</span>
            <el-tooltip content="用中文重新配音，保持原说话人的声音特征" placement="top">
              <el-icon class="help-icon"><QuestionFilled /></el-icon>
            </el-tooltip>
          </div>
          
          <el-form-item label="服务商" prop="voice_cloning_provider">
            <el-select v-model="form.voice_cloning_provider" placeholder="请选择">
              <el-option label="阿里云 CosyVoice（推荐）" value="aliyun-cosyvoice" />
              <el-option label="ElevenLabs" value="elevenlabs" />
            </el-select>
          </el-form-item>
          
          <el-form-item label="API 密钥" prop="voice_cloning_api_key">
            <el-input
              v-model="form.voice_cloning_api_key"
              type="password"
              placeholder="至少10个字符"
              show-password
            />
          <el-text type="info" size="small" class="field-hint">
            保存后会显示完整密钥，请注意妥善保管，必要时可随时更新
          </el-text>
          </el-form-item>

          <el-form-item label="自定义端点">
            <el-input v-model="form.voice_cloning_endpoint" placeholder="例如: https://api.your-proxy.com">
              <template #append>
                <el-tooltip placement="top">
                  <template #content>
                    <div style="max-width: 300px">
                      <p style="margin: 0 0 8px 0; font-weight: 600;">自定义端点用途：</p>
                      <p style="margin: 0 0 8px 0;">• 使用第三方代理服务</p>
                      <p style="margin: 0 0 8px 0;">• 使用企业内部的 API 网关</p>
                      <p style="margin: 0; color: #909399; font-size: 12px;">留空则使用官方默认端点</p>
                    </div>
                  </template>
                  <el-icon><QuestionFilled /></el-icon>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item label="自动选择参考音频">
            <el-switch v-model="form.voice_cloning_auto_select_reference" />
            <el-text type="info" size="small" class="ml-10">推荐开启</el-text>
          </el-form-item>
        </div>
      </el-card>

      <!-- 高级配置（可选） -->
      <el-card shadow="never" class="config-section mt-20">
        <template #header>
          <div class="section-header">
            <span>高级配置</span>
            <el-tag type="info" size="small">可选</el-tag>
          </div>
        </template>

        <!-- 音频分离 -->
        <div class="config-group simple">
          <el-form-item label="音频分离">
            <el-switch v-model="form.audio_separation_enabled" />
            <el-tooltip content="分离人声和背景音乐，提高识别准确率（需要GPU）" placement="top">
              <el-text type="info" size="small" class="ml-10">需要GPU</el-text>
            </el-tooltip>
          </el-form-item>
        </div>

        <!-- 文本润色 -->
        <div class="config-group">
          <el-form-item label="文本润色">
            <el-switch v-model="form.polishing_enabled" />
            <el-tooltip content="翻译前优化识别的原文，纠正错误和断句" placement="top">
              <el-text type="info" size="small" class="ml-10">优化原文准确性</el-text>
            </el-tooltip>
          </el-form-item>
          
          <template v-if="form.polishing_enabled">
            <el-form-item label="服务商" label-width="120px" prop="polishing_provider">
              <el-select v-model="form.polishing_provider" placeholder="请选择" size="small">
                <el-option label="OpenAI GPT-4o" value="openai-gpt4o" />
                <el-option label="自定义 OpenAI 格式" value="openai-compatible" />
                <el-option label="Claude 3.5" value="claude-3.5" />
                <el-option label="Google Gemini" value="google-gemini" />
              </el-select>
            </el-form-item>
            <el-form-item label="API 密钥" label-width="120px" prop="polishing_api_key">
              <el-input
                v-model="form.polishing_api_key"
                type="password"
                placeholder="请输入API密钥"
                show-password
                size="small"
              />
            <el-text type="info" size="small" class="field-hint">
              保存后会显示完整密钥，请注意妥善保管，必要时可随时更新
            </el-text>
            </el-form-item>
            <el-form-item label="自定义端点" label-width="120px" v-if="form.polishing_provider === 'openai-compatible' || form.polishing_provider === 'openai-gpt4o'">
              <el-input
                v-model="form.polishing_endpoint"
                placeholder="例如: https://api.your-proxy.com"
                size="small"
              />
            </el-form-item>
          </template>
        </div>

        <!-- 译文优化 -->
        <div class="config-group">
          <el-form-item label="译文优化">
            <el-switch v-model="form.optimization_enabled" />
            <el-tooltip content="翻译后让中文更自然、符合表达习惯" placement="top">
              <el-text type="info" size="small" class="ml-10">优化译文自然度</el-text>
            </el-tooltip>
          </el-form-item>
          
          <template v-if="form.optimization_enabled">
            <el-form-item label="服务商" label-width="120px" prop="optimization_provider">
              <el-select v-model="form.optimization_provider" placeholder="请选择" size="small">
                <el-option label="OpenAI GPT-4o" value="openai-gpt4o" />
                <el-option label="自定义 OpenAI 格式" value="openai-compatible" />
                <el-option label="Claude 3.5" value="claude-3.5" />
                <el-option label="Google Gemini" value="google-gemini" />
              </el-select>
            </el-form-item>
            <el-form-item label="API 密钥" label-width="120px" prop="optimization_api_key">
              <el-input
                v-model="form.optimization_api_key"
                type="password"
                placeholder="请输入API密钥"
                show-password
                size="small"
              />
            <el-text type="info" size="small" class="field-hint">
              保存后会显示完整密钥，请注意妥善保管，必要时可随时更新
            </el-text>
            </el-form-item>
            <el-form-item label="自定义端点" label-width="120px" v-if="form.optimization_provider === 'openai-compatible' || form.optimization_provider === 'openai-gpt4o'">
              <el-input
                v-model="form.optimization_endpoint"
                placeholder="例如: https://api.your-proxy.com"
                size="small"
              />
            </el-form-item>
          </template>
        </div>
      </el-card>

      <!-- 操作按钮 -->
      <div class="form-actions mt-30">
        <el-button
          v-if="isDev"
          size="large"
          plain
          @click="fillPresetConfig"
        >
          一键填充配置
        </el-button>
        <el-button type="primary" size="large" :loading="saving" @click="saveSettings">
          保存配置
        </el-button>
        <el-button size="large" @click="resetForm">重置</el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { QuestionFilled } from '@element-plus/icons-vue'
import { getSettings, updateSettings } from '@/api/settings-api'
import type { GetSettingsResponse, UpdateSettingsRequest } from '@/api/types'
import { setConfigStatus } from '@/utils/storage'
import { validateConfiguration } from '@/utils/validation'
import axios from 'axios'

const formRef = ref<FormInstance>()

// 原始配置数据
const settings = ref<GetSettingsResponse | null>(null)

// 表单数据
const form = ref<Omit<UpdateSettingsRequest, 'version'>>({
  audio_separation_enabled: false,
  polishing_enabled: false,
  optimization_enabled: false,
  voice_cloning_auto_select_reference: true,
  s2st_provider: '',
  s2st_api_key: ''
})

const isDev = import.meta.env.DEV

// 预设配置（用于快速填充）
const presetConfig: Partial<UpdateSettingsRequest> = {
  processing_mode: 'standard',
  asr_provider: 'aliyun',
  asr_api_key: 'sk-c36a30284fa44101a6e1f556e07c9574',
  asr_endpoint: '',
  audio_separation_enabled: false,
  polishing_enabled: true,
  polishing_provider: 'openai-compatible',
  polishing_api_key: 'sk-aomanoh',
  polishing_endpoint: 'https://balance.aomanoh.com/v1',
  polishing_video_type: '',
  translation_provider: 'openai-compatible',
  translation_api_key: 'sk-aomanoh',
  translation_endpoint: 'https://balance.aomanoh.com/v1',
  translation_video_type: 'casual_natural',
  optimization_enabled: true,
  optimization_provider: 'openai-compatible',
  optimization_api_key: 'sk-aomanoh',
  optimization_endpoint: 'https://balance.aomanoh.com/v1',
  voice_cloning_provider: 'aliyun-cosyvoice',
  voice_cloning_api_key: 'sk-c36a30284fa44101a6e1f556e07c9574',
  voice_cloning_endpoint: '',
  voice_cloning_auto_select_reference: true,
  s2st_provider: '',
  s2st_api_key: ''
}

// 当前版本号
const currentVersion = ref(0)

// 加载状态
const loading = ref(false)
const saving = ref(false)

// 表单验证规则
const validationRules: FormRules = {
  asr_api_key: [
    { required: true, message: '请输入ASR API密钥', trigger: 'blur' }
  ],
  translation_provider: [{ required: true, message: '请选择翻译服务商', trigger: 'change' }],
  translation_api_key: [
    { required: true, message: '请输入翻译API密钥', trigger: 'blur' }
  ],
  voice_cloning_provider: [
    { required: true, message: '请选择声音克隆服务商', trigger: 'change' }
  ],
  voice_cloning_api_key: [
    { required: true, message: '请输入声音克隆API密钥', trigger: 'blur' }
  ]
}

/**
 * 加载配置
 */
const loadSettings = async () => {
  loading.value = true
  try {
    settings.value = await getSettings()
    currentVersion.value = settings.value.version

    // 初始化表单数据
    form.value = {
      processing_mode: settings.value.processing_mode,
      asr_provider: settings.value.asr_provider,
      asr_api_key: settings.value.asr_api_key,
      asr_endpoint: settings.value.asr_endpoint || '',
      audio_separation_enabled: settings.value.audio_separation_enabled,
      polishing_enabled: settings.value.polishing_enabled,
      polishing_provider: settings.value.polishing_provider || '',
      polishing_api_key: settings.value.polishing_api_key || '',
      polishing_endpoint: settings.value.polishing_endpoint || '',
      polishing_custom_prompt: settings.value.polishing_custom_prompt || '',
      polishing_video_type: settings.value.polishing_video_type || '',
      translation_provider: settings.value.translation_provider,
      translation_api_key: settings.value.translation_api_key,
      translation_endpoint: settings.value.translation_endpoint || '',
      translation_video_type: settings.value.translation_video_type || '',
      optimization_enabled: settings.value.optimization_enabled,
      optimization_provider: settings.value.optimization_provider || '',
      optimization_api_key: settings.value.optimization_api_key || '',
      optimization_endpoint: settings.value.optimization_endpoint || '',
      s2st_provider: settings.value.s2st_provider || '',
      s2st_api_key: settings.value.s2st_api_key || '',
      voice_cloning_provider: settings.value.voice_cloning_provider,
      voice_cloning_api_key: settings.value.voice_cloning_api_key,
      voice_cloning_endpoint: settings.value.voice_cloning_endpoint || '',
      voice_cloning_auto_select_reference: settings.value.voice_cloning_auto_select_reference
    }
  } catch (error) {
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}

/**
 * 保存配置
 */
const stringFields: Array<keyof UpdateSettingsRequest> = [
  'processing_mode',
  'asr_provider',
  'asr_api_key',
  'asr_endpoint',
  'translation_endpoint',
  'voice_cloning_endpoint',
  'polishing_provider',
  'polishing_api_key',
  'polishing_endpoint',
  'polishing_custom_prompt',
  'polishing_video_type',
  'optimization_provider',
  'optimization_api_key',
  'optimization_endpoint',
  's2st_provider',
  's2st_api_key',
  'voice_cloning_provider',
  'voice_cloning_api_key',
  'translation_provider',
  'translation_api_key',
  'translation_video_type'
]

const saveSettings = async () => {
  // 表单验证
  stringFields.forEach(field => {
    const value = form.value[field]
    if (typeof value === 'string') {
      form.value[field] = value.trim() as any
    }
  })

  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  // 格式验证（前端拦截）
  const formatValidation = validateConfiguration(form.value)
  if (!formatValidation.valid) {
    await ElMessageBox.alert(
      `<div style="max-height: 300px; overflow-y: auto;">
        <p style="margin-bottom: 12px; font-weight: 600;">发现以下配置问题：</p>
        <ul style="margin: 0; padding-left: 20px;">
          ${formatValidation.errors.map(err => `<li style="margin-bottom: 8px;">${err}</li>`).join('')}
        </ul>
        <p style="margin-top: 12px; color: #909399; font-size: 13px;">
          💡 提示：配置错误可能导致任务处理失败，请仔细检查后重新保存
        </p>
      </div>`,
      '配置格式验证失败',
      {
        confirmButtonText: '我知道了',
        dangerouslyUseHTMLString: true,
        type: 'warning'
      }
    )
    return
  }

  // HTTPS 安全提示
  const endpointsToCheck = [
    form.value.asr_endpoint,
    form.value.translation_endpoint,
    form.value.voice_cloning_endpoint,
    form.value.polishing_endpoint,
    form.value.optimization_endpoint
  ].filter(Boolean)

  const hasHttpEndpoint = endpointsToCheck.some(
    endpoint => endpoint && endpoint.startsWith('http://') && !endpoint.includes('localhost')
  )

  if (hasHttpEndpoint) {
    try {
      await ElMessageBox.confirm(
        '检测到您使用了 HTTP 协议的自定义端点。为保护 API 密钥安全，强烈建议使用 HTTPS 协议。是否继续保存？',
        '安全提示',
        {
          confirmButtonText: '继续保存',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return // 用户取消
    }
  }

  saving.value = true
  try {
    const request: UpdateSettingsRequest = {
      ...form.value,
      version: currentVersion.value
    }

    const response = await updateSettings(request)
    currentVersion.value = response.version
    ElMessage.success(response.message)

    // 更新localStorage缓存
    const hasRequiredConfig =
      form.value.asr_api_key &&
      form.value.translation_api_key &&
      form.value.voice_cloning_api_key
    setConfigStatus(Boolean(hasRequiredConfig))
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 409) {
      // 版本冲突，重新加载
      ElMessage.warning('配置已被修改，正在刷新...')
      await loadSettings()
    }
  } finally {
    saving.value = false
  }
}

/**
 * 一键填充预设配置
 */
const fillPresetConfig = () => {
  form.value = {
    ...form.value,
    ...presetConfig
  }
  ElMessage.success('已填充预设配置，请检查后保存')
}

/**
 * 重置表单
 */
const resetForm = () => {
  if (settings.value) {
    loadSettings()
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 32px 24px;
}

.page-header {
  margin-bottom: 24px;

  .header-title h2 {
    font-size: 28px;
    font-weight: 700;
    color: #1f2937;
    margin: 0 0 8px 0;
  }

  .header-subtitle {
    font-size: 14px;
    color: #6b7280;
    margin: 0;
  }
}

.config-section {
  border-radius: var(--app-border-radius);
  border: 1px solid #e5e7eb;

  .section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
    color: #1f2937;
  }
}

.config-group {
  padding: 20px 0;
  border-bottom: 1px dashed #e5e7eb;

  &:last-child {
    border-bottom: none;
  }

  &.simple {
    padding: 12px 0;
  }

  .group-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 15px;
    font-weight: 600;
    color: #374151;
    margin-bottom: 16px;

    .help-icon {
      color: #9ca3af;
      cursor: help;
      font-size: 16px;

      &:hover {
        color: var(--el-color-primary);
      }
    }
  }

  :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  :deep(.el-form-item__label) {
    font-weight: 500;
    color: #4b5563;
  }
}

.form-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 24px 0;
}

.ml-10 {
  margin-left: 10px;
}

.mt-20 {
  margin-top: 20px;
}

.mt-30 {
  margin-top: 30px;
}

.mb-20 {
  margin-bottom: 20px;
}

.field-hint {
  display: block;
  margin-top: 6px;
  color: #909399;
}
</style>
