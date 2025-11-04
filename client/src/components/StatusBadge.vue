<template>
  <el-tag :type="statusConfig.color">{{ statusConfig.text }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TaskStatus } from '@/api/types'

interface Props {
  status: TaskStatus
}

const props = defineProps<Props>()

// 状态配置
const statusConfig = computed(() => {
  const configs = {
    PENDING: { text: '排队中', color: 'info' as const },
    PROCESSING: { text: '处理中', color: 'warning' as const },
    COMPLETED: { text: '已完成', color: 'success' as const },
    FAILED: { text: '失败', color: 'danger' as const }
  }
  return configs[props.status]
})
</script>

