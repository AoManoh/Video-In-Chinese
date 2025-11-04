# 客户端设计规范（精简版）

**文档版本**: 1.0
**创建日期**: 2025-11-03
**适用范围**: 视频翻译服务 Web 客户端（MVP阶段）

---

## 1. 文档编写规范

### 1.1 文档层次

客户端采用两层文档体系：

**第一层：架构设计**
- 定位：项目的"宪法"，定义技术栈、页面结构、路由设计
- 文件：`Client-Base-Design.md`
- 稳定性：高度稳定，变更需评审

**第二层：页面与API设计**
- 定位：具体页面功能、交互流程、API对接
- 文件：各页面设计文档、API类型定义
- 稳定性：随需求调整，但需保持与后端契约一致

### 1.2 文档必备章节

每个设计文档必须包含：
- 版本历史
- 功能描述
- 与后端接口对齐说明
- Mock数据方案（开发阶段）

### 1.3 Markdown规范

- 使用中文撰写
- 代码块必须标注语言类型
- 使用Mermaid绘制流程图
- 表格必须对齐

---

## 2. 代码规范

### 2.1 技术栈

- **框架**: Vue 3（Composition API）
- **语言**: TypeScript 5.0+
- **UI库**: Element Plus
- **状态管理**: Pinia（可选，MVP阶段可用组合式API + localStorage）
- **HTTP客户端**: Axios
- **构建工具**: Vite

### 2.2 文件命名规范

```
src/
├── views/              # 页面组件（PascalCase）
│   ├── SettingsView.vue
│   ├── UploadView.vue
│   └── TaskListView.vue
├── components/         # 公共组件（PascalCase）
│   ├── TaskCard.vue
│   └── UploadProgress.vue
├── api/               # API接口（kebab-case）
│   ├── settings-api.ts
│   ├── task-api.ts
│   └── types.ts
├── utils/             # 工具函数（kebab-case）
│   ├── http-client.ts
│   └── mock-adapter.ts
└── router/            # 路由配置
    └── index.ts
```

### 2.3 命名规范

**变量/函数**: camelCase
```typescript
const taskList = ref<Task[]>([])
const getTaskStatus = async (taskId: string) => {}
```

**类型/接口**: PascalCase
```typescript
interface GetSettingsResponse {
  version: number
  is_configured: boolean
}
```

**常量**: UPPER_SNAKE_CASE
```typescript
const MAX_UPLOAD_SIZE_MB = 2048
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL
```

**组件**: PascalCase
```vue
<script setup lang="ts">
// SettingsView.vue
</script>
```

### 2.4 TypeScript规范

**必须显式声明类型**
```typescript
// ✅ 正确
const taskId: string = 'abc123'
const status: TaskStatus = 'PENDING'

// ❌ 错误
const taskId = 'abc123'  // 隐式any
```

**使用接口而非类型别名（对象结构）**
```typescript
// ✅ 正确
interface Task {
  task_id: string
  status: TaskStatus
}

// ❌ 避免
type Task = {
  task_id: string
  status: TaskStatus
}
```

**枚举使用const enum（编译优化）**
```typescript
const enum TaskStatus {
  PENDING = 'PENDING',
  PROCESSING = 'PROCESSING',
  COMPLETED = 'COMPLETED',
  FAILED = 'FAILED'
}
```

### 2.5 Vue 3 Composition API规范

**使用`<script setup>`语法**
```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'

const count = ref(0)
const increment = () => count.value++

onMounted(() => {
  console.log('Component mounted')
})
</script>
```

**Props定义使用defineProps**
```vue
<script setup lang="ts">
interface Props {
  taskId: string
  status: TaskStatus
}

const props = defineProps<Props>()
</script>
```

**Emits定义使用defineEmits**
```vue
<script setup lang="ts">
interface Emits {
  (e: 'update', value: string): void
  (e: 'delete', id: string): void
}

const emit = defineEmits<Emits>()
</script>
```

---

## 3. ESLint + Prettier配置

### 3.1 ESLint规则（.eslintrc.cjs）

