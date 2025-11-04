<template>
  <div class="task-list-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-title">
        <h2>任务列表</h2>
        <p class="header-subtitle">查看视频翻译任务的处理进度</p>
      </div>
      <div class="header-actions">
        <el-button @click="refreshAllTasks" :loading="refreshing">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="goToUpload">
          <el-icon><Plus /></el-icon>
          上传新视频
        </el-button>
      </div>
    </div>

    <!-- 空状态 -->
    <el-empty
      v-if="taskList.length === 0"
      description="暂无任务，立即上传视频开始翻译"
      :image-size="160"
      class="empty-state"
    >
      <el-button type="primary" size="large" @click="goToUpload">
        <el-icon><Upload /></el-icon>
        上传视频
      </el-button>
    </el-empty>

    <!-- 任务列表 -->
    <div v-else class="task-grid">
      <TaskCard
        v-for="task in taskList"
        :key="task.task_id"
        :task="task"
        :highlight="task.task_id === highlightTaskId"
        @download="handleDownload"
        @retry="handleRetry"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElNotification } from 'element-plus'
import { Refresh, Plus, Upload } from '@element-plus/icons-vue'
import TaskCard from '@/components/TaskCard.vue'
import { getTaskStatus, downloadFile, triggerDownload } from '@/api/task-api'
import type { Task } from '@/api/types'
import { getTaskList, setTaskList, addTask as addTaskToStorage, updateTask } from '@/utils/storage'
import { taskPoller } from '@/utils/task-poller'
import { getConfigErrorSuggestion } from '@/utils/validation'
import type { GetTaskStatusResponse } from '@/api/types'

const router = useRouter()
const route = useRoute()

// 响应式数据
const taskList = ref<Task[]>([])
const highlightTaskId = ref('')
const refreshing = ref(false)

/**
 * 加载任务列表
 */
const loadTaskList = () => {
  taskList.value = getTaskList()
}

/**
 * 初始化轮询
 */
const initPolling = () => {
  taskList.value.forEach(task => {
    if (task.status === 'PROCESSING' || task.status === 'PENDING') {
      taskPoller.start(task.task_id, response => {
        updateTaskStatus(task.task_id, response)
      })
    }
  })
}

/**
 * 更新任务状态
 */
const updateTaskStatus = (taskId: string, response: GetTaskStatusResponse) => {
  const task = taskList.value.find(t => t.task_id === taskId)
  if (!task) return

  const oldStatus = task.status
  task.status = response.status
  task.result_url = response.result_url
  task.error_message = response.error_message
  task.updated_at = Date.now()

  updateTask(taskId, {
    status: response.status,
    result_url: response.result_url,
    error_message: response.error_message,
    updated_at: task.updated_at
  })

  // 状态变化通知
  if (oldStatus !== response.status) {
    if (response.status === 'COMPLETED') {
      ElNotification.success({
        title: '✅ 任务完成',
        message: `任务 ${taskId.slice(0, 8)} 已完成，可以下载结果`,
        duration: 0
      })
    } else if (response.status === 'FAILED') {
      // 检查是否为配置相关错误
      const errorMessage = response.error_message || '未知错误'
      const suggestion = getConfigErrorSuggestion(errorMessage)
      
      // 构建错误提示
      let notificationMessage = `任务 ${taskId.slice(0, 8)} 处理失败：${errorMessage}`
      if (suggestion) {
        notificationMessage += `\n\n${suggestion}`
      }
      
      ElNotification.error({
        title: '❌ 任务失败',
        message: notificationMessage,
        duration: 0,
        dangerouslyUseHTMLString: false
      })
      
      // 如果是配置错误，额外提示用户前往配置页面
      if (
        errorMessage.includes('API 密钥') ||
        errorMessage.includes('配置') ||
        errorMessage.includes('401') ||
        errorMessage.includes('403')
      ) {
        setTimeout(() => {
          ElMessage.warning({
            message: '建议前往"服务配置"页面检查 API 密钥设置',
            duration: 5000,
            showClose: true
          })
        }, 1000)
      }
    }
  }
}

/**
 * 刷新所有任务
 */
const refreshAllTasks = async () => {
  refreshing.value = true

  try {
    await Promise.all(
      taskList.value.map(task =>
        getTaskStatus(task.task_id)
          .then(response => {
            updateTaskStatus(task.task_id, response)
          })
          .catch(error => {
            console.error(`刷新任务${task.task_id}失败:`, error)
          })
      )
    )

    ElMessage.success('刷新成功')
  } catch (error) {
    ElMessage.error('刷新失败')
  } finally {
    refreshing.value = false
  }
}

/**
 * 下载任务结果
 */
const handleDownload = async (task: Task) => {
  if (!task.result_url) {
    ElMessage.error('下载链接不存在')
    return
  }

  const urlParts = task.result_url.split('/')
  const taskIdFromUrl = urlParts[urlParts.length - 2]
  const fileName = urlParts[urlParts.length - 1]

  try {
    ElMessage.info('正在下载，请稍候...')
    const blob = await downloadFile(taskIdFromUrl, fileName)
    triggerDownload(blob, fileName)
    ElMessage.success('下载成功')
  } catch (error) {
    console.error('下载失败:', error)
  }
}

/**
 * 重试任务
 */
const handleRetry = () => {
  router.push({ name: 'Upload' })
}

/**
 * 跳转到上传页面
 */
const goToUpload = () => {
  router.push({ name: 'Upload' })
}

/**
 * 处理新上传的任务
 */
const handleNewTask = (taskId: string) => {
  const newTask: Task = {
    task_id: taskId,
    status: 'PENDING',
    created_at: Date.now(),
    updated_at: Date.now()
  }

  addTaskToStorage(newTask)
  taskList.value.unshift(newTask)

  taskPoller.start(taskId, response => {
    updateTaskStatus(taskId, response)
  })
}

onMounted(() => {
  loadTaskList()

  const queryTaskId = route.query.taskId as string
  if (queryTaskId) {
    highlightTaskId.value = queryTaskId

    if (!taskList.value.some(t => t.task_id === queryTaskId)) {
      handleNewTask(queryTaskId)
    }

    setTimeout(() => {
      highlightTaskId.value = ''
    }, 3000)
  }

  initPolling()
})

onUnmounted(() => {
  taskPoller.stopAll()
})
</script>

<style scoped>
.task-list-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 32px 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;

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

  .header-actions {
    display: flex;
    gap: 12px;
  }
}

.empty-state {
  margin-top: 80px;
}

.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 20px;
}

@media (max-width: 768px) {
  .task-grid {
    grid-template-columns: 1fr;
  }
}
</style>
