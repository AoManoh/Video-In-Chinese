<template>
  <el-container class="app-container">
    <!-- 顶部导航栏 -->
    <el-header class="app-header">
      <div class="header-content">
        <div class="brand">
          <svg class="brand-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
          <h1 class="brand-title">视频翻译</h1>
        </div>
        <el-menu
          :default-active="currentRoute"
          mode="horizontal"
          :ellipsis="false"
          @select="handleMenuSelect"
        >
          <el-menu-item index="/settings">
            <el-icon><Setting /></el-icon>
            <span>配置</span>
          </el-menu-item>
          <el-menu-item index="/upload">
            <el-icon><Upload /></el-icon>
            <span>上传</span>
          </el-menu-item>
          <el-menu-item index="/tasks">
            <el-icon><List /></el-icon>
            <span>任务</span>
          </el-menu-item>
        </el-menu>
      </div>
    </el-header>

    <!-- 主内容区域 -->
    <el-main class="app-main">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Setting, Upload, List } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

// 当前路由路径
const currentRoute = computed(() => route.path)

/**
 * 菜单选择处理
 */
const handleMenuSelect = (index: string) => {
  router.push(index)
}
</script>

<style scoped>
.app-container {
  min-height: 100vh;
  background: linear-gradient(to bottom, #f9fafb 0%, #f3f4f6 100%);
}

.app-header {
  background-color: #ffffff;
  border-bottom: 1px solid #e5e7eb;
  padding: 0;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  height: var(--app-header-height);

  .header-content {
    max-width: 1400px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
    padding: 0 32px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
    transition: opacity 0.2s;

    &:hover {
      opacity: 0.8;
    }

    .brand-icon {
      width: 32px;
      height: 32px;
      color: var(--el-color-primary);
    }

    .brand-title {
      font-size: 20px;
      font-weight: 700;
      margin: 0;
      background: linear-gradient(135deg, var(--el-color-primary) 0%, #6366f1 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
  }

  .el-menu {
    border-bottom: none;
    background-color: transparent;
  }

  :deep(.el-menu-item) {
    font-weight: 500;
    border-radius: 8px;
    margin: 0 4px;
    
    &:hover {
      background-color: #f3f4f6;
    }
  }
}

.app-main {
  min-height: calc(100vh - var(--app-header-height));
  padding: 0;
}

/* 页面切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
