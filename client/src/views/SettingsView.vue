<template>
  <div class="settings-page">
    <el-page-header @back="goBack" content="配置管理" />

    <!-- 初始化向导 -->
    <el-alert
      v-if="!settings?.is_configured"
      title="欢迎使用视频翻译服务"
      type="info"
      description="请先完成以下基本配置（ASR、翻译、声音克隆），然后即可开始上传视频"
      :closable="false"
      show-icon
      class="mt-20 mb-20"
    />

    <!-- 配置表单 -->
    <el-form
      ref="formRef"
      v-loading="loading"
      :model="form"
      :rules="validationRules"
      label-width="160px"
      label-position="right"
      class="settings-form"
    >
      <!-- 基础配置 -->
      <el-divider content-position="left">基础配置</el-divider>
      <el-form-item label="处理模式">
        <el-tag>标准模式（Standard）</el-tag>
        <el-text type="info" size="small" class="ml-10">V1.0仅支持标准模式</el-text>
      </el-form-item>

      <!-- ASR服务配置（必需） -->
      <el-divider content-position="left">
        ASR服务配置
        <el-tag type="danger" size="small">必需</el-tag>
      </el-divider>
      <el-form-item label="服务商" prop="asr_provider">
        <el-select v-model="form.asr_provider" placeholder="请选择ASR服务商">
          <el-option label="OpenAI Whisper" value="openai-whisper" />
          <el-option label="阿里云语音识别" value="aliyun-asr" />
          <el-option label="Azure Speech" value="azure-speech" />
          <el-option label="Google Cloud Speech" value="google-speech" />
        </el-select>
      </el-form-item>
      <el-form-item label="API密钥" prop="asr_api_key">
        <el-input
          v-model="form.asr_api_key"
          type="password"
          placeholder="请输入API密钥"
          show-password
        />
      </el-form-item>
      <el-form-item label="自定义端点">
        <el-input v-model="form.asr_endpoint" placeholder="可选，留空使用默认端点" />
      </el-form-item>

      <!-- 音频分离配置 -->
      <el-divider content-position="left">
        音频分离配置
        <el-tag type="info" size="small">可选</el-tag>
      </el-divider>
      <el-form-item label="启用音频分离">
        <el-switch v-model="form.audio_separation_enabled" />
        <el-text type="warning" size="small" class="ml-10">需要GPU支持</el-text>
      </el-form-item>

      <!-- 文本润色配置 -->
      <el-divider content-position="left">
        文本润色配置
        <el-tag type="info" size="small">可选</el-tag>
      </el-divider>
      <el-form-item label="启用文本润色">
        <el-switch v-model="form.polishing_enabled" />
      </el-form-item>
      <template v-if="form.polishing_enabled">
        <el-form-item label="服务商" prop="polishing_provider">
          <el-select v-model="form.polishing_provider" placeholder="请选择服务商">
            <el-option label="OpenAI GPT-4o" value="openai-gpt4o" />
            <el-option label="Claude 3.5" value="claude-3.5" />
            <el-option label="Google Gemini" value="google-gemini" />
            <el-option label="火山引擎 Doubao" value="volcengine-doubao" />
          </el-select>
        </el-form-item>
        <el-form-item label="API密钥" prop="polishing_api_key">
          <el-input
            v-model="form.polishing_api_key"
            type="password"
            placeholder="请输入API密钥"
            show-password
          />
        </el-form-item>
        <el-form-item label="自定义Prompt">
          <el-input
            v-model="form.polishing_custom_prompt"
            type="textarea"
            :rows="3"
            placeholder="可选"
          />
        </el-form-item>
        <el-form-item label="预设类型">
          <el-select v-model="form.polishing_video_type" placeholder="可选">
            <el-option label="专业科技" value="professional_tech" />
            <el-option label="口语自然" value="casual_natural" />
            <el-option label="教育严谨" value="educational_rigorous" />
            <el-option label="默认" value="default" />
          </el-select>
        </el-form-item>
      </template>

      <!-- 翻译服务配置（必需） -->
      <el-divider content-position="left">
        翻译服务配置
        <el-tag type="danger" size="small">必需</el-tag>
      </el-divider>
      <el-form-item label="服务商" prop="translation_provider">
        <el-select v-model="form.translation_provider" placeholder="请选择翻译服务商">
          <el-option label="Google Gemini" value="google-gemini" />
          <el-option label="DeepL" value="deepl" />
          <el-option label="Azure Translator" value="azure-translator" />
          <el-option label="火山引擎翻译" value="volcengine-translate" />
        </el-select>
      </el-form-item>
      <el-form-item label="API密钥" prop="translation_api_key">
        <el-input
          v-model="form.translation_api_key"
          type="password"
          placeholder="请输入API密钥"
          show-password
        />
      </el-form-item>
      <el-form-item label="自定义端点">
        <el-input v-model="form.translation_endpoint" placeholder="可选，留空使用默认端点" />
      </el-form-item>
      <el-form-item label="预设类型">
        <el-select v-model="form.translation_video_type" placeholder="可选">
          <el-option label="专业科技" value="professional_tech" />
          <el-option label="口语自然" value="casual_natural" />
          <el-option label="教育严谨" value="educational_rigorous" />
          <el-option label="默认" value="default" />
        </el-select>
      </el-form-item>

      <!-- 译文优化配置 -->
      <el-divider content-position="left">
        译文优化配置
        <el-tag type="info" size="small">可选</el-tag>
      </el-divider>
      <el-form-item label="启用译文优化">
        <el-switch v-model="form.optimization_enabled" />
      </el-form-item>
      <template v-if="form.optimization_enabled">
        <el-form-item label="服务商" prop="optimization_provider">
          <el-select v-model="form.optimization_provider" placeholder="请选择服务商">
            <el-option label="OpenAI GPT-4o" value="openai-gpt4o" />
            <el-option label="Claude 3.5" value="claude-3.5" />
            <el-option label="Google Gemini" value="google-gemini" />
            <el-option label="火山引擎 Doubao" value="volcengine-doubao" />
          </el-select>
        </el-form-item>
        <el-form-item label="API密钥" prop="optimization_api_key">
          <el-input
            v-model="form.optimization_api_key"
            type="password"
            placeholder="请输入API密钥"
            show-password
          />
        </el-form-item>
      </template>

      <!-- 声音克隆配置（必需） -->
      <el-divider content-position="left">
        声音克隆配置
        <el-tag type="danger" size="small">必需</el-tag>
      </el-divider>
      <el-form-item label="服务商" prop="voice_cloning_provider">
        <el-select v-model="form.voice_cloning_provider" placeholder="请选择声音克隆服务商">
          <el-option label="阿里云 CosyVoice" value="aliyun-cosyvoice" />
          <el-option label="ElevenLabs" value="elevenlabs" />
        </el-select>
      </el-form-item>
      <el-form-item label="API密钥" prop="voice_cloning_api_key">
        <el-input
          v-model="form.voice_cloning_api_key"
          type="password"
          placeholder="请输入API密钥"
          show-password
        />
      </el-form-item>
      <el-form-item label="自定义端点">
        <el-input v-model="form.voice_cloning_endpoint" placeholder="可选，留空使用默认端点" />
      </el-form-item>
      <el-form-item label="自动选择参考音频">
        <el-switch v-model="form.voice_cloning_auto_select_reference" />
      </el-form-item>

      <!-- 操作按钮 -->
      <el-form-item>
        <el-button type="primary" :loading="saving" @click="saveSettings">保存配置</el-button>
        <el-button @click="resetForm">重置</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { getSettings, updateSettings } from '@/api/settings-api'
