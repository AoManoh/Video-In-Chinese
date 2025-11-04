/**
 * API接口类型定义
 * 
 * 完全对齐后端Gateway API接口
 * @reference notes/server/2nd/Gateway-design.md v5.9
 */

// ==================== 配置管理相关类型 ====================

/**
 * 获取应用配置的响应体
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第144-189行
 */
export interface GetSettingsResponse {
  /** 配置版本号，用于乐观锁 */
  version: number

  /** 系统是否已完成基本配置（至少配置了ASR、Translation、VoiceCloning） */
  is_configured: boolean

  /** 视频处理模式（V1.0仅支持"standard"） */
  processing_mode: string

  // ASR服务配置
  asr_provider: string
  asr_api_key: string
  asr_endpoint?: string

  // 音频分离配置
  audio_separation_enabled: boolean

  // 文本润色配置
  polishing_enabled: boolean
  polishing_provider?: string
  polishing_api_key?: string
  polishing_endpoint?: string
  polishing_custom_prompt?: string
  polishing_video_type?: string

  // 翻译服务配置
  translation_provider: string
  translation_api_key: string
  translation_endpoint?: string
  translation_video_type?: string

  // 译文优化配置
  optimization_enabled: boolean
  optimization_provider?: string
  optimization_api_key?: string
  optimization_endpoint?: string

  // 声音克隆配置
  voice_cloning_provider: string
  voice_cloning_api_key: string
  voice_cloning_endpoint?: string
  voice_cloning_auto_select_reference: boolean

  // S2ST配置（V2.0）
  s2st_provider?: string
  s2st_api_key?: string
}

/**
 * 更新应用配置的请求体
 * 
 * @backend POST /v1/settings
 * @reference Gateway-design.md v5.9 第192-231行
 */
export interface UpdateSettingsRequest {
  /** 当前配置版本号（乐观锁） */
  version: number

  // 所有字段均为可选，只提交需要修改的字段
  processing_mode?: string

  asr_provider?: string
  asr_api_key?: string
  asr_endpoint?: string

  audio_separation_enabled?: boolean

  polishing_enabled?: boolean
  polishing_provider?: string
  polishing_api_key?: string
  polishing_endpoint?: string
  polishing_custom_prompt?: string
  polishing_video_type?: string

  translation_provider?: string
  translation_api_key?: string
  translation_endpoint?: string
  translation_video_type?: string

  optimization_enabled?: boolean
  optimization_provider?: string
  optimization_api_key?: string
  optimization_endpoint?: string

  voice_cloning_provider?: string
  voice_cloning_api_key?: string
  voice_cloning_endpoint?: string
  voice_cloning_auto_select_reference?: boolean

  s2st_provider?: string
  s2st_api_key?: string
}

/**
 * 更新应用配置的响应体
 * 
 * @backend POST /v1/settings
 * @reference Gateway-design.md v5.9 第234-237行
 */
export interface UpdateSettingsResponse {
  /** 更新后的新版本号 */
  version: number

  /** 成功提示消息 */
  message: string
}

// ==================== 任务管理相关类型 ====================

/**
 * 上传任务的响应体
 * 
 * @backend POST /v1/tasks/upload
 * @reference Gateway-design.md v5.9 第242-244行
 */
export interface UploadTaskResponse {
  /** 创建成功后返回的唯一任务ID */
  task_id: string
}

/**
 * 任务状态枚举
 * 
 * @reference Gateway-design.md v5.9 第254行
 */
export type TaskStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED'

/**
 * 查询任务状态的请求参数
 * 
 * @backend GET /v1/tasks/:taskId/status
 * @reference Gateway-design.md v5.9 第247-249行
 */
export interface GetTaskStatusRequest {
  /** 任务ID（路径参数） */
  taskId: string
}

/**
 * 查询任务状态的响应体
 * 
 * @backend GET /v1/tasks/:taskId/status
 * @reference Gateway-design.md v5.9 第252-257行
 */
export interface GetTaskStatusResponse {
  /** 任务ID */
  task_id: string

  /** 任务状态 */
  status: TaskStatus

  /** 下载URL（仅COMPLETED状态） */
  result_url?: string

  /** 错误消息（仅FAILED状态） */
  error_message?: string
}

/**
 * 下载文件的请求参数
 * 
 * @backend GET /v1/tasks/download/:taskId/:fileName
 * @reference Gateway-design.md v5.9 第260-263行
 */
export interface DownloadFileRequest {
  /** 任务ID（路径参数） */
  taskId: string

  /** 文件名（路径参数） */
  fileName: string
}

// ==================== 错误响应类型 ====================

/**
 * API错误响应
 * 
 * @reference Gateway-design.md v5.9 第8章
 */
export interface APIError {
  /** 内部错误码 */
  code?: string

  /** 错误消息 */
  message: string

  /** 当前版本号（仅409错误） */
  current_version?: number
}

// ==================== HTTP状态码枚举 ====================

/**
 * HTTP状态码枚举
 * 
 * @reference Gateway-design.md v5.9 第8章
 */
export const enum HTTPStatus {
  OK = 200,
  BAD_REQUEST = 400,
  NOT_FOUND = 404,
  CONFLICT = 409,
  PAYLOAD_TOO_LARGE = 413,
  UNSUPPORTED_MEDIA_TYPE = 415,
  CLIENT_CLOSED_REQUEST = 499,
  INTERNAL_SERVER_ERROR = 500,
  SERVICE_UNAVAILABLE = 503,
  INSUFFICIENT_STORAGE = 507
}

/**
 * 错误码枚举
 * 
 * @reference Gateway-design.md v5.9 第8章
 */
export const enum ErrorCode {
  INVALID_ARGUMENT = 'INVALID_ARGUMENT',
  NOT_FOUND = 'NOT_FOUND',
  CONFLICT = 'CONFLICT',
  PAYLOAD_TOO_LARGE = 'PAYLOAD_TOO_LARGE',
  UNSUPPORTED_MEDIA_TYPE = 'UNSUPPORTED_MEDIA_TYPE',
  MIME_TYPE_MISMATCH = 'MIME_TYPE_MISMATCH',
  CLIENT_CLOSED = 'CLIENT_CLOSED',
  INTERNAL_ERROR = 'INTERNAL_ERROR',
  ENCRYPTION_FAILED = 'ENCRYPTION_FAILED',
  DECRYPTION_FAILED = 'DECRYPTION_FAILED',
  UNAVAILABLE = 'UNAVAILABLE',
  REDIS_UNAVAILABLE = 'REDIS_UNAVAILABLE',
  INSUFFICIENT_STORAGE = 'INSUFFICIENT_STORAGE'
}

// ==================== 扩展类型（前端使用） ====================

/**
 * 任务数据结构（扩展后端响应）
 */
export interface Task {
  // 后端字段
  task_id: string
  status: TaskStatus
  result_url?: string
  error_message?: string

  // 前端扩展字段
  created_at: number // Unix时间戳
  updated_at: number // Unix时间戳
}

