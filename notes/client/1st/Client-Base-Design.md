# 客户端架构设计文档（第一层）

**文档版本**: 1.0
**创建日期**: 2025-11-03
**关联后端文档**: 
- `notes/server/1st/Base-Design.md` v2.2
- `notes/server/2nd/Gateway-design.md` v5.9

---

## 版本历史

- **v1.0 (2025-11-03)**:
  - 初始版本，定义客户端架构设计
  - 技术栈选型（Vue 3 + TypeScript + Element Plus）
  - 项目工程结构
  - 路由设计
  - 与后端对接规范
  - Mock数据方案

---

## 1. 项目愿景与MVP范围

### 1.1 项目定位

**单用户自部署的视频翻译服务Web客户端**，提供简洁直观的用户界面，支持用户完成以下核心功能：

1. **配置AI服务**：通过Web界面配置API密钥和服务商选择
2. **上传视频文件**：选择本地视频文件并上传到服务器
3. **查看任务状态**：实时查看任务处理进度
4. **下载翻译结果**：任务完成后下载翻译后的视频

### 1.2 MVP核心范围

- **产品形态**: 单页面Web应用（SPA），无需登录
- **核心流程**: 
  1. 首次访问 → 配置向导（引导用户配置API密钥）
  2. 上传视频 → 自动跳转任务列表
  3. 轮询查询状态 → 下载结果
- **技术路径**: 前后端分离，通过RESTful API与后端交互
- **部署方式**: 静态文件部署（Nginx/Apache），与后端服务分离

### 1.3 非功能性需求

- **浏览器兼容性**: 现代浏览器（Chrome 90+、Firefox 88+、Safari 14+、Edge 90+）
- **响应式设计**: 支持桌面端（1920x1080）和平板端（1024x768），暂不支持移动端
- **性能要求**: 
  - 首屏加载时间 < 2秒
  - 页面交互响应 < 100ms
  - 文件上传支持大文件（最大2048MB）
- **可访问性**: 遵循WCAG 2.1 AA标准（可选）

---

## 2. 技术栈选型

### 2.1 核心技术栈

| 技术 | 版本 | 选型理由 |
|------|------|----------|
| **Vue 3** | 3.3+ | 组合式API，TypeScript支持好，生态成熟 |
| **TypeScript** | 5.0+ | 类型安全，减少运行时错误，IDE支持好 |
| **Element Plus** | 2.4+ | 中文文档友好，组件完整，开发效率高 |
| **Vite** | 5.0+ | 快速开发服务器，优化的生产构建 |
| **Vue Router** | 4.0+ | 官方路由库，与Vue 3完美集成 |
| **Axios** | 1.6+ | 成熟的HTTP客户端，拦截器支持好 |
| **Pinia** | 2.1+ | 官方状态管理库（可选，MVP可用组合式API） |

### 2.2 开发工具

| 工具 | 用途 |
|------|------|
| **ESLint** | 代码质量检查 |
| **Prettier** | 代码格式化 |
| **axios-mock-adapter** | Mock数据（开发阶段） |
| **Vitest** | 单元测试（可选） |
| **Playwright** | E2E测试（可选） |

### 2.3 依赖库清单

```json
{
  "dependencies": {
    "vue": "^3.3.0",
    "vue-router": "^4.0.0",
    "element-plus": "^2.4.0",
    "axios": "^1.6.0",
    "@element-plus/icons-vue": "^2.3.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^4.5.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0",
    "eslint": "^8.50.0",
    "prettier": "^3.0.0",
    "axios-mock-adapter": "^1.22.0"
  }
}
```

---

## 3. 项目工程结构

```
client/
├── public/                          # 静态资源
│   └── favicon.ico
├── src/
│   ├── assets/                      # 静态资源（图片、样式等）
│   │   └── logo.png
│   ├── views/                       # 页面组件
│   │   ├── SettingsView.vue         # 配置管理页面
│   │   ├── UploadView.vue           # 任务上传页面
│   │   └── TaskListView.vue         # 任务列表页面
│   ├── components/                  # 公共组件
│   │   ├── TaskCard.vue             # 任务卡片组件
│   │   ├── UploadProgress.vue       # 上传进度组件
│   │   └── StatusBadge.vue          # 状态徽章组件
│   ├── api/                         # API接口层
│   │   ├── index.ts                 # API入口
│   │   ├── types.ts                 # TypeScript类型定义
│   │   ├── settings-api.ts          # 配置管理API
│   │   └── task-api.ts              # 任务管理API
│   ├── utils/                       # 工具函数
│   │   ├── http-client.ts           # Axios封装
│   │   ├── storage.ts               # localStorage封装
│   │   └── format.ts                # 格式化工具
│   ├── mock/                        # Mock数据（开发阶段）
│   │   ├── index.ts                 # Mock入口
│   │   ├── settings.ts              # 配置管理Mock
│   │   └── task.ts                  # 任务管理Mock
│   ├── router/                      # 路由配置
│   │   └── index.ts
│   ├── styles/                      # 全局样式
│   │   └── index.css
│   ├── App.vue                      # 根组件
│   └── main.ts                      # 应用入口
├── .env.development                 # 开发环境变量
├── .env.production                  # 生产环境变量
├── .eslintrc.cjs                    # ESLint配置
├── .prettierrc.json                 # Prettier配置
├── index.html                       # HTML模板
├── package.json                     # 依赖管理
├── tsconfig.json                    # TypeScript配置
└── vite.config.ts                   # Vite配置
```

