# Gateway Service

Gateway 服务是 video-In-Chinese 项目的 HTTP API 网关，提供 RESTful API 接口供前端调用。

## 服务概述

- **服务名称**: Gateway
- **服务端口**: 8080
- **技术栈**: Go 1.24+, GoZero 框架, RESTful API, gRPC Client
- **依赖服务**: Redis, Task 服务
- **参考文档**: `notes/server/2nd/Gateway-design.md` v5.8

## 功能特性

### 1. 配置管理
- 获取应用配置（GET /v1/settings）
- 更新应用配置（POST /v1/settings）
- API Key 加密存储（AES-256-GCM）
- API Key 脱敏显示（前缀-***-后6位）
- 乐观锁并发控制（Redis Lua 脚本）

### 2. 任务管理
- 上传视频文件（POST /v1/tasks/upload）
- 查询任务状态（GET /v1/tasks/:taskId/status）
- 下载处理结果（GET /v1/tasks/download/:taskId/:fileName）

### 3. 安全特性
- 路径遍历攻击防护
- 符号链接攻击防护
- MIME Type 白名单验证
- 磁盘空间预检（fileSize * 3 + 500MB）
- 流式文件上传/下载（不占用大量内存）

### 4. 跨平台支持
- Unix/Linux: 使用 syscall.Statfs 检查磁盘空间
- Windows: 使用 kernel32.dll GetDiskFreeSpaceExW 检查磁盘空间

## API 接口

### 1. GET /v1/settings
获取当前的应用配置信息。

**响应示例**:
```json
{
  "version": 1,
  "is_configured": true,
  "asr_vendor": "aliyun",
  "asr_api_key": "sk-abc***-xyz123",
  "translation_vendor": "google",
  "translation_api_key": "AIza***-xyz123",
  "voice_cloning_vendor": "aliyun_cosyvoice",
  "voice_cloning_api_key": "sk-def***-xyz123",
  "enable_audio_separation": true,
  "enable_text_polishing": false,
  "enable_translation_optimization": true
}
```

### 2. POST /v1/settings
更新应用配置（支持乐观锁）。

**请求示例**:
```json
{
  "version": 1,
  "asr_vendor": "aliyun",
  "asr_api_key": "sk-new-key-123456",
  "translation_vendor": "google",
  "translation_api_key": "sk-abc***-xyz123",
  "voice_cloning_vendor": "aliyun_cosyvoice",
  "voice_cloning_api_key": "sk-def***-xyz123",
  "enable_audio_separation": true,
  "enable_text_polishing": false,
  "enable_translation_optimization": true
}
```

**响应示例**:
```json
{
  "version": 2,
  "message": "Settings updated successfully"
}
```

### 3. POST /v1/tasks/upload
上传视频文件并创建处理任务。

**请求**: multipart/form-data
- `file`: 视频文件（最大 2GB）

**响应示例**:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 4. GET /v1/tasks/:taskId/status
查询任务处理状态。

**响应示例**:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "result_url": "/v1/tasks/download/550e8400-e29b-41d4-a716-446655440000/result.mp4",
  "error_message": ""
}
```

### 5. GET /v1/tasks/download/:taskId/:fileName
下载处理结果文件。

**响应**: 文件流（Content-Type: video/mp4）

## 环境变量配置

```bash
# Redis 配置
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Task 服务 gRPC 配置
TASK_RPC_ENDPOINTS=127.0.0.1:50050

# 文件存储路径
TEMP_STORAGE_PATH=./data/temp
LOCAL_STORAGE_PATH=../../data/videos  # keep in sync with processor/task services

# 上传限制
MAX_UPLOAD_SIZE_MB=2048

# MIME Type 白名单
SUPPORTED_MIME_TYPES=video/mp4,video/quicktime,video/x-matroska,video/x-msvideo

# API Key 加密密钥（32字节）
API_KEY_ENCRYPTION_SECRET=your-32-byte-secret-key-here-12

