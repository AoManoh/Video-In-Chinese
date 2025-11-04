<template>
  <div class="upload-page">
    <el-page-header @back="goBack" content="上传视频" />

    <el-card class="upload-card mt-20">
      <!-- 拖拽上传区域 -->
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
        <el-icon class="el-icon--upload"><upload-filled /></el-icon>
        <div class="el-upload__text">将文件拖到此处，或<em>点击选择文件</em></div>
        <template #tip>
          <div class="el-upload__tip">仅支持 MP4、MOV、MKV 格式，最大 2048MB</div>
        </template>
      </el-upload>

      <!-- 文件信息展示 -->
      <div v-if="selectedFile" class="file-info">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="文件名">{{ selectedFile.name }}</el-descriptions-item>
          <el-descriptions-item label="文件大小">{{
            formatFileSize(selectedFile.size)
          }}</el-descriptions-item>
          <el-descriptions-item label="文件格式">{{ selectedFile.type }}</el-descriptions-item>
          <el-descriptions-item label="上传状态">
            <el-tag :type="uploadStatusType">{{ uploadStatusText }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 上传进度条 -->
        <el-progress
          v-if="uploading"
          :percentage="uploadProgress"
          :status="uploadStatus"
          class="mt-20"
        >
          <template #default="{ percentage }">
            <span>{{ percentage }}%</span>
            <span v-if="uploadSpeed"> ({{ uploadSpeed }})</span>
          </template>
        </el-progress>

        <!-- 操作按钮 -->
        <div class="actions mt-20">
          <el-button
            v-if="!uploading && !uploadComplete"
            type="primary"
            size="large"
            @click="startUpload"
          >
            开始上传
          </el-button>
          <el-button v-if="uploading" type="danger" size="large" @click="cancelUpload">
            取消上传
          </el-button>
          <el-button v-if="uploadComplete" type="success" size="large" @click="goToTaskList">
            查看任务
          </el-button>
          <el-button v-if="!uploading" size="large" @click="resetUpload">重新选择</el-button>
        </div>
      </div>
    </el-card>

    <!-- 上传须知 -->
    <el-card class="notice-card mt-20">
      <template #header>
        <span>上传须知</span>
      </template>
      <ul>
        <li>支持的视频格式：MP4、MOV、MKV</li>
        <li>单个文件最大支持 2048MB</li>
        <li>上传成功后将自动创建翻译任务</li>
        <li>任务处理时间取决于视频长度（约 10分钟视频需要 30分钟）</li>
        <li>处理过程中请勿关闭浏览器</li>
      </ul>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
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
  if (bytes < 1024) {
    return `${bytes} B`
  } else if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`
  } else if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / 1024 / 1024).toFixed(2)} MB`
  } else {
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
  }
}

/**
 * 格式化上传速度
 */
const formatSpeed = (bytesPerSecond: number): string => {
  if (bytesPerSecond < 1024) {
    return `${bytesPerSecond.toFixed(0)} B/s`
  } else if (bytesPerSecond < 1024 * 1024) {
    return `${(bytesPerSecond / 1024).toFixed(2)} KB/s`
  } else {
    return `${(bytesPerSecond / 1024 / 1024).toFixed(2)} MB/s`
  }
}

/**
 * 计算上传速度
 */
const calculateUploadSpeed = (loaded: number) => {
  const now = Date.now()
  const timeDiff = (now - lastTime) / 1000 // 秒
  const loadedDiff = loaded - lastLoaded

  if (timeDiff >= 1) {
    const speed = loadedDiff / timeDiff // 字节/秒
    uploadSpeed.value = formatSpeed(speed)
    lastLoaded = loaded
    lastTime = now
  }
}

/**
 * 文件验证
 */
const validateFile = (file: File): boolean => {
  // 大小验证
  if (file.size > MAX_FILE_SIZE_BYTES) {
    ElMessage.error(`文件大小超过限制（最大 ${MAX_FILE_SIZE_MB}MB）`)
    return false
  }

  // 格式验证
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
      return // 用户取消
    }
  }

  // 重置状态并创建新的 AbortController
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

    // 上传成功
    taskId.value = response.task_id
    uploadComplete.value = true
    uploading.value = false
    uploadAbortController = null

    ElMessage.success('上传成功！正在跳转到任务列表...')

    // 3秒后自动跳转
    setTimeout(() => {
      goToTaskList()
    }, 3000)
  } catch (error) {
    // 检查是否是用户取消
    if (uploadAbortController?.signal.aborted) {
      // 用户主动取消，不显示错误消息
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

/**
 * 返回上一页
 */
const goBack = () => {
  router.back()
}
</script>

<style scoped>
.upload-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px;
}

.upload-card {
  .upload-dragger {
    width: 100%;
  }

  .file-info {
    margin-top: 20px;

    .actions {
      display: flex;
      gap: 12px;
    }
  }
}

.notice-card {
  ul {
    margin: 0;
    padding-left: 20px;

    li {
      margin-bottom: 8px;
      color: var(--el-text-color-secondary);
    }
  }
}

.mt-20 {
  margin-top: 20px;
}
</style>

