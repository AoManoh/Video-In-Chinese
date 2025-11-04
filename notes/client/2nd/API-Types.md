# API接口类型定义文档（第二层）

**文档版本**: 1.0
**创建日期**: 2025-11-03
**关联后端文档**: `notes/server/2nd/Gateway-design.md` v5.9

---

## 版本历史

- **v1.0 (2025-11-03)**:
  - 初始版本，定义所有API接口的TypeScript类型
  - 对齐后端Gateway-design.md v5.9第5章API定义
  - 提供完整的Mock数据示例
  - 提供Axios封装示例

---

## 1. 类型定义总览

### 1.1 接口对齐映射表

| 前端类型定义 | 后端接口 | 后端文档位置 | Mock数据 |
|-------------|---------|-------------|---------|
| `GetSettingsResponse` | GET /v1/settings | Gateway-design.md 第144-189行 | ✅ |
| `UpdateSettingsRequest` | POST /v1/settings | Gateway-design.md 第192-231行 | - |
| `UpdateSettingsResponse` | POST /v1/settings | Gateway-design.md 第234-237行 | ✅ |
| `UploadTaskResponse` | POST /v1/tasks/upload | Gateway-design.md 第242-244行 | ✅ |
| `GetTaskStatusRequest` | GET /v1/tasks/:taskId/status | Gateway-design.md 第247-249行 | - |
| `GetTaskStatusResponse` | GET /v1/tasks/:taskId/status | Gateway-design.md 第252-257行 | ✅ |
| `DownloadFileRequest` | GET /v1/tasks/download/:taskId/:fileName | Gateway-design.md 第260-263行 | - |

---

## 2. 配置管理相关类型

### 2.1 GetSettingsResponse

**后端接口**: `GET /v1/settings`
**后端文档**: Gateway-design.md 第144-189行

```typescript
/**
 * 获取应用配置的响应体
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第144-189行
 */
export interface GetSettingsResponse {
  /** 
   * 配置版本号，用于乐观锁
   * @required
   */
  version: number

  /** 
   * 系统是否已完成基本配置
   * 判断逻辑：至少配置了 ASR、Translation、VoiceCloning 三个必需服务的 API Key
   * 如果为 false，前端应显示"初始化向导"引导用户完成配置
   * @required
   */
  is_configured: boolean

  // --- 处理模式 ---

  /** 
   * 视频处理模式
   * V1.0仅支持 "standard"
   * @required
   */
  processing_mode: string

  // --- AI服务配置 ---

  /** 
   * ASR服务商标识
   * @required_if is_configured=true
   */
  asr_provider: string

  /** 
   * 脱敏后的ASR API Key
   * 格式：前缀-***-后6位（如 sk-proj-***-xyz789）
   * @required_if is_configured=true
   */
  asr_api_key: string

  /** 
   * 自定义的ASR服务端点URL
   * @optional
   */
  asr_endpoint?: string

  /** 
   * 是否启用音频分离（需要GPU）
   * @required
   * @default false
   */
  audio_separation_enabled: boolean

  /** 
   * 是否启用文本润色功能
   * @required
   */
  polishing_enabled: boolean

  /** 
   * 文本润色服务商标识
   * @optional
   */
  polishing_provider?: string

  /** 
   * 脱敏后的文本润色API Key
   * @optional
   */
  polishing_api_key?: string

  /** 
   * 用户自定义的润色Prompt
   * @optional
   */
  polishing_custom_prompt?: string

  /** 
   * 翻译预设类型
   * @values "professional_tech" | "casual_natural" | "educational_rigorous"
   * @optional
   */
  polishing_video_type?: string

  /** 
   * 翻译服务商标识
   * @required_if is_configured=true
   */
  translation_provider: string

  /** 
   * 脱敏后的翻译API Key
   * @required_if is_configured=true
   */
  translation_api_key: string

  /** 
   * 自定义的翻译服务端点URL
   * @optional
   */
  translation_endpoint?: string

  /** 
   * 翻译预设类型
   * @values "professional_tech" | "casual_natural" | "educational_rigorous"
   * @optional
   */
  translation_video_type?: string

  /** 
   * 是否启用译文优化功能
   * @required
   */
  optimization_enabled: boolean

  /** 
   * 译文优化服务商标识
   * @optional
   */
  optimization_provider?: string

  /** 
   * 脱敏后的译文优化API Key
   * @optional
   */
  optimization_api_key?: string

  /** 
   * 声音克隆服务商标识
   * @required_if is_configured=true
   */
  voice_cloning_provider: string

  /** 
   * 脱敏后的声音克隆API Key
   * @required_if is_configured=true
   */
  voice_cloning_api_key: string

  /** 
   * 自定义的声音克隆服务端点URL
   * @optional
   */
  voice_cloning_endpoint?: string

  /** 
   * 是否自动选择参考音频
   * @required
   * @default true
   */
  voice_cloning_auto_select_reference: boolean

  /** 
   * S2ST服务商标识（V2.0功能）
   * @optional
   */
  s2st_provider?: string

  /** 
   * 脱敏后的S2ST API Key（V2.0功能）
   * @optional
   */
  s2st_api_key?: string
}
```

