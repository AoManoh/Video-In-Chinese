# 客户端文档总览

**创建日期**: 2025-11-03
**文档版本**: 1.0

---

## 文档体系

客户端采用**两层文档体系**，从宏观架构到具体页面设计：

```
notes/client/
├── 1st/                                  # 第一层：架构设计
│   ├── Client-Base-Design.md v1.0        # 客户端架构设计文档
│   ├── design-rules.md v1.0              # 设计规范文档
│   └── 架构设计.txt
└── 2nd/                                  # 第二层：页面与API设计
    ├── API-Types.md v1.0                 # API接口类型定义（核心）
    ├── Settings-Page-Design.md v1.0      # 配置管理页面设计
    ├── Upload-Page-Design.md v1.0        # 任务上传页面设计
    ├── TaskList-Page-Design.md v1.0      # 任务列表页面设计
    └── 页面设计.txt
```

---

## 文档层次说明

### 第一层：架构设计

**定位**：项目的"宪法"，定义技术栈、项目结构、与后端对接规范

**核心文档**:
- `Client-Base-Design.md`：技术栈选型、工程结构、路由设计、Mock数据方案
- `design-rules.md`：代码规范、命名规范、Git提交规范

**稳定性**：高度稳定，变更需评审

### 第二层：页面与API设计

**定位**：开发人员的直接蓝图，包含页面功能、交互流程、API对接

**核心文档**:
- `API-Types.md`：**最重要**，完整的TypeScript类型定义，与后端接口完全对齐
- `Settings-Page-Design.md`：配置管理页面（20+个配置项）
- `Upload-Page-Design.md`：任务上传页面（文件验证、上传进度）
- `TaskList-Page-Design.md`：任务列表页面（轮询策略、下载功能）

**稳定性**：随需求调整，但需保持与后端契约一致

---

## 与后端文档的对应关系

| 客户端文档 | 对齐的后端文档 | 对齐内容 |
|-----------|---------------|---------|
| `Client-Base-Design.md` | `Base-Design.md` v2.2 | 系统架构、技术选型、MVP范围 |
| `API-Types.md` | `Gateway-design.md` v5.9 第5章 | API接口定义、Request/Response类型 |
| `Settings-Page-Design.md` | `Gateway-design.md` v5.9 第6.1-6.2章 | GET/POST /v1/settings逻辑 |
| `Upload-Page-Design.md` | `Gateway-design.md` v5.9 第6.3章 | POST /v1/tasks/upload逻辑 |
| `TaskList-Page-Design.md` | `Gateway-design.md` v5.9 第6.4-6.5章 | 任务状态查询、文件下载逻辑 |

---

## 核心特性

### 1. 接口对齐清晰

每个API调用都在注释中标注对应的后端接口：

```typescript
/**
 * 获取应用配置
 * 
 * @backend GET /v1/settings
 * @reference Gateway-design.md v5.9 第276-277行
 */
export const getSettings = async (): Promise<GetSettingsResponse> => {
  // ...
}
```

### 2. Mock数据完整

提供所有接口的Mock数据示例，便于前端独立开发：

```json
// mock/data/settings.json
{
  "version": 1,
  "is_configured": true,
  "asr_provider": "openai-whisper",
  "asr_api_key": "sk-proj-***-xyz789",
  ...
}
```

### 3. 环境切换简单

通过环境变量控制Mock/真实后端：

```bash
# .env.development (开发阶段，使用Mock)
VITE_USE_MOCK=true

# .env.production (后端实现后)
VITE_USE_MOCK=false
```

### 4. 类型安全

完整的TypeScript类型定义，减少运行时错误：

```typescript
interface GetSettingsResponse {
  version: number
  is_configured: boolean
  asr_provider: string
  // ... 20+个字段
}
```

---

## 后端实现后的接口对齐流程

### 步骤1：关闭Mock数据

```bash
# .env.development
VITE_USE_MOCK=false
VITE_API_BASE_URL=http://localhost:8080
```

### 步骤2：逐个接口验证

按照 `API-Types.md` 中的接口对齐映射表，逐个验证：

- [ ] GET /v1/settings
  - 检查响应字段数量和类型
  - 验证is_configured逻辑
  - 验证API Key脱敏格式
  
- [ ] POST /v1/settings
  - 验证乐观锁机制（版本号冲突）
  - 验证API Key更新逻辑（包含***的字段不更新）
  
- [ ] POST /v1/tasks/upload
  - 验证文件大小限制（2048MB）
  - 验证MIME Type检测
  - 验证磁盘空间不足错误（507）
  
- [ ] GET /v1/tasks/:taskId/status
  - 验证状态枚举值（PENDING、PROCESSING、COMPLETED、FAILED）
  - 验证result_url格式
  
- [ ] GET /v1/tasks/download/:taskId/:fileName
  - 验证文件流式传输
  - 验证Range请求支持

### 步骤3：错误处理验证

验证所有HTTP状态码处理：

- [ ] 400 Bad Request
- [ ] 404 Not Found
- [ ] 409 Conflict
- [ ] 413 Payload Too Large
- [ ] 415 Unsupported Media Type
- [ ] 500 Internal Server Error
- [ ] 503 Service Unavailable
- [ ] 507 Insufficient Storage

### 步骤4：删除Mock代码（可选）

```bash
# 后端稳定后，删除Mock相关代码
rm -rf src/mock/
```

---

## 快速开始

### 1. 阅读顺序

初次阅读建议按以下顺序：

1. `design-rules.md`（了解规范）
2. `Client-Base-Design.md`（了解整体架构）
3. `API-Types.md`（了解API接口定义，**最重要**）
4. 各页面设计文档（了解具体功能）

### 2. 开发顺序

建议按以下顺序开发：

1. 搭建项目框架（Vite + Vue 3 + TypeScript）
2. 配置ESLint + Prettier
3. 创建API类型定义（`src/api/types.ts`）
4. 配置Mock数据（`src/mock/`）
5. 开发配置管理页面（`src/views/SettingsView.vue`）
6. 开发任务上传页面（`src/views/UploadView.vue`）
7. 开发任务列表页面（`src/views/TaskListView.vue`）
8. 集成测试（后端实现后）

---

## 文档维护

### 变更通知机制

- **破坏性变更**（API接口签名变更）：立即通知前端团队，召开评审会
- **非破坏性变更**（新增字段）：周例会统一通知

### 版本同步

- 客户端文档版本号独立管理
- 关联后端文档版本号（在文档头部标注）
- 后端接口变更时，同步更新客户端文档

### 文档审查

参考 `design-rules.md` 第1章"文档编写规范"：

- [ ] 文档包含版本历史
- [ ] 文档包含与后端接口对齐说明
- [ ] 代码示例清晰可读
- [ ] Mermaid流程图正确渲染

---

## 附录：技术栈版本要求

| 依赖 | 最低版本 | 推荐版本 | 备注 |
|------|---------|---------|------|
| Node.js | 18.0 | 20.0 | LTS版本 |
| npm | 9.0 | 10.0 | 包管理器 |
| Vue | 3.3 | 3.4 | 组合式API |
| TypeScript | 5.0 | 5.3 | 类型检查 |
| Element Plus | 2.4 | 2.5 | UI组件库 |
| Vite | 5.0 | 5.0 | 构建工具 |

---

**文档结束**
