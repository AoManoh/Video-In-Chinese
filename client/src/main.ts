/**
 * 应用入口
 */

import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './styles/index.css'
import App from './App.vue'
import router from './router'
import { cleanupExpiredTasks } from './utils/storage'

// 条件导入Mock（开发阶段）
if (import.meta.env.VITE_USE_MOCK === 'true') {
  import('./mock').then(({ setupMock }) => {
    setupMock()
  })
}

// 清理过期任务（7天前的任务）
cleanupExpiredTasks()

const app = createApp(App)

app.use(ElementPlus)
app.use(router)

app.mount('#app')