### 2.2 UpdateSettingsRequest

**后端接口**: `POST /v1/settings`
**后端文档**: Gateway-design.md 第192-231行

```typescript
/**
 * 更新应用配置的请求体
 * 
 * @backend POST /v1/settings
 * @reference Gateway-design.md v5.9 第192-231行
 */
export interface UpdateSettingsRequest {
  /** 
   * 当前配置的版本号，用于乐观锁检查
   * @required
   */
  version: number

  // --- 处理模式 ---

  /** 
   * 更新处理模式
   * @optional
   */
  processing_mode?: string

  // --- AI服务配置 ---
  // 注意：除Version外，所有字段均为可选。只提交需要修改的字段。
  // API Key字段：如果提交的值包含"***", 则后端会忽略此字段，保持原值不变。

  asr_provider?: string
  /** 如果包含"***"则保持原值 */
  asr_api_key?: string
  asr_endpoint?: string

  /** 使用指针类型以区分 "未提交" 和 "提交了false值" */
  audio_separation_enabled?: boolean

  /** 使用指针类型以区分 "未提交" 和 "提交了false值" */
  polishing_enabled?: boolean
  polishing_provider?: string
  /** 如果包含"***"则保持原值 */
  polishing_api_key?: string
  polishing_custom_prompt?: string
  polishing_video_type?: string

  translation_provider?: string
  /** 如果包含"***"则保持原值 */
  translation_api_key?: string
  translation_endpoint?: string
  translation_video_type?: string

  optimization_enabled?: boolean
  optimization_provider?: string
  /** 如果包含"***"则保持原值 */
  optimization_api_key?: string

  voice_cloning_provider?: string
  /** 如果包含"***"则保持原值 */
  voice_cloning_api_key?: string
  voice_cloning_endpoint?: string
  voice_cloning_auto_select_reference?: boolean

  s2st_provider?: string
  /** 如果包含"***"则保持原值 */
  s2st_api_key?: string
}
```

### 2.3 UpdateSettingsResponse

**后端接口**: `POST /v1/settings`
**后端文档**: Gateway-design.md 第234-237行

```typescript
/**
 * 更新应用配置的响应体
 * 
 * @backend POST /v1/settings
 * @reference Gateway-design.md v5.9 第234-237行
 */
export interface UpdateSettingsResponse {
  /** 
   * 更新成功后，返回新的配置版本号
   * @required
   */
  version: number

  /** 
   * 成功提示信息
   * @example "配置已成功更新"
   * @required
   */
  message: string
}
```

---

## 3. 任务管理相关类型

### 3.1 UploadTaskResponse

**后端接口**: `POST /v1/tasks/upload`
**后端文档**: Gateway-design.md 第242-244行

```typescript
/**
 * 上传任务的响应体
 * 
 * @backend POST /v1/tasks/upload
 * @reference Gateway-design.md v5.9 第242-244行
 */
export interface UploadTaskResponse {
  /** 
   * 创建成功后返回的唯一任务ID
   * @required
   */
  task_id: string
}
```

### 3.2 GetTaskStatusRequest

