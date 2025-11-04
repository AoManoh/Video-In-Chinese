<template>
  <div class="upload-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h2>上传视频</h2>
      <p class="header-subtitle">上传视频文件，自动翻译成中文</p>
    </div>

    <el-row :gutter="24">
      <!-- 上传区域 -->
      <el-col :span="16">
        <el-card shadow="never" class="upload-card">
          <!-- 拖拽上传 -->
          <el-upload
            v-if="!selectedFile"
            ref="uploadRef"
            class="upload-dragger"
            drag
            :auto-upload="false"
            :show-file-list="false"
            :on-change="handleFileChange"
            :before-upload="beforeUpload"
            accept="video/mp4,video/quicktime,video/x-matroska"
          >
            <el-icon class="upload-icon"><UploadFilled /></el-icon>
            <div class="upload-text">
              <p class="main-text">拖拽视频文件到此处</p>
              <p class="sub-text">或点击选择文件</p>
            </div>
            <template #tip>
              <div class="upload-tip">
                支持 MP4、MOV、MKV 格式，最大 2GB
              </div>
            </template>
          </el-upload>

          <!-- 文件信息 -->
          <div v-if="selectedFile" class="file-preview">
            <div class="file-icon-wrapper">
              <el-icon class="file-icon"><VideoCamera /></el-icon>
            </div>
            
            <div class="file-details">
              <h3 class="file-name">{{ selectedFile.name }}</h3>
              <div class="file-meta">
                <el-tag size="small" type="info">{{ formatFileSize(selectedFile.size) }}</el-tag>
                <el-tag size="small" :type="uploadStatusType">{{ uploadStatusText }}</el-tag>
              </div>
            </div>

            <!-- 上传进度 -->
            <el-progress
              v-if="uploading"
              :percentage="uploadProgress"
              :status="uploadStatus"
              class="upload-progress"
            >
              <template #default="{ percentage }">
                <span class="progress-text">{{ percentage }}%</span>
                <span v-if="uploadSpeed" class="progress-speed">{{ uploadSpeed }}</span>
              </template>
            </el-progress>

            <!-- 操作按钮 -->
            <div class="file-actions">
              <el-button
                v-if="!uploading && !uploadComplete"
                type="primary"
                size="large"
                @click="startUpload"
              >
                <el-icon><Upload /></el-icon>
                开始上传
              </el-button>
              <el-button
                v-if="uploading"
                type="danger"
                size="large"
                @click="cancelUpload"
              >
                <el-icon><Close /></el-icon>
                取消上传
              </el-button>
              <el-button
                v-if="uploadComplete"
                type="success"
                size="large"
                @click="goToTaskList"
              >
                <el-icon><Check /></el-icon>
                查看任务
              </el-button>
              <el-button
                v-if="!uploading"
                size="large"
                @click="resetUpload"
              >
                <el-icon><RefreshLeft /></el-icon>
                重新选择
              </el-button>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 说明侧边栏 -->
      <el-col :span="8">
        <el-card shadow="never" class="info-card">
          <template #header>
            <span style="font-weight: 600">上传说明</span>
          </template>
          <div class="info-list">
            <div class="info-item">
              <el-icon class="item-icon" color="#3b82f6"><Document /></el-icon>
              <div class="item-content">
                <h4>文件格式</h4>
                <p>支持 MP4、MOV、MKV</p>
              </div>
            </div>
            <div class="info-item">
              <el-icon class="item-icon" color="#10b981"><Files /></el-icon>
              <div class="item-content">
                <h4>文件大小</h4>
                <p>最大支持 2048MB</p>
              </div>
            </div>
            <div class="info-item">
              <el-icon class="item-icon" color="#f59e0b"><Clock /></el-icon>
              <div class="item-content">
                <h4>处理时间</h4>
                <p>约为视频时长的 3 倍</p>
              </div>
            </div>
            <div class="info-item">
              <el-icon class="item-icon" color="#ef4444"><Warning /></el-icon>
              <div class="item-content">
                <h4>注意事项</h4>
                <p>处理中请勿关闭浏览器</p>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  UploadFilled,
  VideoCamera,
  Upload,
  Close,
  Check,
  RefreshLeft,
  Document,
  Files,
  Clock,
  Warning
} from '@element-plus/icons-vue'
import { uploadTask } from '@/api/task-api'
import type { UploadUserFile } from 'element-plus'

