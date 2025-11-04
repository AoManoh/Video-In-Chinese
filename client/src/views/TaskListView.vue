<template>
  <div class="task-list-page">
    <el-page-header content="任务列表">
      <template #extra>
        <el-button @click="refreshAllTasks" :loading="refreshing" :icon="Refresh">
          刷新
        </el-button>
        <el-button type="primary" @click="goToUpload">上传新视频</el-button>
      </template>
    </el-page-header>

    <!-- 空状态 -->
    <el-empty
      v-if="taskList.length === 0"
      description="暂无任务"
      :image-size="200"
      class="mt-20"
    >
      <el-button type="primary" @click="goToUpload">立即上传</el-button>
    </el-empty>

    <!-- 任务列表 -->
    <div v-else class="task-grid mt-20">
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
import { Refresh } from '@element-plus/icons-vue'
import TaskCard from '@/components/TaskCard.vue'
import { getTaskStatus, downloadFile, triggerDownload } from '@/api/task-api'
import type { Task } from '@/api/types'
import { getTaskList, setTaskList, addTask as addTaskToStorage, updateTask } from '@/utils/storage'
import { taskPoller } from '@/utils/task-poller'

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
 * 保存任务列表
 */
const saveTaskList = () => {
  setTaskList(taskList.value)
}

/**
 * 初始化轮询
 */
const initPolling = () => {
  taskList.value.forEach(task => {
    // 只轮询处理中的任务
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

  // 更新storage
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
        title: '任务完成',
        message: `任务 ${taskId.slice(0, 8)} 已完成，可以下载结果`,
        duration: 0 // 不自动关闭
      })
    } else if (response.status === 'FAILED') {
      ElNotification.error({
        title: '任务失败',
        message: `任务 ${taskId.slice(0, 8)} 处理失败：${response.error_message}`,
        duration: 0 // 不自动关闭
      })
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

  // 解析result_url
  // 格式：/v1/tasks/download/{taskId}/{fileName}
  const urlParts = task.result_url.split('/')
  const taskIdFromUrl = urlParts[urlParts.length - 2]
  const fileName = urlParts[urlParts.length - 1]

  try {
    ElMessage.info('正在下载，请稍候...')

    const blob = await downloadFile(taskIdFromUrl, fileName)
    triggerDownload(blob, fileName)

    ElMessage.success('下载成功')
  } catch (error) {
    // 错误已由http-client拦截器处理
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
  // 添加到列表
  const newTask: Task = {
    task_id: taskId,
    status: 'PENDING',
    created_at: Date.now(),
    updated_at: Date.now()
  }

  addTaskToStorage(newTask)
  taskList.value.unshift(newTask)

  // 开始轮询
  taskPoller.start(taskId, response => {
    updateTaskStatus(taskId, response)
  })
}

onMounted(() => {
  // 加载任务列表
  loadTaskList()

  // 处理query参数中的taskId（来自上传页面）
  const queryTaskId = route.query.taskId as string
  if (queryTaskId) {
    highlightTaskId.value = queryTaskId

    // 检查任务是否在列表中
    if (!taskList.value.some(t => t.task_id === queryTaskId)) {
      handleNewTask(queryTaskId)
    }

    // 3秒后取消高亮
    setTimeout(() => {
      highlightTaskId.value = ''
    }, 3000)
  }

  // 启动轮询
  initPolling()
})

onUnmounted(() => {
  // 停止所有轮询
  taskPoller.stopAll()
})
</script>

<style scoped>
.task-list-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 16px;
}

.mt-20 {
  margin-top: 20px;
}
</style>

