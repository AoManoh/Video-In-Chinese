# Video-In-Chinese

AI 驱动的视频翻译和配音系统

---

## 快速开始

### 新同事入门

如果您是第一次接触本项目，请按以下步骤操作：

1. **阅读快速启动参考卡** ⭐ 推荐
   - 📄 [docs/QUICK_START_REFERENCE.md](docs/QUICK_START_REFERENCE.md)
   - 最精简的启动指南，5-10 分钟即可上手

2. **启动项目**
   - 按照快速参考卡中的步骤启动 4 个后端服务
   - 通过前端界面配置 AI 服务商

3. **遇到问题？**
   - 📄 [docs/TROUBLESHOOTING_GUIDE.md](docs/TROUBLESHOOTING_GUIDE.md)
   - 包含常见错误及解决方案

### 文档导航

| 文档 | 适用场景 | 阅读时间 |
|------|---------|---------|
| [快速启动参考卡](docs/QUICK_START_REFERENCE.md) | 日常启动、快速查阅 | 5-10 分钟 |
| [项目启动指南](docs/PROJECT_STARTUP_GUIDE.md) | 首次启动、深入了解 | 15-20 分钟 |
| [AudioSeparator服务说明](docs/AudioSeparator服务说明.md) | 音频分离服务（Demucs） | 10 分钟 |
| [虚拟环境子进程分析](docs/VENV_SUBPROCESS_ANALYSIS.md) | Windows venv 幽灵进程 现象 | 10 分钟 |
| [故障排查指南](docs/TROUBLESHOOTING_GUIDE.md) | 遇到错误、性能问题 | 20-30 分钟 |
| [Spleeter音频分离失败问题诊断报告](docs/Spleeter音频分离失败问题诊断报告.md) | 音频分离失效 | 15 分钟 |
| [Demucs技术升级方案评估](docs/Demucs技术升级方案评估.md) | 技术升级评估 | 20 分钟 |
| [建议评估报告](docs/建议评估报告.md) | 技术建议准确性评估 | 10 分钟 |
| [文档中心](docs/README.md) | 查找特定文档 | - |

### 功能特性

- 🎬 视频上传和处理
- 🎙️ 音频分离和语音识别（ASR）
- 🌐 多语言翻译
- ✨ AI 驱动的译文优化
- 🗣️ 声音克隆和配音
- 🔄 重试和降级机制

### 技术栈

**后端**:
- Go 1.21+
- Go-Zero 框架
- gRPC 通信
- Redis 存储
- Python 3.11 (AudioSeparator)
- PyTorch 2.9+ (Demucs 引擎)

**前端**:
- (根据实际情况填写)

**基础设施**:
- Docker (Redis)
- 阿里云 OSS (可选)

### 系统架构

```
前端 (5173)
    ↓ HTTP
Gateway (8080) ← API 网关、配置管理
    ↓ gRPC
Task Service (50050) ← 任务管理
    ↓ gRPC
Processor ← 任务编排
    ├─ gRPC → AudioSeparator (50052) ← 音频分离 (Demucs)
    └─ gRPC → AI Adaptor (50053) ← AI 服务适配器
    ↓ HTTP/gRPC
外部 AI 服务 (ASR, 翻译, 声音克隆)
```

---

## 快速启动

### 前置条件

- Go 1.21+
- Node.js 18+
- Docker Desktop
- PowerShell

### 启动步骤（简化版）

1. **启动 Redis**
```powershell
docker start redis-test
```

2. **清空配置缓存**
```powershell
docker exec redis-test redis-cli DEL app:settings
```

3. **启动后端服务**（5 个独立终端，按顺序启动）

终端 1 - Task Service:
```powershell
cd server/mcp/task
go run . -f etc/task.yaml
```

终端 2 - AudioSeparator (可选但推荐):
```powershell
cd server/mcp/audio_separator
D:\Go-Project\video-In-Chinese\server\.venv\Scripts\python.exe main.py
```

终端 3 - AI Adaptor:
```powershell
cd server/mcp/ai_adaptor
# 设置环境变量（见快速参考卡）
go run . -f etc/ai_adaptor.yaml
```

终端 4 - Processor:
```powershell
cd server/mcp/processor
go run . -f etc/processor.yaml
```

终端 5 - Gateway:
```powershell
cd server/app/gateway
go run . -f etc/gateway-api.yaml
```

4. **配置前端**
   - 访问 `http://localhost:5173`
   - 进入设置页面
   - 填写 AI 服务商配置
   - 保存配置

**详细步骤请参考**: [docs/QUICK_START_REFERENCE.md](docs/QUICK_START_REFERENCE.md)

---

## 常见问题

### Q: 启动时提示端口被占用？

**A**: 检查并终止占用端口的进程
```powershell
netstat -ano | findstr :50050
taskkill /F /PID [进程ID]
```

### Q: Redis 连接失败？

**A**: 确保 AI Adaptor 设置了正确的环境变量
```powershell
$env:REDIS_HOST="127.0.0.1"
```

### Q: 中文显示乱码？

**A**: 在每个终端启动前设置 UTF-8
```powershell
[Console]::InputEncoding = [Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
chcp 65001 > $null
```

**更多问题**: [docs/TROUBLESHOOTING_GUIDE.md](docs/TROUBLESHOOTING_GUIDE.md)