const router = useRouter()

// AbortController 用于取消上传
let uploadAbortController: AbortController | null = null

// 文件验证规则
const MAX_FILE_SIZE_MB = 2048
const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024
const ALLOWED_MIME_TYPES = ['video/mp4', 'video/quicktime', 'video/x-matroska']

// 响应式数据
const selectedFile = ref<File | null>(null)
const uploading = ref(false)
const uploadComplete = ref(false)
const uploadProgress = ref(0)
const uploadSpeed = ref('')
const taskId = ref('')

// 上传速度计算
let lastLoaded = 0
let lastTime = Date.now()

// 计算属性
const uploadStatus = computed(() => {
  if (uploadComplete.value) return 'success'
  return undefined
})

const uploadStatusType = computed(() => {
  if (uploadComplete.value) return 'success'
  if (uploading.value) return 'warning'
  return 'info'
})

const uploadStatusText = computed(() => {
  if (uploadComplete.value) return '上传成功'
  if (uploading.value) return '上传中'
  return '等待上传'
})

/**
 * 格式化文件大小
 */
const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

/**
 * 格式化上传速度
 */
const formatSpeed = (bytesPerSecond: number): string => {
  if (bytesPerSecond < 1024) return `${bytesPerSecond.toFixed(0)} B/s`
  if (bytesPerSecond < 1024 * 1024) return `${(bytesPerSecond / 1024).toFixed(2)} KB/s`
  return `${(bytesPerSecond / 1024 / 1024).toFixed(2)} MB/s`
}

/**
 * 计算上传速度
 */
const calculateUploadSpeed = (loaded: number) => {
  const now = Date.now()
  const timeDiff = (now - lastTime) / 1000
  const loadedDiff = loaded - lastLoaded

  if (timeDiff >= 1) {
    const speed = loadedDiff / timeDiff
    uploadSpeed.value = formatSpeed(speed)
    lastLoaded = loaded
    lastTime = now
  }
}

/**
 * 文件验证
 */
const validateFile = (file: File): boolean => {
  if (file.size > MAX_FILE_SIZE_BYTES) {
    ElMessage.error(`文件大小超过限制（最大 ${MAX_FILE_SIZE_MB}MB）`)
    return false
  }

  if (!ALLOWED_MIME_TYPES.includes(file.type)) {
    ElMessage.error('不支持的文件格式，仅支持 MP4、MOV、MKV 格式')
    return false
  }

  return true
}

/**
 * 文件选择回调
 */
const handleFileChange = (uploadFile: UploadUserFile) => {
  const file = uploadFile.raw
  if (!file) return

  if (validateFile(file)) {
    selectedFile.value = file
  }
}

/**
 * 上传前钩子
 */
const beforeUpload = (file: File) => {
  return validateFile(file)
}

/**
 * 开始上传
 */
