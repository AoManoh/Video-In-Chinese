/**
 * 常量定义
 * 
 * 与后端配置对齐
 * @reference notes/server/2nd/Gateway-design.md v5.9
 */

import type { TaskStatus } from '@/api/types'

// ==================== 文件上传相关 ====================

/**
 * 最大上传文件大小（MB）
 * 
 * @backend MAX_UPLOAD_SIZE_MB
 * @reference Gateway-design.md v5.9 第384行
 */
export const MAX_UPLOAD_SIZE_MB = 2048

/**
 * 最大上传文件大小（字节）
 */
export const MAX_FILE_SIZE_BYTES = MAX_UPLOAD_SIZE_MB * 1024 * 1024

/**
 * 支持的MIME Type
 * 
 * @backend SUPPORTED_MIME_TYPES
 * @reference Gateway-design.md v5.9 第385行
 */
export const ALLOWED_MIME_TYPES = [
  'video/mp4',
  'video/quicktime', // MOV
  'video/x-matroska' // MKV
] as const

/**
 * 支持的文件扩展名
 */
export const ALLOWED_EXTENSIONS = ['.mp4', '.mov', '.mkv'] as const

// ==================== 任务轮询相关 ====================

/**
 * 轮询初始间隔（毫秒）
 * 
 * @reference Client-Base-Design.md 第9.1节
 */
export const POLLING_INITIAL_INTERVAL = 3000

/**
 * 轮询最大间隔（毫秒）
 * 
 * @reference Client-Base-Design.md 第9.1节
 */
export const POLLING_MAX_INTERVAL = 10000

// ==================== 任务状态配置 ====================

/**
 * 任务状态配置
 * 
 * @reference TaskList-Page-Design.md 第4.1节
 */
export const TASK_STATUS_CONFIG = {
  PENDING: {
    text: '排队中',
    color: 'info' as const,
    icon: 'Clock',
    description: '任务已创建，正在排队等待处理'
  },
  PROCESSING: {
    text: '处理中',
    color: 'warning' as const,
    icon: 'Loading',
    description: '任务正在处理中，请耐心等待'
  },
  COMPLETED: {
    text: '已完成',
    color: 'success' as const,
    icon: 'CircleCheck',
    description: '任务已完成，可以下载结果'
  },
  FAILED: {
    text: '失败',
    color: 'danger' as const,
    icon: 'CircleClose',
    description: '任务处理失败'
  }
} as const satisfies Record<TaskStatus, { text: string; color: string; icon: string; description: string }>

// ==================== localStorage相关 ====================

/**
 * 最多保存的任务数量
 */
export const MAX_TASKS_STORAGE = 50

/**
 * 任务过期时间（7天，毫秒）
 */
export const TASK_EXPIRY_TIME = 7 * 24 * 60 * 60 * 1000

// ==================== 服务商选项 ====================

/**
 * ASR服务商选项
 */
export const ASR_PROVIDER_OPTIONS = [
  { label: 'OpenAI Whisper', value: 'openai-whisper' },
  { label: '阿里云语音识别', value: 'aliyun-asr' },
  { label: 'Azure Speech', value: 'azure-speech' },
  { label: 'Google Cloud Speech', value: 'google-speech' }
] as const

/**
 * 翻译服务商选项
 */
export const TRANSLATION_PROVIDER_OPTIONS = [
  { label: 'Google Gemini', value: 'google-gemini' },
  { label: 'DeepL', value: 'deepl' },
  { label: 'Azure Translator', value: 'azure-translator' },
  { label: '火山引擎翻译', value: 'volcengine-translate' }
] as const

/**
 * LLM服务商选项（文本润色、译文优化）
 */
export const LLM_PROVIDER_OPTIONS = [
  { label: 'OpenAI GPT-4o', value: 'openai-gpt4o' },
  { label: 'Claude 3.5', value: 'claude-3.5' },
  { label: 'Google Gemini', value: 'google-gemini' },
  { label: '火山引擎 Doubao', value: 'volcengine-doubao' }
] as const

/**
 * 声音克隆服务商选项
 */
export const VOICE_CLONING_PROVIDER_OPTIONS = [
  { label: '阿里云 CosyVoice', value: 'aliyun-cosyvoice' },
  { label: 'ElevenLabs', value: 'elevenlabs' }
] as const

/**
 * 翻译预设类型选项
 */
export const VIDEO_TYPE_OPTIONS = [
  {
    label: '专业科技',
    value: 'professional_tech',
    description: '保留专业术语，避免口语化'
  },
  {
    label: '口语自然',
    value: 'casual_natural',
    description: '口语化表达，更自然流畅'
  },
  {
    label: '教育严谨',
    value: 'educational_rigorous',
    description: '严谨准确，适合教学场景'
  },
  {
    label: '默认',
    value: 'default',
    description: '无倾向性，平衡准确性和流畅性'
  }
] as const