---

## 4. 路由设计

### 4.1 路由表

```typescript
const routes = [
  {
    path: '/',
    redirect: '/settings'  // 首次访问默认跳转到配置页面
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: { 
      title: '配置管理',
      requiresConfig: false  // 不需要检查配置完成状态
    }
  },
  {
    path: '/upload',
    name: 'Upload',
    component: () => import('@/views/UploadView.vue'),
    meta: { 
      title: '上传视频',
      requiresConfig: true   // 需要先完成配置
    }
  },
  {
    path: '/tasks',
    name: 'TaskList',
    component: () => import('@/views/TaskListView.vue'),
    meta: { 
      title: '任务列表',
      requiresConfig: true   // 需要先完成配置
    }
  }
]
```

### 4.2 路由守卫

**配置检查守卫**：确保用户完成配置后才能访问上传和任务列表页面

```typescript
router.beforeEach(async (to, from, next) => {
  // 检查是否需要配置验证
  if (to.meta.requiresConfig) {
    try {
      // 调用后端API检查配置状态
      const settings = await getSettings()
      
      if (!settings.is_configured) {
        // 未配置，跳转到配置页面
        ElMessage.warning('请先完成基本配置')
        next({ name: 'Settings' })
        return
      }
    } catch (error) {
      // API调用失败，允许访问（降级策略）
      console.error('Failed to check configuration:', error)
    }
  }
  
  next()
})
```

### 4.3 页面跳转流程

```mermaid
graph TD
    A[首次访问] --> B[/]
    B --> C{检查is_configured}
    C -->|false| D[跳转到/settings]
    C -->|true| E[跳转到/tasks]
    D --> F[用户配置API密钥]
    F --> G[保存配置]
    G --> H[跳转到/upload]
    H --> I[用户上传视频]
    I --> J[跳转到/tasks]
    J --> K[查看任务状态]
```

---

## 5. 与后端对接规范

### 5.1 后端API基础信息

**后端服务**: Gateway服务（RESTful API）
**参考文档**: `notes/server/2nd/Gateway-design.md` v5.9

| 后端接口 | 方法 | 前端调用位置 |
|---------|------|-------------|
| `/v1/settings` | GET | SettingsView（获取配置） |
| `/v1/settings` | POST | SettingsView（更新配置） |
| `/v1/tasks/upload` | POST | UploadView（上传文件） |
| `/v1/tasks/:taskId/status` | GET | TaskListView（查询状态） |
| `/v1/tasks/download/:taskId/:fileName` | GET | TaskListView（下载文件） |

### 5.2 接口对齐策略

**原则1：类型定义完全一致**
- 前端TypeScript类型必须与后端API契约一致
- 字段命名使用snake_case（与后端保持一致）
- 枚举值使用大写字符串（如 "PENDING"、"PROCESSING"）

**原则2：每个API调用必须标注后端接口**
```typescript
/**
 * 获取应用配置
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第5章 第276-277行
 * @returns GetSettingsResponse
 */
export const getSettings = async (): Promise<GetSettingsResponse> => {
  // ...
}
```

**原则3：错误处理与后端错误码对齐**
- 前端必须处理后端定义的所有HTTP状态码（400、404、409、413、415、500、503、507）
- 错误消息格式：`{ code: string, message: string }`
- 参考：`Gateway-design.md` 第8章"错误码清单"

### 5.3 HTTP客户端封装