**后端接口**: `GET /v1/tasks/:taskId/status`
**后端文档**: Gateway-design.md 第247-249行

```typescript
/**
 * 查询任务状态的请求参数（路径参数）
 * 
 * @backend GET /v1/tasks/:taskId/status
 * @reference Gateway-design.md v5.9 第247-249行
 */
export interface GetTaskStatusRequest {
  /** 
   * 需要查询的任务ID
   * @required
   */
  taskId: string
}
```

### 3.3 GetTaskStatusResponse

**后端接口**: `GET /v1/tasks/:taskId/status`
**后端文档**: Gateway-design.md 第252-257行

```typescript
/**
 * 任务状态枚举
 */
export type TaskStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED'

/**
 * 查询任务状态的响应体
 * 
 * @backend GET /v1/tasks/:taskId/status
 * @reference Gateway-design.md v5.9 第252-257行
 */
export interface GetTaskStatusResponse {
  /** 
   * 任务ID
   * @required
   */
  task_id: string

  /** 
   * 任务当前状态
   * @values "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED"
   * @required
   */
  status: TaskStatus

  /** 
   * 仅在任务状态为 "COMPLETED" 时出现
   * 格式：/v1/tasks/download/{taskId}/{fileName}
   * @optional
   */
  result_url?: string

  /** 
   * 仅在任务状态为 "FAILED" 时出现，包含失败原因
   * @optional
   */
  error_message?: string
}
```

### 3.4 DownloadFileRequest

**后端接口**: `GET /v1/tasks/download/:taskId/:fileName`
**后端文档**: Gateway-design.md 第260-263行

```typescript
/**
 * 下载文件的请求参数（路径参数）
 * 
 * @backend GET /v1/tasks/download/:taskId/:fileName
 * @reference Gateway-design.md v5.9 第260-263行
 */
export interface DownloadFileRequest {
  /** 
   * 文件所属的任务ID
   * @required
   */
  taskId: string

  /** 
   * 要下载的文件名
   * @example "result.mp4" 或 "original.mp4"
   * @required
   */
  fileName: string
}
```

---

## 4. 错误响应类型

### 4.1 错误响应结构

**后端文档**: Gateway-design.md 第8章"错误码清单"

```typescript
/**
 * API错误响应
 * 
 * @backend 所有接口的错误响应格式
 * @reference Gateway-design.md v5.9 第8章
 */
export interface APIError {
  /** 
   * 内部错误码
   * @example "INVALID_ARGUMENT", "REDIS_UNAVAILABLE"
   */
  code?: string

  /** 
   * 错误消息
   * @example "无效的任务ID"
   */
  message: string

  /** 
   * 当前版本号（仅409 Conflict错误）
   * @optional
   */
  current_version?: number
}
```

### 4.2 HTTP状态码映射

```typescript
/**
 * HTTP状态码枚举
 * 
 * @reference Gateway-design.md v5.9 第8章
 */
export const enum HTTPStatus {
  // 2xx 成功
  OK = 200,

  // 4xx 客户端错误
  BAD_REQUEST = 400,
  NOT_FOUND = 404,
  CONFLICT = 409,
  PAYLOAD_TOO_LARGE = 413,
  UNSUPPORTED_MEDIA_TYPE = 415,
  CLIENT_CLOSED_REQUEST = 499,

  // 5xx 服务端错误
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
  // 4xx 客户端错误
  INVALID_ARGUMENT = 'INVALID_ARGUMENT',
  NOT_FOUND = 'NOT_FOUND',
  CONFLICT = 'CONFLICT',
  PAYLOAD_TOO_LARGE = 'PAYLOAD_TOO_LARGE',
  UNSUPPORTED_MEDIA_TYPE = 'UNSUPPORTED_MEDIA_TYPE',
  MIME_TYPE_MISMATCH = 'MIME_TYPE_MISMATCH',
  CLIENT_CLOSED = 'CLIENT_CLOSED',

  // 5xx 服务端错误
  INTERNAL_ERROR = 'INTERNAL_ERROR',
  ENCRYPTION_FAILED = 'ENCRYPTION_FAILED',
  DECRYPTION_FAILED = 'DECRYPTION_FAILED',
  UNAVAILABLE = 'UNAVAILABLE',
  REDIS_UNAVAILABLE = 'REDIS_UNAVAILABLE',
  INSUFFICIENT_STORAGE = 'INSUFFICIENT_STORAGE'
}
```

