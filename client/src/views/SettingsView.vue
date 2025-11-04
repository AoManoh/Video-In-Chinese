<template>
  <div class="settings-page">
    <el-page-header @back="goBack" content="配置管理" />

    <!-- 初始化向导 -->
    <el-alert
      v-if="!settings?.is_configured"
      title="欢迎使用视频翻译服务 🎉"
      type="info"
      :closable="false"
      show-icon
      class="mt-20 mb-20"
    >
      <template #default>
        <div class="welcome-guide">
          <p class="guide-intro">
            本系统可以自动将视频翻译成中文，包括：<strong>语音识别</strong> →
            <strong>文本翻译</strong> → <strong>声音克隆</strong> →
            <strong>生成中文配音视频</strong>
          </p>
          <p class="guide-steps">
            请先完成以下<strong>必需配置</strong>（标记为
            <el-tag type="danger" size="small">必需</el-tag> 的三项）：
          </p>
          <ol class="config-list">
            <li>
              <strong>ASR服务</strong> - 自动语音识别，将视频中的外语语音转为文字
            </li>
            <li>
              <strong>翻译服务</strong> - 将识别出的外语文字翻译成中文
            </li>
            <li>
              <strong>声音克隆</strong> - 用中文重新配音，保留原视频说话人的声音特征
            </li>
          </ol>
          <p class="guide-tip">
            💡 提示：其他配置项均为可选，可以提升翻译质量，建议后续再配置
          </p>
        </div>
      </template>
    </el-alert>

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
      <el-alert type="success" :closable="false" class="mb-20">
        <template #title>
          <span>🎙️ 什么是 ASR（自动语音识别）？</span>
        </template>
        <div class="config-description">
          <p>
            <strong>ASR（Automatic Speech Recognition）</strong
            >是自动语音识别服务，它能将视频中的语音自动转换成文字。
          </p>
          <p class="description-detail">
            <strong>工作原理：</strong>系统会提取视频中的音频，然后使用 ASR
            服务将说话内容识别成文字（例如英语、日语等）。
          </p>
          <p class="description-example">
            <strong>举例：</strong>视频中有人说 "Hello, how are you?"，ASR
            会识别出这段文字。
          </p>
          <p class="description-note">
            💡
            <strong>推荐服务商：</strong>OpenAI Whisper（支持多种语言，识别准确率高）
          </p>
        </div>
      </el-alert>
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
          placeholder="请输入API密钥（至少10个字符）"
          show-password
        />
        <template #extra>
          <el-text type="info" size="small">
            💡 API密钥是服务商提供的访问凭证，请妥善保管。获取方式：登录服务商官网
            → 控制台 → API密钥管理
          </el-text>
        </template>
      </el-form-item>
      <el-form-item label="自定义端点">
        <el-input v-model="form.asr_endpoint" placeholder="可选，留空使用默认端点" />
        <template #extra>
          <el-text type="info" size="small">
            🔧 高级选项：仅当需要使用自定义服务器地址时填写，例如：https://api.example.com
          </el-text>
        </template>
      </el-form-item>

      <!-- 音频分离配置 -->
      <el-divider content-position="left">
        音频分离配置
        <el-tag type="info" size="small">可选</el-tag>
      </el-divider>
      <el-alert type="info" :closable="false" class="mb-20">
        <template #title>
          <span>🎵 什么是音频分离？</span>
        </template>
        <div class="config-description">
          <p>
            <strong>音频分离</strong>可以将视频中的人声和背景音乐分开，让语音识别更准确。
          </p>
          <p class="description-detail">
            <strong>适用场景：</strong>视频中有背景音乐或噪音时，开启此功能可以提高
            ASR 识别准确率。
          </p>
          <p class="description-note">
            ⚠️
            <strong>注意：</strong>此功能需要服务器具备
            GPU，处理速度会稍慢，如果视频音质清晰可以不开启。
          </p>
        </div>
      </el-alert>
      <el-form-item label="启用音频分离">
        <el-switch v-model="form.audio_separation_enabled" />
        <el-text type="warning" size="small" class="ml-10">需要GPU支持</el-text>
      </el-form-item>

      <!-- 文本润色配置 -->
      <el-divider content-position="left">
        文本润色配置
        <el-tag type="info" size="small">可选</el-tag>
      </el-divider>
      <el-alert type="info" :closable="false" class="mb-20">
        <template #title>
          <span>✨ 什么是文本润色？</span>
        </template>
        <div class="config-description">
          <p>
            <strong>文本润色</strong>会在翻译<strong>之前</strong>，使用
            AI 对识别出的原文进行优化，纠正错误、补全断句。
          </p>
          <p class="description-detail">
            <strong>工作原理：</strong>ASR 识别的文字可能有错误或不通顺，AI
            会将其修正为完整、准确的句子后再进行翻译。
          </p>
          <p class="description-example">
            <strong>举例：</strong>ASR 识别为 "he he said um hello" → 润色后为
            "He said hello"
          </p>
          <p class="description-note">
            💡
            <strong>推荐：</strong>对于口语化、有停顿的视频（如演讲、访谈）建议开启
          </p>
        </div>
      </el-alert>
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
      <el-alert type="success" :closable="false" class="mb-20">
        <template #title>
          <span>🌏 翻译服务是做什么的？</span>
        </template>
        <div class="config-description">
          <p>
            <strong>翻译服务</strong>负责将 ASR
            识别出的外语文字翻译成中文文字。
          </p>
          <p class="description-detail">
            <strong>工作流程：</strong>ASR 识别出英文 → 翻译服务转换成中文 →
            最后用中文重新配音
          </p>
          <p class="description-example">
            <strong>举例：</strong>"Hello, how are you?" → "你好，最近怎么样？"
          </p>
          <p class="description-note">
            💡
            <strong>推荐服务商：</strong>Google Gemini（翻译自然流畅，支持上下文理解）
          </p>
        </div>
      </el-alert>
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
      <el-alert type="info" :closable="false" class="mb-20">
        <template #title>
          <span>🎯 什么是译文优化？</span>
        </template>
        <div class="config-description">
          <p>
            <strong>译文优化</strong>会在翻译<strong>之后</strong>，使用 AI
            让中文译文更加自然、符合中文表达习惯。
          </p>
          <p class="description-detail">
            <strong>工作原理：</strong>有些直译出来的中文可能生硬、不通顺，AI
            会将其优化为更地道的中文表达。
          </p>
          <p class="description-example">
            <strong>举例：</strong>直译 "我很好，谢谢" → 优化后 "挺好的，谢谢关心"
          </p>
          <p class="description-note">
            💡
            <strong>建议：</strong>希望译文更加口语化、自然时开启（会增加一些处理时间）
          </p>
        </div>
      </el-alert>
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
      <el-alert type="success" :closable="false" class="mb-20">
        <template #title>
          <span>🎤 什么是声音克隆？</span>
        </template>
        <div class="config-description">
          <p>
            <strong>声音克隆（TTS - Text-to-Speech）</strong
            >能用中文朗读翻译后的文字，并尽量保持原视频说话人的声音特征。
          </p>
          <p class="description-detail">
            <strong>工作原理：</strong>系统会分析原视频中的声音特点（音色、语调等），然后用中文重新配音，让听起来像是原说话人在说中文。
          </p>
          <p class="description-example">
            <strong>举例：</strong>原视频是男性低沉的声音，生成的中文配音也会是男性低沉的声音。
          </p>
          <p class="description-note">
            💡
            <strong>推荐服务商：</strong>阿里云 CosyVoice（支持中文声音克隆，音质自然）
          </p>
        </div>
      </el-alert>
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
        <el-text type="info" size="small" class="ml-10">
          推荐开启：系统会自动从视频中提取最佳音频片段作为声音参考
        </el-text>
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