```typescript
// utils/http-client.ts
import axios from 'axios'
import { ElMessage } from 'element-plus'

const httpClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 300000,  // 5分钟超时（与后端HTTP_TIMEOUT_SECONDS对齐）
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
httpClient.interceptors.request.use(
  config => {
    // 开发阶段：打印请求信息
    if (import.meta.env.DEV) {
      console.log('[API Request]', config.method?.toUpperCase(), config.url)
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
httpClient.interceptors.response.use(
  response => {
    return response
  },
  error => {
    if (axios.isAxiosError(error)) {
      const status = error.response?.status
      const message = error.response?.data?.message || '请求失败'
      
      // 根据后端错误码映射（Gateway-design.md 第8章）
      switch (status) {
        case 400:
          ElMessage.error(`请求参数错误: ${message}`)
          break
        case 404:
          ElMessage.error('资源不存在')
          break
        case 409:
          ElMessage.warning('配置冲突，请刷新后重试')
          break
        case 413:
          ElMessage.error('文件太大，超过2048MB限制')
          break
        case 415:
          ElMessage.error('不支持的文件格式')
          break
        case 503:
          ElMessage.error('服务暂时不可用')
          break
        case 507:
          ElMessage.error('服务器存储空间不足')
          break
        default:
          ElMessage.error(message)
      }
    } else {
      ElMessage.error('网络错误')
    }
    
    return Promise.reject(error)
  }
)

export default httpClient
```

---

## 6. Mock数据方案（开发阶段）

### 6.1 Mock方案选择

**MVP阶段采用**: axios-mock-adapter（简单易用，快速搭建）

**切换机制**: 环境变量控制
```bash
# .env.development
VITE_USE_MOCK=true
VITE_API_BASE_URL=http://localhost:8080

# .env.production
VITE_USE_MOCK=false
VITE_API_BASE_URL=https://api.production.com
```

### 6.2 Mock数据结构

```
src/mock/
├── index.ts           # Mock入口，根据环境变量启用/禁用
├── settings.ts        # 配置管理API的Mock实现
├── task.ts            # 任务管理API的Mock实现
└── data/              # Mock数据JSON文件
    ├── settings.json  # 默认配置数据
    └── tasks.json     # 示例任务数据
```

### 6.3 Mock实现示例

```typescript
// mock/index.ts
import MockAdapter from 'axios-mock-adapter'
import httpClient from '@/utils/http-client'
import { setupSettingsMock } from './settings'
import { setupTaskMock } from './task'

export const setupMock = () => {
  const mock = new MockAdapter(httpClient, { delayResponse: 500 })  // 模拟500ms延迟
  
  setupSettingsMock(mock)
  setupTaskMock(mock)
  
  console.log('[Mock] Mock数据已启用')
}

// main.ts
if (import.meta.env.VITE_USE_MOCK === 'true') {
  const { setupMock } = await import('./mock')
  setupMock()
}
```

### 6.4 后端实现后的切换步骤

**步骤1**: 修改环境变量
```bash
# .env.development
VITE_USE_MOCK=false  # 关闭Mock
VITE_API_BASE_URL=http://localhost:8080  # 指向真实后端
```

**步骤2**: 验证接口对齐
- 逐个API验证请求/响应格式
- 检查错误处理是否正确
- 确认所有HTTP状态码处理

**步骤3**: 删除Mock代码（可选）
```bash
# 后端稳定后，可删除Mock相关代码
rm -rf src/mock/
```

---

## 7. 状态管理策略

### 7.1 MVP阶段：组合式API + localStorage

**理由**：
- MVP功能简单，无复杂的状态共享需求
- 组合式API足够应对3个页面的状态管理
- localStorage用于持久化配置信息（避免频繁调用后端）

**实现示例**：
```typescript
// utils/storage.ts
export const storage = {
  // 存储配置信息（用于缓存is_configured状态）
  setConfigStatus(isConfigured: boolean) {
    localStorage.setItem('is_configured', String(isConfigured))
  },
  
  getConfigStatus(): boolean {
    return localStorage.getItem('is_configured') === 'true'
  },
  
  // 存储任务列表（用于断点续传）
  setTaskList(tasks: Task[]) {
    localStorage.setItem('task_list', JSON.stringify(tasks))
  },
  
  getTaskList(): Task[] {
    const data = localStorage.getItem('task_list')
    return data ? JSON.parse(data) : []
  }
}
```

### 7.2 后续扩展：Pinia

如果后续功能扩展，可引入Pinia进行状态管理：
- 用户会话管理（V2.0+）
- 任务历史记录（V2.0+）
- 全局配置状态（V2.0+）

---

## 8. 文件上传策略

### 8.1 大文件上传

**后端限制**: 最大2048MB（`MAX_UPLOAD_SIZE_MB`）

**前端策略**:
- 使用FormData上传（支持大文件）
- 显示上传进度条（Axios onUploadProgress）
- 客户端预检查文件大小和格式
- 支持取消上传

