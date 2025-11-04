<template>
  <el-container class="app-container">
    <!-- 顶部导航栏 -->
    <el-header class="app-header">
      <div class="header-content">
        <h1 class="app-title">视频翻译服务</h1>
        <el-menu
          :default-active="currentRoute"
          mode="horizontal"
          :ellipsis="false"
          @select="handleMenuSelect"
        >
          <el-menu-item index="/settings">配置管理</el-menu-item>
          <el-menu-item index="/upload">上传视频</el-menu-item>
          <el-menu-item index="/tasks">任务列表</el-menu-item>
        </el-menu>
      </div>
    </el-header>

    <!-- 主内容区域 -->
    <el-main class="app-main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'

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
}

.app-header {
  background-color: #ffffff;
  border-bottom: 1px solid #dcdfe6;
  padding: 0;

  .header-content {
    max-width: 1400px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
    padding: 0 20px;
  }

  .app-title {
    font-size: 20px;
    font-weight: 600;
    margin: 0;
    color: #303133;
  }

  .el-menu {
    border-bottom: none;
  }
}

.app-main {
  background-color: #f5f7fa;
  min-height: calc(100vh - 60px);
}
</style>
