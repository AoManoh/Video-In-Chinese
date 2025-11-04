<template>
  <el-card class="task-card" :class="{ highlight: highlight }" shadow="never">
    <!-- 任务头部 -->
    <div class="card-header">
      <div class="task-id-wrapper">
        <el-icon class="id-icon"><Document /></el-icon>
        <span class="task-id">{{ task.task_id.slice(0, 12) }}...</span>
      </div>
      <StatusBadge :status="task.status" />
    </div>

    <!-- 任务状态展示 -->
    <div class="task-status">
      <!-- 状态图标 -->
      <div class="status-icon-wrapper">
        <el-icon v-if="task.status === 'PENDING'" class="status-icon pending">
          <Clock />
        </el-icon>
        <el-icon v-else-if="task.status === 'PROCESSING'" class="status-icon processing rotating">
          <Loading />
        </el-icon>
        <el-icon v-else-if="task.status === 'COMPLETED'" class="status-icon completed">
          <CircleCheck />
        </el-icon>
        <el-icon v-else-if="task.status === 'FAILED'" class="status-icon failed">
          <CircleClose />
        </el-icon>
      </div>

      <!-- 状态描述 -->
      <div class="status-description">
        <p v-if="task.status === 'PENDING'" class="status-text">
          任务已创建，正在排队...
        </p>
        <p v-else-if="task.status === 'PROCESSING'" class="status-text">
          正在处理中，请耐心等待
          <br />
          <span class="status-hint">处理时间约为视频时长的 3 倍</span>
        </p>
        <p v-else-if="task.status === 'COMPLETED'" class="status-text">
          任务已完成！
        </p>
        <p v-else-if="task.status === 'FAILED'" class="status-text">
          任务处理失败
          <br />
          <span class="status-error">{{ task.error_message }}</span>
        </p>
      </div>
    </div>

    <!-- 任务信息 -->
    <div class="task-info">
      <div class="info-item">
        <el-icon class="info-icon"><Calendar /></el-icon>
        <span class="info-text">{{ formatTime(task.created_at) }}</span>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="task-actions">
      <el-button
        v-if="task.status === 'COMPLETED'"
        type="primary"
        size="default"
        @click="$emit('download', task)"
        style="width: 100%"
      >
        <el-icon><Download /></el-icon>
        下载结果
      </el-button>
      <el-button
        v-if="task.status === 'FAILED'"
        type="warning"
        size="default"
        @click="$emit('retry', task)"
        style="width: 100%"
      >
        <el-icon><RefreshLeft /></el-icon>
        重新上传
      </el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import {
  Clock,
  Loading,
  CircleCheck,
  CircleClose,
  Document,
  Calendar,
  Download,
  RefreshLeft
} from '@element-plus/icons-vue'
import StatusBadge from './StatusBadge.vue'
import type { Task } from '@/api/types'

interface Props {
  task: Task
  highlight?: boolean
}

interface Emits {
  (e: 'download', task: Task): void
  (e: 'retry', task: Task): void
}

defineProps<Props>()
defineEmits<Emits>()

/**
 * 格式化时间
 */
const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  // 少于1分钟
  if (diff < 60000) {
    return '刚刚'
  }

  // 少于1小时
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return `${minutes}分钟前`
  }

  // 少于24小时
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    return `${hours}小时前`
  }

  // 超过24小时，显示日期
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  
  // 同一年份只显示月日时分
  if (year === now.getFullYear()) {
    return `${month}-${day} ${hour}:${minute}`
  }
  
  return `${year}-${month}-${day}`
}
</script>

<style scoped>
.task-card {
  border: 1px solid #e5e7eb;
  border-radius: var(--app-border-radius);
  transition: all 0.3s;
  background: #ffffff;

  &:hover {
    transform: translateY(-4px);
    box-shadow: var(--app-shadow-lg);
  }

  &.highlight {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }

  :deep(.el-card__body) {
    padding: 20px;
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f3f4f6;

  .task-id-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;

    .id-icon {
      font-size: 18px;
      color: #6b7280;
    }

    .task-id {
      font-size: 13px;
      font-family: 'Courier New', monospace;
      color: #4b5563;
      font-weight: 500;
    }
  }
}

.task-status {
  margin-bottom: 20px;

  .status-icon-wrapper {
    text-align: center;
    margin-bottom: 16px;

    .status-icon {
      font-size: 56px;

      &.pending {
        color: var(--el-color-info);
      }

      &.processing {
        color: var(--el-color-warning);
      }

      &.completed {
        color: var(--el-color-success);
      }

      &.failed {
        color: var(--el-color-danger);
      }

      &.rotating {
        animation: rotate 2s linear infinite;
      }
    }
  }

  .status-description {
    text-align: center;
    min-height: 50px;

    .status-text {
      font-size: 15px;
      color: #374151;
      margin: 0;
      line-height: 1.6;
    }

    .status-hint {
      font-size: 13px;
      color: #6b7280;
    }

    .status-error {
      font-size: 13px;
      color: var(--el-color-danger);
    }
  }
}

.task-info {
  padding: 12px;
  background: #f9fafb;
  border-radius: 8px;
  margin-bottom: 16px;

  .info-item {
    display: flex;
    align-items: center;
    gap: 8px;

    .info-icon {
      font-size: 16px;
      color: #6b7280;
    }

    .info-text {
      font-size: 13px;
      color: #6b7280;
    }
  }
}

.task-actions {
  display: flex;
  gap: 8px;
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