---

## 5. API调用封装

### 5.1 配置管理API

```typescript
// api/settings-api.ts
import httpClient from '@/utils/http-client'
import type { GetSettingsResponse, UpdateSettingsRequest, UpdateSettingsResponse } from './types'

/**
 * 获取应用配置
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第276-277行
 * @returns GetSettingsResponse
 */
export const getSettings = async (): Promise<GetSettingsResponse> => {
  const response = await httpClient.get<GetSettingsResponse>('/v1/settings')
  return response.data
}

/**
 * 更新应用配置
 * 
 * @backend POST /v1/settings
 * @reference Gateway-design.md v5.9 第279-281行
 * @param request 更新请求（仅包含需要修改的字段）
 * @returns UpdateSettingsResponse
 * @throws {APIError} 409 Conflict - 配置版本冲突（需要刷新后重试）
 */
export const updateSettings = async (
  request: UpdateSettingsRequest
): Promise<UpdateSettingsResponse> => {
  const response = await httpClient.post<UpdateSettingsResponse>('/v1/settings', request)
  return response.data
}
```

### 5.2 任务管理API

```typescript
// api/task-api.ts
import httpClient from '@/utils/http-client'
import type { 
  UploadTaskResponse, 
  GetTaskStatusResponse 
} from './types'

/**
 * 上传视频文件并创建任务
 * 
 * @backend POST /v1/tasks/upload
 * @reference Gateway-design.md v5.9 第289-290行
 * @param file 视频文件
 * @param onProgress 上传进度回调
 * @returns UploadTaskResponse
 */
export const uploadTask = async (
  file: File,
  onProgress?: (percent: number) => void
): Promise<UploadTaskResponse> => {
  const formData = new FormData()
  formData.append('file', file)

  const response = await httpClient.post<UploadTaskResponse>('/v1/tasks/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    onUploadProgress: (progressEvent) => {
      if (onProgress && progressEvent.total) {
        const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        onProgress(percent)
      }
    }
  })

  return response.data
}

/**
 * 查询任务状态
 * 
 * @backend GET /v1/tasks/:taskId/status
 * @reference Gateway-design.md v5.9 第293-294行
 * @param taskId 任务ID
 * @returns GetTaskStatusResponse
 */
export const getTaskStatus = async (taskId: string): Promise<GetTaskStatusResponse> => {
  const response = await httpClient.get<GetTaskStatusResponse>(`/v1/tasks/${taskId}/status`)
  return response.data
}

/**
 * 下载任务结果文件
 * 
 * @backend GET /v1/tasks/download/:taskId/:fileName
 * @reference Gateway-design.md v5.9 第297-298行
 * @param taskId 任务ID
 * @param fileName 文件名
 * @returns Blob（文件内容）
 */
export const downloadFile = async (taskId: string, fileName: string): Promise<Blob> => {
  const response = await httpClient.get(`/v1/tasks/download/${taskId}/${fileName}`, {
    responseType: 'blob'
  })
  return response.data
}

/**
 * 辅助函数：触发浏览器下载
 * 
 * @param blob 文件内容
 * @param fileName 保存的文件名
 */
export const triggerDownload = (blob: Blob, fileName: string) => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  link.click()
  URL.revokeObjectURL(url)
}
```

---

## 6. Mock数据示例

### 6.1 mockSettings.json

