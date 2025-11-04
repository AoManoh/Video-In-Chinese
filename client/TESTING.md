# 前端集成测试指南

## 启动开发服务器

```bash
cd client
npm run dev
```

访问：http://localhost:5173

## Mock模式验证

确保`.env.development`包含：
```bash
VITE_USE_MOCK=true
VITE_API_BASE_URL=http://localhost:8080
```

浏览器控制台应显示：`[Mock] Mock数据已启用`

---

## 测试清单

### 1. 配置管理页面（/settings）

- [ ] 页面加载成功，显示初始化向导（is_configured=false）
- [ ] 所有表单字段正确显示（20+个配置项）
- [ ] 必填项验证（ASR、Translation、VoiceCloning）
- [ ] 填写必填项并保存
- [ ] 验证version从1递增到2
- [ ] 验证is_configured变为true
- [ ] 测试乐观锁：
  - 打开两个浏览器标签
  - 同时修改配置
  - 验证后保存的会收到409错误并重新加载

### 2. 任务上传页面（/upload）

- [ ] 从/settings保存后自动跳转到/upload（或手动访问）
- [ ] 拖拽上传区域正常显示
- [ ] 选择测试视频文件（任意视频文件）
- [ ] 文件大小验证：
  - 选择<2048MB的文件：通过
  - （可选）选择>2048MB的文件：显示错误
- [ ] 文件格式验证：
  - 选择MP4/MOV/MKV：通过
  - 选择其他格式：显示错误
- [ ] 点击"开始上传"
- [ ] 验证上传进度显示（0% → 100%）
- [ ] 验证上传速度显示（KB/s或MB/s）
- [ ] 上传成功后显示"上传成功"
- [ ] 3秒后自动跳转到/tasks

### 3. 任务列表页面（/tasks）

- [ ] 页面显示新上传的任务（高亮显示）
- [ ] 任务状态初始为"PENDING"（排队中）
- [ ] 观察控制台日志：
  - 3秒后显示：`[Mock] 任务xxx状态变更: PROCESSING`
  - 10秒后显示：`[Mock] 任务xxx状态变更: COMPLETED`
- [ ] 验证轮询间隔（控制台日志）：
  - 初始3秒
  - 第二次6秒
  - 第三次10秒（最大）
- [ ] 任务完成后：
  - 显示桌面通知（如果授权）
  - 停止轮询
  - 显示"下载结果"按钮
- [ ] 点击"下载结果"按钮
- [ ] 验证浏览器触发文件下载（result.mp4）
- [ ] 刷新页面，验证任务列表仍然存在（localStorage持久化）

### 4. 页面跳转流程

- [ ] 访问 / → 自动跳转到/settings
- [ ] 未配置时访问/upload → 跳转到/settings并显示警告
- [ ] 未配置时访问/tasks → 跳转到/settings并显示警告
- [ ] 配置完成后访问/upload → 正常显示
- [ ] 配置完成后访问/tasks → 正常显示

### 5. 错误处理验证

修改Mock数据返回状态码，测试错误处理：

在`src/mock/settings.ts`中临时修改：
```typescript
mock.onGet('/v1/settings').reply(() => [503, { message: '服务不可用' }])
```

验证：
- [ ] 显示错误消息："服务暂时不可用"
- [ ] 页面不崩溃

### 6. localStorage验证

- [ ] 打开浏览器开发者工具 → Application → Local Storage
- [ ] 验证存储的键：
  - `is_configured`
  - `task_list`
- [ ] 验证任务列表最多50个
- [ ] 手动添加7天前的任务（修改created_at），刷新页面验证自动清理

---

## 后端对接准备

### 1. 切换到真实后端

修改`.env.development`：
```bash
VITE_USE_MOCK=false
VITE_API_BASE_URL=http://localhost:8080
```

重启开发服务器：
```bash
npm run dev
```

### 2. 接口对齐检查（参考API-Types.md第9章）

- [ ] GET /v1/settings - 字段完全一致
- [ ] POST /v1/settings - 乐观锁机制正常
- [ ] POST /v1/tasks/upload - 文件上传正常
- [ ] GET /v1/tasks/:taskId/status - 状态枚举一致
- [ ] GET /v1/tasks/download/:taskId/:fileName - 文件下载正常

### 3. HTTP状态码验证

- [ ] 400 Bad Request
- [ ] 404 Not Found
- [ ] 409 Conflict
- [ ] 413 Payload Too Large
- [ ] 415 Unsupported Media Type
- [ ] 500 Internal Server Error
- [ ] 503 Service Unavailable
- [ ] 507 Insufficient Storage

---

## 已知问题

1. **Vue文件ESLint检查**：由于Vue插件兼容性问题，暂时只检查.ts文件，Vue文件由vue-tsc检查
2. **取消上传功能**：UploadView.vue中的cancelUpload标记为TODO，需要实现AbortController

---

**测试完成后，前端即可投入使用（Mock模式），等待后端实现后切换到真实API**

