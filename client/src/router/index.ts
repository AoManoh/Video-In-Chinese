/**
 * Vue Router配置
 * 
 * @reference notes/client/1st/Client-Base-Design.md 第4节
 */

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getSettings } from '@/api/settings-api'
import { getConfigStatus, setConfigStatus } from '@/utils/storage'

/**
 * 路由定义
 */
const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/settings' // 首次访问默认跳转到配置页面
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: {
      title: '配置管理',
      requiresConfig: false // 不需要检查配置完成状态
    }
  },
  {
    path: '/upload',
    name: 'Upload',
    component: () => import('@/views/UploadView.vue'),
    meta: {
      title: '上传视频',
      requiresConfig: true // 需要先完成配置
    }
  },
  {
    path: '/tasks',
    name: 'TaskList',
    component: () => import('@/views/TaskListView.vue'),
    meta: {
      title: '任务列表',
      requiresConfig: true // 需要先完成配置
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

/**
 * 路由守卫：配置检查
 * 
 * 确保用户完成配置后才能访问上传和任务列表页面
 */
router.beforeEach(async (to, _from, next) => {
  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} - 视频翻译服务`
  }

  // 检查是否需要配置验证
  if (to.meta.requiresConfig) {
    try {
      // 先检查localStorage缓存（优化性能）
      const cachedStatus = getConfigStatus()

      if (!cachedStatus) {
        // 缓存显示未配置，调用API再次确认
        const settings = await getSettings()

        if (!settings.is_configured) {
          // 未配置，跳转到配置页面
          ElMessage.warning('请先完成基本配置（ASR、翻译、声音克隆）')
          next({ name: 'Settings' })
          return
        }

        // 更新缓存
        setConfigStatus(settings.is_configured)
      }
    } catch (error) {
      // API调用失败，允许访问（降级策略）
      console.error('Failed to check configuration:', error)
      // 不阻止访问，避免路由死锁
    }
  }

  next()
})

export default router

