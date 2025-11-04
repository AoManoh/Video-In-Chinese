<template>
  <el-progress :percentage="progress" :status="progressStatus" class="upload-progress">
    <template #default="{ percentage }">
      <span>{{ percentage }}%</span>
      <span v-if="speed" class="speed-text"> ({{ speed }})</span>
    </template>
  </el-progress>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  progress: number
  speed?: string
  status?: 'uploading' | 'success' | 'error'
}

const props = defineProps<Props>()

// 进度条状态
const progressStatus = computed(() => {
  if (props.status === 'success') return 'success'
  if (props.status === 'error') return 'exception'
  return undefined
})
</script>

<style scoped>
.upload-progress {
  width: 100%;
}

.speed-text {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>