import type { GetSettingsResponse, UpdateSettingsRequest } from '@/api/types'
import { setConfigStatus } from '@/utils/storage'
import axios from 'axios'

const router = useRouter()
const formRef = ref<FormInstance>()

// 原始配置数据
const settings = ref<GetSettingsResponse | null>(null)

// 表单数据
const form = ref<Partial<UpdateSettingsRequest>>({
  audio_separation_enabled: false,
  polishing_enabled: false,
  optimization_enabled: false,
  voice_cloning_auto_select_reference: true
})

// 当前版本号
const currentVersion = ref(0)

// 加载状态
const loading = ref(false)
const saving = ref(false)

// 表单验证规则
const validationRules: FormRules = {
  asr_provider: [{ required: true, message: '请选择ASR服务商', trigger: 'change' }],
  asr_api_key: [
    { required: true, message: '请输入ASR API密钥', trigger: 'blur' },
    { min: 10, message: 'API密钥长度至少10个字符', trigger: 'blur' }
  ],
  translation_provider: [{ required: true, message: '请选择翻译服务商', trigger: 'change' }],
  translation_api_key: [
    { required: true, message: '请输入翻译API密钥', trigger: 'blur' },
    { min: 10, message: 'API密钥长度至少10个字符', trigger: 'blur' }
  ],
  voice_cloning_provider: [
    { required: true, message: '请选择声音克隆服务商', trigger: 'change' }
  ],
  voice_cloning_api_key: [
    { required: true, message: '请输入声音克隆API密钥', trigger: 'blur' },
    { min: 10, message: 'API密钥长度至少10个字符', trigger: 'blur' }
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
      version: settings.value.version,
      processing_mode: settings.value.processing_mode,
      asr_provider: settings.value.asr_provider,
      asr_api_key: settings.value.asr_api_key,
      asr_endpoint: settings.value.asr_endpoint || '',
      audio_separation_enabled: settings.value.audio_separation_enabled,
      polishing_enabled: settings.value.polishing_enabled,
      polishing_provider: settings.value.polishing_provider || '',
      polishing_api_key: settings.value.polishing_api_key || '',
      polishing_custom_prompt: settings.value.polishing_custom_prompt || '',
      polishing_video_type: settings.value.polishing_video_type || '',
      translation_provider: settings.value.translation_provider,
      translation_api_key: settings.value.translation_api_key,
      translation_endpoint: settings.value.translation_endpoint || '',
      translation_video_type: settings.value.translation_video_type || '',
      optimization_enabled: settings.value.optimization_enabled,
      optimization_provider: settings.value.optimization_provider || '',
      optimization_api_key: settings.value.optimization_api_key || '',
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
const saveSettings = async () => {
  // 表单验证
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const request: UpdateSettingsRequest = {
      version: currentVersion.value,
      ...form.value
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
 * 重置表单
 */
const resetForm = () => {
  if (settings.value) {
    loadSettings()
  }
}

/**
 * 返回上一页
 */
const goBack = () => {
  router.back()
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
}

.settings-form {
  margin-top: 20px;
}

.ml-10 {
  margin-left: 10px;
}

.mt-20 {
  margin-top: 20px;
}

.mb-20 {
  margin-bottom: 20px;
}
</style>