# HTTP 配置
HTTP_TIMEOUT_SECONDS=300
MAX_CONCURRENT_CONNECTIONS=100
```

## 项目结构

```
server/app/gateway/
├── etc/
│   └── gateway-api.yaml          # 配置文件
├── internal/
│   ├── config/
│   │   └── config.go             # 配置结构体
│   ├── handler/
│   │   ├── settings/
│   │   │   ├── get_settings_handler.go
│   │   │   └── update_settings_handler.go
│   │   └── task/
│   │       ├── upload_task_handler.go
│   │       ├── get_task_status_handler.go
│   │       └── download_file_handler.go
│   ├── logic/
│   │   ├── settings/
│   │   │   ├── get_settings_logic.go      # 6步流程
│   │   │   └── update_settings_logic.go   # 5步流程
│   │   └── task/
│   │       ├── upload_task_logic.go       # 9步流程
│   │       ├── get_task_status_logic.go   # 5步流程
│   │       └── download_file_logic.go     # 6步流程
│   ├── svc/
│   │   └── service_context.go    # 服务上下文（Redis + Task gRPC）
│   ├── types/
│   │   └── types.go              # API 类型定义
│   └── utils/
│       ├── crypto.go             # API Key 加密/解密
│       ├── lua_scripts.go        # Redis Lua 脚本
│       ├── disk.go               # 磁盘空间检查（Unix）
│       ├── disk_windows.go       # 磁盘空间检查（Windows）
│       ├── path.go               # 路径安全检查
│       └── mime.go               # MIME Type 检测
├── gateway.api                   # API 定义文件
├── gateway.go                    # 服务入口
└── README.md                     # 本文档
```

## 编译和运行

### 编译
```bash
cd server/app/gateway
go build -o gateway.exe .
```

### 运行
```bash
./gateway.exe -f etc/gateway-api.yaml
```

### 开发模式
```bash
go run gateway.go -f etc/gateway-api.yaml
```

## 依赖服务

### 1. Redis
- 用途: 存储应用配置（app:settings）
- 端口: 6379
- 启动命令: `redis-server`

### 2. Task 服务
- 用途: 任务管理（创建任务、查询状态）
- 端口: 50050
- 启动命令: `cd server/mcp/task && go run task.go`

## 开发规范

### 1. 代码风格
- 使用 gofmt 格式化代码
- 使用 go vet 静态分析
- 使用 golangci-lint 代码质量检查

### 2. 错误处理
- 所有错误必须记录日志（logx.Errorf）
- 返回用户友好的错误消息
- 避免暴露内部实现细节

### 3. 日志记录
- 使用 logx.Infof 记录关键操作
- 使用 logx.Errorf 记录错误信息
- 日志格式: `[FunctionName] 描述: 详细信息`

### 4. 安全性
- 所有文件路径必须经过安全检查
- 所有 API Key 必须加密存储
- 所有文件上传必须验证 MIME Type

## 测试

### 单元测试
```bash
go test ./internal/logic/... -v
go test ./internal/utils/... -v
```

### 集成测试
```bash
go test ./... -v
```

### 覆盖率报告
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 故障排查

### 1. Redis 连接失败
- 检查 Redis 是否启动: `redis-cli ping`
- 检查配置文件中的 Redis 地址和端口
- 检查防火墙设置

### 2. Task 服务连接失败
- 检查 Task 服务是否启动
- 检查配置文件中的 Task RPC 端点
- 检查网络连接

### 3. 文件上传失败
- 检查磁盘空间是否充足
- 检查文件大小是否超过限制（2GB）
- 检查 MIME Type 是否在白名单中
- 检查临时目录是否存在且可写

### 4. 文件下载失败
- 检查文件是否存在
- 检查文件路径是否安全
- 检查文件权限

## 性能优化

### 1. 流式文件处理
- 使用 io.Copy 流式上传/下载
- 避免将整个文件加载到内存
- 适合处理大文件（2GB）

### 2. 并发控制
- 使用 MaxConcurrentConnections 限制并发连接数
- 避免资源耗尽

### 3. 超时控制
- 使用 HttpTimeoutSeconds 设置 HTTP 超时
- 避免长时间阻塞

## 版本历史

- **v1.0.0** (2025-11-05): 初始版本
  - 实现配置管理（getSettings, updateSettings）
  - 实现任务管理（uploadTask, getTaskStatus, downloadFile）
  - 实现工具函数（crypto, disk, path, mime）
  - 编译验证通过

## 许可证

MIT License

## 联系方式

- 项目地址: https://github.com/your-org/video-In-Chinese
- 问题反馈: https://github.com/your-org/video-In-Chinese/issues