const startUpload = async () => {
  if (!selectedFile.value) return

  // 大文件上传前确认
  if (selectedFile.value.size > 500 * 1024 * 1024) {
    try {
      await ElMessageBox.confirm(
        `文件大小为 ${formatFileSize(selectedFile.value.size)}，上传可能需要较长时间。是否继续？`,
        '确认上传',
        {
          confirmButtonText: '继续',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return
    }
  }

  uploading.value = true
  uploadProgress.value = 0
  uploadSpeed.value = ''
  lastLoaded = 0
  lastTime = Date.now()
  uploadAbortController = new AbortController()

  try {
    const response = await uploadTask(
      selectedFile.value,
      percent => {
        uploadProgress.value = percent
        calculateUploadSpeed((selectedFile.value!.size * percent) / 100)
      },
      uploadAbortController.signal
    )

    taskId.value = response.task_id
    uploadComplete.value = true
    uploading.value = false
    uploadAbortController = null

    ElMessage.success('上传成功！正在跳转到任务列表...')

    setTimeout(() => {
      goToTaskList()
    }, 2000)
  } catch (error) {
    if (uploadAbortController?.signal.aborted) {
      return
    }
    
    uploading.value = false
    uploadProgress.value = 0
    uploadSpeed.value = ''
    uploadAbortController = null
  }
}

/**
 * 取消上传
 */
const cancelUpload = () => {
  if (uploadAbortController) {
    uploadAbortController.abort()
    uploadAbortController = null
  }
  
  uploading.value = false
  uploadProgress.value = 0
  uploadSpeed.value = ''
  ElMessage.info('上传已取消')
}

/**
 * 重置上传
 */
const resetUpload = () => {
  selectedFile.value = null
  uploading.value = false
  uploadComplete.value = false
  uploadProgress.value = 0
  uploadSpeed.value = ''
  taskId.value = ''
}

/**
 * 跳转到任务列表
 */
const goToTaskList = () => {
  router.push({ name: 'TaskList', query: { taskId: taskId.value } })
}
</script>

<style scoped>
.upload-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 24px;
}

.page-header {
  margin-bottom: 32px;

  h2 {
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

.upload-card,
.info-card {
  border-radius: var(--app-border-radius);
  border: 1px solid #e5e7eb;
}

/* 上传区域样式 */
.upload-dragger {
  :deep(.el-upload-dragger) {
    border: 2px dashed #d1d5db;
    border-radius: 12px;
    background: #f9fafb;
    transition: all 0.3s;
    padding: 60px 40px;

    &:hover {
      border-color: var(--el-color-primary);
      background: #eff6ff;
    }
  }

  .upload-icon {
    font-size: 64px;
    color: #9ca3af;
    margin-bottom: 16px;
  }

  .upload-text {
    .main-text {
      font-size: 18px;
      font-weight: 600;
      color: #374151;
      margin: 0 0 8px 0;
    }

    .sub-text {
      font-size: 14px;
      color: #6b7280;
      margin: 0;
    }
  }

  .upload-tip {
    margin-top: 16px;
    font-size: 13px;
    color: #9ca3af;
  }
}

/* 文件预览样式 */
.file-preview {
  text-align: center;
  padding: 40px 24px;

  .file-icon-wrapper {
    margin-bottom: 24px;

    .file-icon {
      font-size: 80px;
      color: var(--el-color-primary);
    }
  }

  .file-details {
    margin-bottom: 24px;

    .file-name {
      font-size: 18px;
      font-weight: 600;
      color: #1f2937;
      margin: 0 0 12px 0;
      word-break: break-all;
    }

    .file-meta {
      display: flex;
      justify-content: center;
      gap: 8px;
    }
  }

  .upload-progress {
    margin-bottom: 24px;

    .progress-text {
      font-weight: 600;
    }

    .progress-speed {
      margin-left: 8px;
      color: #6b7280;
      font-size: 13px;
    }
  }

  .file-actions {
    display: flex;
    justify-content: center;
    gap: 12px;
  }
}

/* 信息卡片样式 */
.info-card {
  :deep(.el-card__header) {
    padding: 16px 20px;
    border-bottom: 1px solid #e5e7eb;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }
}

.info-list {
  .info-item {
    display: flex;
    gap: 12px;
    padding: 16px 0;
    border-bottom: 1px dashed #e5e7eb;

    &:last-child {
      border-bottom: none;
      padding-bottom: 0;
    }

    &:first-child {
      padding-top: 0;
    }

    .item-icon {
      font-size: 24px;
      flex-shrink: 0;
    }

    .item-content {
      flex: 1;

      h4 {
        font-size: 14px;
        font-weight: 600;
        color: #374151;
        margin: 0 0 4px 0;
      }

      p {
        font-size: 13px;
        color: #6b7280;
        margin: 0;
      }
    }
  }
}
</style>