```json
{
  "version": 1,
  "is_configured": true,
  "processing_mode": "standard",
  
  "asr_provider": "openai-whisper",
  "asr_api_key": "sk-proj-***-xyz789",
  "asr_endpoint": "",
  
  "audio_separation_enabled": false,
  
  "polishing_enabled": true,
  "polishing_provider": "openai-gpt4o",
  "polishing_api_key": "sk-proj-***-abc123",
  "polishing_custom_prompt": "",
  "polishing_video_type": "professional_tech",
  
  "translation_provider": "google-gemini",
  "translation_api_key": "AIza***-def456",
  "translation_endpoint": "",
  "translation_video_type": "professional_tech",
  
  "optimization_enabled": false,
  "optimization_provider": "",
  "optimization_api_key": "",
  
  "voice_cloning_provider": "aliyun-cosyvoice",
  "voice_cloning_api_key": "LTAI***-ghi789",
  "voice_cloning_endpoint": "",
  "voice_cloning_auto_select_reference": true,
  
  "s2st_provider": "",
  "s2st_api_key": ""
}
```

### 6.2 mockUploadResponse.json

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 6.3 mockTaskStatus.json

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PROCESSING",
  "result_url": null,
  "error_message": null
}
```

### 6.4 mockTaskStatusCompleted.json

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "result_url": "/v1/tasks/download/550e8400-e29b-41d4-a716-446655440000/result.mp4",
  "error_message": null
}
```

### 6.5 mockTaskStatusFailed.json

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "FAILED",
  "result_url": null,
  "error_message": "音频提取失败：文件格式不支持"
}
```

---

## 7. Mock实现示例

### 7.1 settings.ts

```typescript
// mock/settings.ts
import type MockAdapter from 'axios-mock-adapter'
import type { GetSettingsResponse, UpdateSettingsResponse } from '@/api/types'
import mockSettingsData from './data/settings.json'

export const setupSettingsMock = (mock: MockAdapter) => {
  // GET /v1/settings
  mock.onGet('/v1/settings').reply(200, mockSettingsData)

  // POST /v1/settings
  mock.onPost('/v1/settings').reply((config) => {
    const request = JSON.parse(config.data)
    
    // 模拟乐观锁检查
    if (request.version !== mockSettingsData.version) {
      return [
        409,
        {
          code: 'CONFLICT',
          message: '配置已被其他用户修改，请刷新后重试',
          current_version: mockSettingsData.version
        }
      ]
    }

    // 模拟版本号递增
    const newVersion = mockSettingsData.version + 1
    const response: UpdateSettingsResponse = {
      version: newVersion,
      message: '配置已成功更新'
    }

    // 更新Mock数据（仅用于测试）
    Object.assign(mockSettingsData, request, { version: newVersion })

    return [200, response]
  })
}
```

### 7.2 task.ts

```typescript
// mock/task.ts
import type MockAdapter from 'axios-mock-adapter'
import type { UploadTaskResponse, GetTaskStatusResponse } from '@/api/types'

// 模拟任务状态存储
const taskStatusMap = new Map<string, GetTaskStatusResponse>()

export const setupTaskMock = (mock: MockAdapter) => {
  // POST /v1/tasks/upload
  mock.onPost('/v1/tasks/upload').reply(() => {
    const taskId = crypto.randomUUID()
    const response: UploadTaskResponse = {
      task_id: taskId
    }

    // 初始化任务状态
    taskStatusMap.set(taskId, {
      task_id: taskId,
      status: 'PENDING'
    })

    // 模拟状态变化（3秒后变为PROCESSING，10秒后变为COMPLETED）
    setTimeout(() => {
      taskStatusMap.set(taskId, {
        task_id: taskId,
        status: 'PROCESSING'
      })
    }, 3000)

    setTimeout(() => {
      taskStatusMap.set(taskId, {
        task_id: taskId,
        status: 'COMPLETED',
        result_url: `/v1/tasks/download/${taskId}/result.mp4`
      })
    }, 10000)

    return [200, response]
  })

  // GET /v1/tasks/:taskId/status
  mock.onGet(/\/v1\/tasks\/(.+)\/status/).reply((config) => {
    const taskId = config.url?.match(/\/v1\/tasks\/(.+)\/status/)?.[1]
    
    if (!taskId || !taskStatusMap.has(taskId)) {
      return [
        404,
        {
          code: 'NOT_FOUND',
          message: '任务不存在'
        }
      ]
    }

    const status = taskStatusMap.get(taskId)!
    return [200, status]
  })

  // GET /v1/tasks/download/:taskId/:fileName
  mock.onGet(/\/v1\/tasks\/download\/(.+)\/(.+)/).reply(() => {
    // 返回一个空的Blob（实际应该返回视频文件）
    const blob = new Blob(['mock video file'], { type: 'video/mp4' })
    return [200, blob]
  })
}
```

---

## 8. 使用示例

### 8.1 获取配置

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getSettings } from '@/api/settings-api'
import type { GetSettingsResponse } from '@/api/types'
import { ElMessage } from 'element-plus'

const settings = ref<GetSettingsResponse | null>(null)
const loading = ref(false)

const loadSettings = async () => {
  loading.value = true
  try {
    settings.value = await getSettings()
  } catch (error) {
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
```

