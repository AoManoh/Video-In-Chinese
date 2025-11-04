<template>
  <el-card class="task-card" :class="{ highlight: highlight }">
    <!-- 任务标题 -->
    <template #header>
      <div class="card-header">
        <span class="task-id">任务 {{ task.task_id.slice(0, 8) }}</span>
        <el-tag :type="statusConfig.color">{{ statusConfig.text }}</el-tag>
      </div>
    </template>

    <!-- 任务信息 -->
    <div class="task-info">
      <!-- 状态图标 -->
      <div class="status-icon">
        <el-icon v-if="task.status === 'PENDING'" class="pending">
          <Clock />
        </el-icon>
        <el-icon v-else-if="task.status === 'PROCESSING'" class="processing rotating">
          <Loading />
        </el-icon>
        <el-icon v-else-if="task.status === 'COMPLETED'" class="completed">
          <CircleCheck />
        </el-icon>
        <el-icon v-else-if="task.status === 'FAILED'" class="failed">
          <CircleClose />
        </el-icon>
      </div>

      <!-- 状态描述 -->
      <div class="status-desc">
        <p v-if="task.status === 'PENDING'">任务已创建，正在排队等待处理...</p>
        <p v-else-if="task.status === 'PROCESSING'">
          任务正在处理中，请耐心等待...
          <br />
          <el-text type="info" size="small">（处理时间约为视频时长的3倍）</el-text>
        </p>
        <p v-else-if="task.status === 'COMPLETED'">
          任务已完成！
          <br />
          <el-link type="primary" @click="$emit('download', task)">点击下载结果</el-link>
        </p>
        <p v-else-if="task.status === 'FAILED'">
          任务处理失败
          <br />
          <el-text type="danger" size="small">错误信息：{{ task.error_message }}</el-text>
        </p>
      </div>
    </div>

    <!-- 操作按钮 -->
    <template #footer>
      <div class="card-footer">
        <el-button
          v-if="task.status === 'COMPLETED'"
          type="primary"
          size="small"
          @click="$emit('download', task)"
        >
          下载结果
        </el-button>
        <el-button
          v-if="task.status === 'FAILED'"
          type="warning"
          size="small"
          @click="$emit('retry', task)"
        >
          重新上传
        </el-button>
        <el-text type="info" size="small">创建时间：{{ formatTime(task.created_at) }}</el-text>
      </div>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Clock, Loading, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import type { Task } from '@/api/types'

interface Props {
  task: Task
  highlight?: boolean
}

interface Emits {
  (e: 'download', task: Task): void
  (e: 'retry', task: Task): void
}

const props = defineProps<Props>()
defineEmits<Emits>()

// 状态配置
const statusConfig = computed(() => {
  const configs = {
    PENDING: { text: '排队中', color: 'info' as const },
    PROCESSING: { text: '处理中', color: 'warning' as const },
    COMPLETED: { text: '已完成', color: 'success' as const },
    FAILED: { text: '失败', color: 'danger' as const }
  }
  return configs[props.task.status]
})

/**
 * 格式化时间
 */
const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  const second = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}
</script>

<style scoped>
.task-card {
  margin-bottom: 16px;
  transition: all 0.3s;

  &.highlight {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 10px rgba(64, 158, 255, 0.3);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .task-id {
      font-weight: bold;
    }
  }

  .task-info {
    .status-icon {
      font-size: 48px;
      text-align: center;
      margin-bottom: 16px;

      .pending {
        color: var(--el-color-info);
      }

      .processing {
        color: var(--el-color-warning);
      }

      .completed {
        color: var(--el-color-success);
      }

      .failed {
        color: var(--el-color-danger);
      }

      .rotating {
        animation: rotate 2s linear infinite;
      }
    }

    .status-desc {
      text-align: center;
      min-height: 60px;

      p {
        margin: 0;
        line-height: 1.6;
      }
    }
  }

  .card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>