```javascript
module.exports = {
  root: true,
  env: {
    browser: true,
    es2021: true,
    node: true
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended',
    'plugin:@typescript-eslint/recommended',
    '@vue/typescript/recommended',
    '@vue/prettier'
  ],
  parserOptions: {
    ecmaVersion: 2021
  },
  rules: {
    'no-console': process.env.NODE_ENV === 'production' ? 'warn' : 'off',
    'no-debugger': process.env.NODE_ENV === 'production' ? 'warn' : 'off',
    '@typescript-eslint/no-explicit-any': 'error',
    'vue/multi-word-component-names': 'off'
  }
}
```

### 3.2 Prettier规则（.prettierrc.json）

```json
{
  "semi": false,
  "singleQuote": true,
  "trailingComma": "none",
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false,
  "arrowParens": "avoid"
}
```

---

## 4. Git提交规范

### 4.1 Commit Message格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 4.2 Type类型

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档修改
- `style`: 代码格式修改（不影响逻辑）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

### 4.3 Scope范围

- `settings`: 配置管理页面
- `upload`: 上传页面
- `task-list`: 任务列表页面
- `api`: API接口层
- `mock`: Mock数据
- `router`: 路由
- `build`: 构建配置

### 4.4 示例

```bash
# 功能开发
feat(settings): 实现配置管理页面基础布局

# Bug修复
fix(api): 修复上传文件时Content-Type错误

# 文档更新
docs(api): 补充API类型定义注释

# Mock数据
chore(mock): 新增任务状态查询Mock数据
```

---

## 5. API对接规范

### 5.1 接口对齐原则

- 每个API调用必须在注释中标注对应的后端接口
- TypeScript类型定义必须与后端契约一致
- 字段命名使用snake_case（与后端保持一致）

### 5.2 示例

```typescript
/**
 * 获取应用配置
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第5章
 */
export const getSettings = async (): Promise<GetSettingsResponse> => {
  const response = await httpClient.get<GetSettingsResponse>('/v1/settings')
  return response.data
}
```

### 5.3 错误处理规范

```typescript
try {
  const data = await getSettings()
  // 处理成功响应
} catch (error) {
  if (axios.isAxiosError(error)) {
    // HTTP错误
    const status = error.response?.status
    const message = error.response?.data?.message || '请求失败'
    ElMessage.error(message)
  } else {
    // 其他错误
    ElMessage.error('未知错误')
  }
}
```

---

## 6. Mock数据规范

### 6.1 Mock方案选择

**MVP阶段推荐**: axios-mock-adapter（简单易用）

**生产环境推荐**: MSW（Service Worker拦截，更真实）

### 6.2 Mock数据文件结构

```
src/mock/
├── index.ts           # Mock入口
├── settings.ts        # 配置管理Mock
├── task.ts            # 任务管理Mock
└── data/              # Mock数据
    ├── settings.json
    └── tasks.json
```

### 6.3 环境变量控制

```typescript
// .env.development
VITE_USE_MOCK=true
VITE_API_BASE_URL=http://localhost:8080

// .env.production
VITE_USE_MOCK=false
VITE_API_BASE_URL=https://api.production.com
```

```typescript
// main.ts
if (import.meta.env.VITE_USE_MOCK === 'true') {
  const { setupMock } = await import('./mock')
  setupMock()
}
```

---

## 7. 测试规范（可选，MVP阶段）

### 7.1 单元测试

使用Vitest + Vue Test Utils

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskCard from '@/components/TaskCard.vue'

describe('TaskCard', () => {
  it('renders task status correctly', () => {
    const wrapper = mount(TaskCard, {
      props: {
        taskId: 'abc123',
        status: 'PENDING'
      }
    })
    expect(wrapper.text()).toContain('排队中')
  })
})
```

### 7.2 E2E测试（可选）

使用Playwright或Cypress

---

## 8. 文档变更历史

| 版本 | 日期       | 变更内容   |
| ---- | ---------- | ---------- |
| 1.0  | 2025-11-03 | 初始版本   |

---

**文档结束**