### 8.2 更新配置

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { updateSettings } from '@/api/settings-api'
import type { UpdateSettingsRequest } from '@/api/types'
import { ElMessage } from 'element-plus'

const currentVersion = ref(1)

const saveSettings = async () => {
  const request: UpdateSettingsRequest = {
    version: currentVersion.value,
    asr_provider: 'openai-whisper',
    asr_api_key: 'sk-proj-abc123...',
    // 其他字段...
  }

  try {
    const response = await updateSettings(request)
    currentVersion.value = response.version
    ElMessage.success(response.message)
  } catch (error) {
    // 错误已由http-client拦截器处理
  }
}
</script>
```

### 8.3 上传任务

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { uploadTask } from '@/api/task-api'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const uploadProgress = ref(0)
const router = useRouter()

const handleUpload = async (file: File) => {
  try {
    const response = await uploadTask(file, (percent) => {
      uploadProgress.value = percent
    })
    
    ElMessage.success('上传成功')
    router.push({ name: 'TaskList', query: { taskId: response.task_id } })
  } catch (error) {
    // 错误已由http-client拦截器处理
  }
}
</script>
```

### 8.4 查询任务状态

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getTaskStatus } from '@/api/task-api'
import type { TaskStatus } from '@/api/types'

const props = defineProps<{ taskId: string }>()
const status = ref<TaskStatus>('PENDING')
const pollerTimer = ref<number | null>(null)

const startPolling = () => {
  pollerTimer.value = window.setInterval(async () => {
    try {
      const response = await getTaskStatus(props.taskId)
      status.value = response.status

      // 任务完成或失败，停止轮询
      if (response.status === 'COMPLETED' || response.status === 'FAILED') {
        stopPolling()
      }
    } catch (error) {
      stopPolling()
    }
  }, 3000)
}

const stopPolling = () => {
  if (pollerTimer.value) {
    clearInterval(pollerTimer.value)
    pollerTimer.value = null
  }
}

onMounted(() => {
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})
</script>
```

---

## 9. 后端实现后的接口对齐检查清单

### 9.1 类型定义检查

- [ ] GetSettingsResponse字段数量一致（20+个字段）
- [ ] GetSettingsResponse字段名称一致（使用snake_case）
- [ ] GetSettingsResponse字段类型一致（string、number、boolean）
- [ ] UpdateSettingsRequest字段完全可选
- [ ] UploadTaskResponse返回task_id
- [ ] GetTaskStatusResponse状态枚举值一致（PENDING、PROCESSING、COMPLETED、FAILED）

### 9.2 HTTP状态码检查

- [ ] 200 OK - 成功响应
- [ ] 400 Bad Request - 参数错误
- [ ] 404 Not Found - 资源不存在
- [ ] 409 Conflict - 配置版本冲突
- [ ] 413 Payload Too Large - 文件过大
- [ ] 415 Unsupported Media Type - 文件格式不支持
- [ ] 500 Internal Server Error - 服务器内部错误
- [ ] 503 Service Unavailable - 服务不可用
- [ ] 507 Insufficient Storage - 磁盘空间不足

### 9.3 错误响应格式检查

- [ ] 错误响应包含message字段
- [ ] 错误响应可能包含code字段
- [ ] 409错误包含current_version字段

---

## 10. 文档变更历史

| 版本 | 日期       | 变更内容                                |
| ---- | ---------- | --------------------------------------- |
| 1.0  | 2025-11-03 | 初始版本，定义所有API接口的TypeScript类型 |

---

**文档结束**
