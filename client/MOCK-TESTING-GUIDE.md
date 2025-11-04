# Mock模式功能测试指南

**文档版本**: 1.0  
**创建日期**: 2025-11-04  
**适用环境**: 前端开发阶段（无需后端）

---

## 📖 目录

1. [Mock模式说明](#1-mock模式说明)
2. [测试环境准备](#2-测试环境准备)
3. [功能测试场景](#3-功能测试场景)
4. [常见问题解答](#4-常见问题解答)

---

## 1. Mock模式说明

### 1.1 什么是Mock模式？

Mock模式是**纯前端的模拟测试环境**，使用 `axios-mock-adapter` 拦截所有HTTP请求并返回预设的模拟数据。

```
┌─────────────────────────────────────────────────────────┐
│  前端应用流程                                           │
│                                                         │
│  用户操作 → 前端发起请求 → Mock拦截器 → 返回模拟数据   │
│                            ↓                            │
│                        不发送真实网络请求               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 1.2 Mock模式做什么？

✅ **测试前端UI交互**
- 表单验证逻辑
- 状态切换和渲染
- 错误提示显示
- 路由跳转逻辑

✅ **测试前端状态管理**
- localStorage持久化
- 任务列表轮询
- 数据更新和同步

✅ **模拟后端响应**
- 成功响应（200）
- 错误响应（400、404、409等）
- 网络延迟（500ms）

### 1.3 Mock模式不做什么？

❌ **不调用真实后端服务**
- 不需要启动Go服务器
- 不需要Redis/数据库
- 不需要配置后端环境

❌ **不调用AI服务商API**
- 不会向OpenAI发送请求
- 不会向Google Gemini发送请求
- 不会向阿里云发送请求
- 不会产生任何API费用

❌ **不验证API Key有效性**
- 可以输入任意字符串
- 不检查格式正确性
- 不验证服务商连通性

---

## 2. 测试环境准备

### 2.1 确认Mock模式已启用

1. 检查环境变量配置：

```bash
# client/.env.development
VITE_USE_MOCK=true
VITE_API_BASE_URL=http://localhost:8080
```

2. 启动开发服务器：

```bash
cd client
npm run dev
```

3. 确认控制台输出：

```
VITE v5.x.x ready in xxx ms

➜  Local:   http://localhost:5173/
➜  Network: use --host to expose

[Mock] Mock数据已启用
[Mock] 使用VITE_USE_MOCK环境变量控制Mock开关
```

### 2.2 打开浏览器开发者工具

按 `F12` 打开开发者工具，切换到 **Console** 标签页，你会看到：

```
[Mock] Mock数据已启用
[Mock] 使用VITE_USE_MOCK环境变量控制Mock开关
```

---

## 3. 功能测试场景

### 场景1：初次访问（未配置状态）

#### 3.1 预期行为

✅ 页面自动跳转到 `/settings`  
✅ 显示初始化向导提示："系统首次启动，请完成基本配置"  
✅ 所有配置字段为空  
✅ 必填字段标记为红色星号

#### 3.2 测试步骤

1. 打开浏览器访问 `http://localhost:5173`
2. 观察页面跳转到 `/settings`
3. 观察控制台输出：

```
[API Request] GET /v1/settings
[Mock] GET /v1/settings
```

4. 检查Mock返回的数据：

```json
{
  "version": 1,
  "is_configured": false,  // 初始未配置
  "asr_provider": "",
  "asr_api_key": "",
  "translation_provider": "",
  "translation_api_key": "",
  "voice_cloning_provider": "",
  "voice_cloning_api_key": "",
  ...
}
```

---

### 场景2：填写配置并保存

#### 3.1 预期行为

✅ 必填字段验证生效  
✅ 保存成功后显示成功提示  
✅ 版本号自动递增（1 → 2）  
✅ `is_configured` 自动变为 `true`  
✅ 初始化向导提示消失

#### 3.2 测试步骤

1. **填写必填项**（可以输入任意字符串）：

```
ASR服务商: openai-whisper
ASR API Key: sk-test-123456  // 任意字符串即可

翻译服务商: google-gemini
翻译 API Key: AIza-test-789  // 任意字符串即可

声音克隆服务商: aliyun-cosyvoice
声音克隆 API Key: LTAI-test-abc  // 任意字符串即可
```

2. **点击"保存配置"按钮**

3. **观察控制台输出**：

```
[API Request] POST /v1/settings
[Mock] POST /v1/settings
```

4. **观察Mock数据更新**：

```json
{
  "version": 2,  // 版本号递增
  "is_configured": true,  // 自动变为true
  "asr_provider": "openai-whisper",
  "asr_api_key": "sk-test-123456",
  ...
}
```

5. **验证UI变化**：
   - ✅ 顶部成功提示："配置已成功更新"
   - ✅ 初始化向导提示消失
   - ✅ 导航菜单的"上传视频"和"任务列表"可点击

---

### 场景3：测试乐观锁机制（版本冲突）

#### 3.1 预期行为

✅ 模拟版本冲突（409错误）  
✅ 显示错误提示："配置已被其他用户修改，请刷新后重试"

#### 3.2 测试步骤

1. 打开浏览器开发者工具 → Console
2. 手动修改Mock数据的版本号：

```javascript
// 在浏览器控制台执行
// 这会导致下次保存时版本号不匹配
```

3. 修改任意配置项并保存
4. 观察控制台输出：

```
[API Request] POST /v1/settings
[Mock] POST /v1/settings
HTTP 409 Conflict
```

5. 观察页面显示：
   - ✅ 错误提示："配置已被其他用户修改，请刷新后重试"

6. 刷新页面重新加载配置

---

### 场景4：上传视频文件

#### 4.1 预期行为

✅ 文件选择/拖拽功能正常  
✅ 文件大小和格式验证生效  
✅ 上传进度条显示（0% → 100%）  
✅ 上传成功后自动跳转到任务列表  
✅ Mock自动生成任务ID

#### 4.2 测试步骤

1. **导航到上传页面**：
   - 点击顶部导航"上传视频"
   - 或直接访问 `http://localhost:5173/upload`

2. **选择文件**（方式1：点击）：
   - 点击"点击选择文件"按钮
   - 选择任意视频文件（MP4、MOV、MKV）

3. **选择文件**（方式2：拖拽）：
   - 将视频文件拖拽到上传区域
   - 观察"拖拽到此处"提示

4. **验证文件信息**：
   - ✅ 显示文件名
   - ✅ 显示文件大小（如：1.5 GB）
   - ✅ 显示预估耗时

5. **测试文件验证**：

```
# 测试格式验证
- 尝试上传非视频文件（如 .txt）
- 应显示错误："仅支持MP4、MOV、MKV格式"

# 测试大小验证
- 尝试上传超过2048MB的文件
- 应显示错误："文件超过2048MB限制"
```

6. **开始上传**：
   - 点击"开始上传"按钮
   - 观察进度条动画（模拟500ms网络延迟）

7. **观察控制台输出**：

```
[API Request] POST /v1/tasks/upload
[Mock] POST /v1/tasks/upload
[Mock] 任务abc12345状态变更: PROCESSING (3秒后)
[Mock] 任务abc12345状态变更: COMPLETED (10秒后)
```

8. **验证自动跳转**：
   - ✅ 上传成功后自动跳转到 `/tasks`
   - ✅ 新任务显示在列表顶部
   - ✅ 新任务有高亮边框

---

### 场景5：任务列表与状态轮询

#### 5.1 预期行为

✅ 显示所有任务列表  
✅ 自动轮询任务状态（指数退避：3s → 6s → 10s）  
✅ 状态自动更新：PENDING → PROCESSING → COMPLETED  
✅ 完成后显示"下载结果"按钮  
✅ localStorage持久化（刷新不丢失）

#### 5.2 测试步骤

1. **观察初始状态**（上传后立即跳转）：
   - 任务状态：**排队中**（PENDING）
   - 状态图标：灰色圆圈

2. **观察状态变化**（3秒后）：
   - 控制台输出：`[Mock] 任务abc12345状态变更: PROCESSING`
   - 任务状态：**处理中**（PROCESSING）
   - 状态图标：橙色旋转动画

3. **观察轮询日志**（每3秒一次，然后6秒，然后10秒）：

```
[API Request] GET /v1/tasks/abc12345/status
[Mock] GET /v1/tasks/:taskId/status abc12345
(3秒后再次请求)
[API Request] GET /v1/tasks/abc12345/status
(6秒后再次请求)
...
```

4. **观察状态变化**（10秒后）：
   - 控制台输出：`[Mock] 任务abc12345状态变更: COMPLETED`
   - 任务状态：**已完成**（COMPLETED）
   - 状态图标：绿色对勾
   - 显示"下载结果"按钮

5. **测试下载功能**：
   - 点击"下载结果"按钮
   - 观察控制台输出：

```
[API Request] GET /v1/tasks/download/abc12345/result.mp4
[Mock] GET /v1/tasks/download/:taskId/:fileName
```

   - 浏览器自动下载一个名为 `result.mp4` 的文件（Mock数据，只有几个字节）

6. **测试localStorage持久化**：
   - 刷新页面（F5）
   - ✅ 任务列表仍然存在
   - ✅ 任务状态保持不变

---

### 场景6：错误场景测试

#### 6.1 测试404错误（任务不存在）

1. 在浏览器地址栏手动访问：

```
http://localhost:5173/tasks
```

2. 在控制台手动触发查询不存在的任务：

```javascript
// 浏览器控制台执行
fetch('/v1/tasks/non-existent-task-id/status')
```

3. 观察Mock返回404错误：

```json
{
  "code": "NOT_FOUND",
  "message": "任务不存在"
}
```

#### 6.2 测试路由守卫

1. 清除localStorage：

```javascript
// 浏览器控制台执行
localStorage.clear()
```

2. 手动访问 `/upload` 或 `/tasks`
3. 预期行为：
   - ✅ 自动跳转到 `/settings`
   - ✅ 显示提示："请先完成基本配置"

---

## 4. 常见问题解答

### Q1: Mock模式需要填写真实的API Key吗？

**答**：❌ **不需要**。Mock模式不会验证API Key，你可以输入任意字符串：

```
✅ 可以输入: sk-test-123
✅ 可以输入: random-string
✅ 可以输入: abc123xyz
❌ 不会验证格式
❌ 不会调用API
❌ 不会产生费用
```

### Q2: Mock模式会调用真实的AI服务吗？

**答**：❌ **不会**。所有请求都被 `axios-mock-adapter` 拦截：

```
前端请求 → Mock拦截器 → 返回模拟数据
           ↓
      不发送网络请求
```

**验证方法**：
- 打开浏览器开发者工具 → Network标签页
- 你会发现**没有任何真实的网络请求**
- 所有请求都是 `(from disk cache)` 或 `(mock)`

### Q3: Mock的任务状态是如何变化的？

**答**：Mock使用 `setTimeout` 模拟状态变化：

```javascript
// 上传后
taskStatus = 'PENDING'

// 3秒后
setTimeout(() => {
  taskStatus = 'PROCESSING'
}, 3000)

// 10秒后
setTimeout(() => {
  taskStatus = 'COMPLETED'
}, 10000)
```

这是**纯前端的定时器**，不涉及任何后端处理。

### Q4: Mock模式下可以测试什么？

**可以测试**：
- ✅ UI渲染和布局
- ✅ 表单验证逻辑
- ✅ 状态切换动画
- ✅ 错误提示显示
- ✅ 路由跳转逻辑
- ✅ localStorage持久化
- ✅ 轮询机制
- ✅ 文件验证逻辑（大小、格式）

**无法测试**：
- ❌ API Key有效性
- ❌ 真实的视频处理
- ❌ AI服务商响应时间
- ❌ 服务器性能
- ❌ 网络传输速度

### Q5: 如何切换到真实后端？

**答**：修改一行环境变量：

```bash
# client/.env.development
VITE_USE_MOCK=false  # 改为false
VITE_API_BASE_URL=http://localhost:8080
```

然后重启开发服务器：

```bash
npm run dev
```

### Q6: Mock模式的上传进度是真实的吗？

**答**：❌ **不是**。Mock模式的上传进度是**模拟的**：

```javascript
// Mock使用延迟返回响应
mock.onPost('/v1/tasks/upload').reply(config => {
  return [200, { task_id: 'xxx' }]
}, { delayResponse: 500 })  // 模拟500ms延迟
```

真实的上传进度需要等后端实现后才能测试。

### Q7: 如何验证Mock模式正在运行？

**答**：查看控制台输出：

```
✅ 应该看到:
[Mock] Mock数据已启用
[Mock] GET /v1/settings
[Mock] POST /v1/settings
[Mock] POST /v1/tasks/upload

❌ 如果没有[Mock]前缀，说明Mock未启用
```

### Q8: Mock数据会保存吗？

**答**：部分保存：
- ✅ Mock的配置数据在**内存中**，刷新页面会重置
- ✅ 任务列表保存在**localStorage**，刷新页面不丢失
- ❌ 关闭浏览器后，Mock内存数据丢失

---

## 5. 完整测试流程总结

### 5.1 快速测试流程（5分钟）

```bash
# 1. 启动Mock模式
cd client
npm run dev

# 2. 访问应用
浏览器打开 http://localhost:5173

# 3. 填写配置（任意字符串）
ASR API Key: test-123
Translation API Key: test-456
Voice Cloning API Key: test-789

# 4. 上传文件
选择任意视频文件 → 开始上传

# 5. 观察任务列表
等待10秒 → 状态变为"已完成" → 下载结果
```

### 5.2 完整测试流程（30分钟）

1. **配置管理测试** (10分钟)
   - [ ] 初次访问（未配置状态）
   - [ ] 填写必填项并保存
   - [ ] 修改配置并保存
   - [ ] 测试乐观锁冲突
   - [ ] 测试表单验证

2. **文件上传测试** (10分钟)
   - [ ] 点击选择文件
   - [ ] 拖拽选择文件
   - [ ] 测试格式验证（上传非视频文件）
   - [ ] 测试大小验证（上传超大文件）
   - [ ] 观察上传进度
   - [ ] 验证自动跳转

3. **任务列表测试** (10分钟)
   - [ ] 观察初始状态（PENDING）
   - [ ] 观察状态变化（PROCESSING → COMPLETED）
   - [ ] 测试下载功能
   - [ ] 测试localStorage持久化
   - [ ] 测试轮询机制（查看控制台日志）
   - [ ] 刷新页面验证数据保留

---

## 6. 下一步：真实后端联调

Mock测试完成后，等待后端Gateway服务实现，然后：

1. 修改环境变量：`VITE_USE_MOCK=false`
2. 启动后端服务：`go run main.go`
3. 重新测试所有功能
4. 这次会调用真实的AI服务API（需要有效的API Key）

---

**文档结束**

如有问题，请参考：
- `client/DEVELOPMENT.md` - 开发指南
- `client/TESTING.md` - 测试指南
- `notes/client/README.md` - 文档总览