---

## 项目结构

```
video-In-Chinese/
├── server/                 # 后端服务
│   ├── app/
│   │   └── gateway/       # API 网关 (8080)
│   └── mcp/
│       ├── task/          # 任务服务 (50050)
│       ├── ai_adaptor/    # AI 适配器 (50053)
│       └── processor/     # 处理器
├── client/                # 前端应用
├── docs/                  # 项目文档 ⭐
│   ├── README.md         # 文档中心
│   ├── QUICK_START_REFERENCE.md  # 快速参考
│   ├── PROJECT_STARTUP_GUIDE.md  # 启动指南
│   └── TROUBLESHOOTING_GUIDE.md  # 故障排查
├── data/                  # 数据目录
└── README.md             # 本文件
```

---

## 开发指南

### 环境变量

AI Adaptor 需要以下环境变量：

```powershell
# Redis 配置
$env:REDIS_HOST="127.0.0.1"
$env:REDIS_PORT="6379"
$env:REDIS_PASSWORD=""
$env:REDIS_DB="0"

# 加密密钥
$env:API_KEY_ENCRYPTION_SECRET="YOUR_ENCRYPTION_SECRET"

# 阿里云 OSS（可选）
$env:ALIYUN_OSS_ACCESS_KEY_ID="..."
$env:ALIYUN_OSS_ACCESS_KEY_SECRET="..."
$env:ALIYUN_OSS_BUCKET_NAME="..."
$env:ALIYUN_OSS_ENDPOINT="..."
```

### 配置文件

| 服务 | 配置文件 |
|------|---------|
| Gateway | `server/app/gateway/etc/gateway-api.yaml` |
| Task Service | `server/mcp/task/etc/task.yaml` |
| AI Adaptor | `server/mcp/ai_adaptor/etc/ai_adaptor.yaml` |
| Processor | `server/mcp/processor/etc/processor.yaml` |

### 服务端口

| 服务 | 端口 | 协议 |
|------|------|------|
| Gateway | 8080 | HTTP |
| Task Service | 50050 | gRPC |
| AudioSeparator | 50052 | gRPC |
| AI Adaptor | 50053 | gRPC |
| Processor | - | 内部 |

---

## 测试

### 健康检查

```powershell
# 检查所有服务
tasklist | findstr "task.exe ai_adaptor.exe gateway.exe processor.exe"

# 检查端口
netstat -ano | findstr "50050 50053 8080"

# 测试 Gateway API
Invoke-WebRequest -Uri "http://localhost:8080/v1/settings" -Method GET
```

### 端到端测试

1. 上传视频文件
2. 观察任务状态变化
3. 检查各处理步骤：
   - 音频分离
   - ASR 语音识别
   - 翻译
   - 译文优化
   - 声音克隆

---

## 贡献指南

### 代码规范

- Go 代码遵循 Go 官方规范
- 提交前运行 `go fmt`
- 添加必要的注释和文档

### 文档更新

- 配置变更必须同步更新文档
- 新的故障案例添加到故障排查指南
- 更新文档后修改时间戳

---

## 许可证

(根据实际情况填写)

---

## 联系方式

如有问题或建议，请联系项目维护者。

---

**提示**: 
- 新同事请先阅读 [docs/QUICK_START_REFERENCE.md](docs/QUICK_START_REFERENCE.md)
- 遇到问题请查阅 [docs/TROUBLESHOOTING_GUIDE.md](docs/TROUBLESHOOTING_GUIDE.md)
- 所有文档索引请访问 [docs/README.md](docs/README.md)

---

# 启动补充

> 很多员工存在启动失败问题，针对此类问题。下面提供一个明确能正确启动服务的案例，注意，当前启动流程默认你已经完全按照要求，启动了前置 docker 依赖服务。如果并未启动，请立刻根据文档，检查docker服务状态

首先启动 Task Service：

```cmd
powershell -NoProfile -Command "[Console]::InputEncoding = [Text.UTF8Encoding]::new(`$false); [Console]::OutputEncoding = [Text.UTF8Encoding]::new(`$false); chcp 65001 > `$null; cd 'D:\Go-Project\video-In-Chinese\server\mcp\task'; go run . -f etc/task.yaml"
```

启动 AI Adaptor：

```cmd
执行PowerShell命令，包括以下步骤：
# 1. 设置UTF-8和chcp
[Console]::InputEncoding = [Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
chcp 65001 > $null
# 2. 正确加载 server/.env 文件中的OSS及秘钥变量
```

启动 Gateway：

```cmd
powershell -NoProfile -Command "[Console]::InputEncoding = [Text.UTF8Encoding]::new(`$false); [Console]::OutputEncoding = [Text.UTF8Encoding]::new(`$false); chcp 65001 > `$null; cd 'D:\Go-Project\video-In-Chinese\server\app\gateway'; go run . -f etc/gateway-api.yaml"
```

启动 Processor：

```cmd
powershell -NoProfile -Command "[Console]::InputEncoding = [Text.UTF8Encoding]::new(`$false); [Console]::OutputEncoding = [Text.UTF8Encoding]::new(`$false); chcp 65001 > `$null; cd 'D:\Go-Project\video-In-Chinese\server\mcp\processor'; go run . -f etc/processor.yaml"
```