**实现示例**：
```typescript
const uploadFile = async (file: File, onProgress: (percent: number) => void) => {
  // 客户端预检查
  if (file.size > MAX_UPLOAD_SIZE_MB * 1024 * 1024) {
    throw new Error('文件超过2048MB限制')
  }
  
  const formData = new FormData()
  formData.append('file', file)
  
  const response = await httpClient.post('/v1/tasks/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    onUploadProgress: (progressEvent) => {
      const percent = Math.round((progressEvent.loaded * 100) / (progressEvent.total || 1))
      onProgress(percent)
    }
  })
  
  return response.data
}
```

### 8.2 文件格式验证

**后端支持的格式**: MP4、MOV、MKV（`SUPPORTED_MIME_TYPES`）

**前端验证**:
```typescript
const ALLOWED_VIDEO_TYPES = ['video/mp4', 'video/quicktime', 'video/x-matroska']

const validateFile = (file: File): boolean => {
  if (!ALLOWED_VIDEO_TYPES.includes(file.type)) {
    ElMessage.error('仅支持MP4、MOV、MKV格式')
    return false
  }
  return true
}
```

---

## 9. 任务状态轮询策略

### 9.1 轮询机制

**后端建议**: 前端轮询间隔3秒，指数退避至最多10秒（`Gateway-design.md` 第2.4节）

**实现策略**:
```typescript
class TaskPoller {
  private interval = 3000  // 初始3秒
  private maxInterval = 10000  // 最大10秒
  private timerId: number | null = null
  
  start(taskId: string, onUpdate: (status: TaskStatus) => void) {
    this.poll(taskId, onUpdate)
  }
  
  private async poll(taskId: string, onUpdate: (status: TaskStatus) => void) {
    try {
      const response = await getTaskStatus(taskId)
      onUpdate(response.status)
      
      // 任务完成或失败，停止轮询
      if (response.status === 'COMPLETED' || response.status === 'FAILED') {
        this.stop()
        return
      }
      
      // 指数退避（3s → 6s → 10s）
      this.interval = Math.min(this.interval * 2, this.maxInterval)
      
      // 继续轮询
      this.timerId = window.setTimeout(() => this.poll(taskId, onUpdate), this.interval)
    } catch (error) {
      console.error('轮询失败:', error)
      this.stop()
    }
  }
  
  stop() {
    if (this.timerId) {
      clearTimeout(this.timerId)
      this.timerId = null
    }
    this.interval = 3000  // 重置间隔
  }
}
```

### 9.2 状态映射

```typescript
const STATUS_MAP = {
  PENDING: { text: '排队中', color: 'info' },
  PROCESSING: { text: '处理中', color: 'warning' },
  COMPLETED: { text: '已完成', color: 'success' },
  FAILED: { text: '失败', color: 'danger' }
}
```

---

## 10. 构建与部署

### 10.1 构建命令

```bash
# 开发环境（启用Mock数据）
npm run dev

# 生产构建
npm run build

# 预览生产构建
npm run preview
```

### 10.2 环境变量配置

```bash
# .env.development
VITE_USE_MOCK=true
VITE_API_BASE_URL=http://localhost:8080

# .env.production
VITE_USE_MOCK=false
VITE_API_BASE_URL=https://your-domain.com
```

### 10.3 部署方案

**方案1: Nginx部署**
```nginx
server {
    listen 80;
    server_name your-domain.com;
    root /var/www/video-translator/client/dist;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # API代理（避免CORS问题）
    location /v1/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

**方案2: Docker部署**
```dockerfile
# Dockerfile
FROM node:18-alpine as builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

---

## 11. MVP的局限性与未来迭代方向

### 11.1 MVP阶段局限性

**功能层面**:
- 无用户系统（单用户自部署）
- 无任务历史记录
- 无任务取消功能
- 不支持移动端

**技术层面**:
- 无单元测试
- 无E2E测试
- 无国际化支持
- 无PWA支持

### 11.2 V2.0迭代方向

**功能扩展**:
- 用户登录/注册
- 任务历史记录
- 任务管理（取消、删除）
- WebSocket实时状态推送
- 移动端适配

**技术优化**:
- 引入Pinia状态管理
- 添加单元测试（Vitest）
- 添加E2E测试（Playwright）
- 国际化支持（vue-i18n）
- PWA支持（离线访问）

---

## 12. 文档变更历史

| 版本 | 日期       | 变更内容                         |
| ---- | ---------- | -------------------------------- |
| 1.0  | 2025-11-03 | 初始版本，定义客户端架构设计     |

---

**文档结束**
