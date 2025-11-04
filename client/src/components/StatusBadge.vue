<template>
  <span class="status-badge" :class="`status-${statusConfig.type}`">
    <span class="badge-dot"></span>
    <span class="badge-text">{{ statusConfig.text }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TaskStatus } from '@/api/types'

interface Props {
  status: TaskStatus
}

const props = defineProps<Props>()

const statusConfig = computed(() => {
  const configs = {
    PENDING: { text: '排队中', type: 'info' },
    PROCESSING: { text: '处理中', type: 'warning' },
    COMPLETED: { text: '已完成', type: 'success' },
    FAILED: { text: '失败', type: 'danger' }
  }
  return configs[props.status]
})
</script>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;

  .badge-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }
}

.status-info {
  background: #eff6ff;
  color: #3b82f6;

  .badge-dot {
    background: #3b82f6;
  }
}

.status-warning {
  background: #fef3c7;
  color: #f59e0b;

  .badge-dot {
    background: #f59e0b;
    animation: pulse 2s infinite;
  }
}

.status-success {
  background: #dcfce7;
  color: #10b981;

  .badge-dot {
    background: #10b981;
  }
}

.status-danger {
  background: #fee2e2;
  color: #ef4444;

  .badge-dot {
    background: #ef4444;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