/* 欢迎引导样式 */
.welcome-guide {
  line-height: 1.8;
}

.guide-intro {
  font-size: 15px;
  margin-bottom: 12px;
  color: #606266;
}

.guide-steps {
  margin: 12px 0 8px 0;
  color: #303133;
}

.config-list {
  margin: 8px 0 12px 20px;
  padding-left: 0;
}

.config-list li {
  margin-bottom: 8px;
  color: #606266;
}

.guide-tip {
  margin-top: 12px;
  padding: 8px 12px;
  background-color: #f0f9ff;
  border-left: 3px solid #409eff;
  border-radius: 4px;
  color: #409eff;
  font-size: 14px;
}

/* 配置说明样式 */
.config-description {
  line-height: 1.8;
}

.config-description p {
  margin: 8px 0;
  color: #606266;
}

.config-description strong {
  color: #303133;
  font-weight: 600;
}

.description-detail {
  padding-left: 12px;
  border-left: 2px solid #e4e7ed;
}

.description-example {
  padding: 8px 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
}

.description-note {
  color: #409eff !important;
  font-size: 14px;
}

/* 工具类 */
.ml-10 {
  margin-left: 10px;
}

.mt-20 {
  margin-top: 20px;
}

.mb-20 {
  margin-bottom: 20px;
}

/* 表单额外提示 */
:deep(.el-form-item__extra) {
  line-height: 1.6;
  margin-top: 4px;
}

/* 分割线优化 */
:deep(.el-divider__text) {
  font-weight: 600;
  font-size: 16px;
  color: #303133;
}

/* Alert 优化 */
:deep(.el-alert) {
  border-radius: 8px;
}

:deep(.el-alert__title) {
  font-size: 15px;
  font-weight: 600;
}
</style>

