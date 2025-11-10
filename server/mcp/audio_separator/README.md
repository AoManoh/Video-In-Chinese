# AudioSeparator 服务

AudioSeparator 是一个 Python gRPC 微服务，负责音频分离（人声 + 背景音）。

## 技术栈

- Python 3.9+
- gRPC
- Spleeter (2stems 模型)
- TensorFlow 2.13+

## 项目结构

```
audio_separator/
├── main.py                      # gRPC 服务入口
├── separator_service.py         # gRPC 服务实现
├── spleeter_wrapper.py          # Spleeter 模型封装
├── config.py                    # 配置管理
├── proto/
│   ├── audioseparator.proto     # gRPC 接口定义
│   ├── audioseparator_pb2.py    # 自动生成的 Python gRPC 代码
│   └── audioseparator_pb2_grpc.py
├── tests/                       # 测试文件目录
├── requirements.txt             # Python 依赖清单
└── README.md                    # 本文件
```

## 环境要求

### Python 版本
- 最低版本: Python 3.9
- 推荐版本: Python 3.10

### 系统依赖
- **ffmpeg**: 音频格式转换
- **libsndfile1**: 音频文件读写

#### 安装系统依赖 (Ubuntu/Debian)
```bash
sudo apt-get update
sudo apt-get install -y ffmpeg libsndfile1
```

#### 安装系统依赖 (macOS)
```bash
brew install ffmpeg libsndfile
```

#### 安装系统依赖 (Windows)
- 下载 ffmpeg: https://ffmpeg.org/download.html
- 添加 ffmpeg 到系统 PATH

## 安装步骤

### 1. 创建虚拟环境
```bash
cd server/mcp/audio_separator
python3 -m venv venv
source venv/bin/activate  # Linux/macOS
# 或
venv\Scripts\activate  # Windows
```

### 2. 安装 Python 依赖
```bash
pip install -r requirements.txt
```

### 3. 生成 gRPC 代码
```bash
python -m grpc_tools.protoc \
    -I./proto \
    --python_out=./proto \
    --grpc_python_out=./proto \
    ./proto/audioseparator.proto
```

这将生成以下文件:
- `proto/audioseparator_pb2.py` - Protocol Buffers 消息类
- `proto/audioseparator_pb2_grpc.py` - gRPC 服务类

### 4. 修复导入路径（如果需要）
生成的代码可能需要修复导入路径。编辑 `proto/audioseparator_pb2_grpc.py`，将:
```python
import audioseparator_pb2 as audioseparator__pb2
```
改为:
```python
from proto import audioseparator_pb2 as audioseparator__pb2
```

## 配置

通过环境变量配置服务:

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `AUDIO_SEPARATOR_GRPC_PORT` | `50052` | gRPC 服务端口 |
| `AUDIO_SEPARATOR_MODEL_NAME` | `spleeter:2stems` | Spleeter 模型名称 |
| `AUDIO_SEPARATOR_MODEL_PATH` | `/models` | 模型文件路径 |
| `AUDIO_SEPARATOR_MAX_WORKERS` | `1` | 最大并发处理数 |
| `AUDIO_SEPARATOR_TIMEOUT` | `600` | 超时时间（秒） |
| `AUDIO_SEPARATOR_USE_GPU` | `false` | 是否使用 GPU |
| `LOG_LEVEL` | `info` | 日志级别 (debug/info/warn/error) |

### 示例配置文件 (.env)
```bash
AUDIO_SEPARATOR_GRPC_PORT=50052
AUDIO_SEPARATOR_MAX_WORKERS=1
AUDIO_SEPARATOR_TIMEOUT=600
LOG_LEVEL=info
```

## 运行服务

### 开发模式
```bash
python main.py
```

### 生产模式（使用环境变量）
```bash
export AUDIO_SEPARATOR_GRPC_PORT=50052
export LOG_LEVEL=info
python main.py
```

## 测试

### 运行单元测试
```bash
pytest tests/
```

### 运行集成测试
```bash
pytest tests/test_separator_service_integration.py
```

## 性能指标

### CPU 模式
- 10分钟音频处理时间: 5-8 分钟
- 内存占用: 500MB-1GB
- 模型加载时间: 30-60 秒

### GPU 模式
- 10分钟音频处理时间: 2-3 分钟
- 内存占用: 1-2GB
- 模型加载时间: 10-20 秒

## 注意事项

### Windows 本地开发环境

在 Windows 环境中使用虚拟环境时，您可能会观察到以下现象：

**现象描述**：
- 使用虚拟环境的 `python.exe` 启动服务后，进程列表中显示的是**系统 Python 路径**
- 例如：`C:\Environment\python\python3.11\python.exe` 而不是 `D:\..\.venv\Scripts\python.exe`

**根本原因**：
这是 **Windows venv 的标准实现方式**，不是 Bug：
1. 虚拟环境的 `python.exe` 是一个**轻量级启动器**（约 3-4 MB 内存）
2. 它的职责是设置正确的 `sys.path`（指向虚拟环境的 site-packages）
3. 然后调用系统 Python 解释器执行实际工作（约 200+ MB 内存）

**验证方法**：
```powershell
# 检查进程
Get-Process python | Select-Object Id, @{Name='Memory(MB)';Expression={[math]::Round($_.WorkingSet/1MB,2)}}, Path

# 您会看到两个进程：
# 1. 小进程（3-4 MB）：虚拟环境的启动器
# 2. 大进程（200+ MB）：系统 Python，但加载虚拟环境的包
```

**重要说明**：
- **功能完全正常**：服务正确加载虚拟环境的所有依赖（torch, demucs 等）
- **依赖隔离有效**：所有第三方包来自虚拟环境，不会污染系统环境
- **仅视觉混淆**：进程列表显示系统 Python 路径，但不影响功能
- **生产环境无此问题**：Docker 部署时不存在此现象

**详细分析**：
参见 [docs/VENV_SUBPROCESS_ANALYSIS.md](../../docs/VENV_SUBPROCESS_ANALYSIS.md)

---

## 故障排查

### 问题: 模型加载失败
**解决方案**: 检查网络连接，Demucs 首次运行时会自动下载模型文件（约 80MB）

### 问题: 内存不足 (OOM)
**解决方案**:
1. 确保 `AUDIO_SEPARATOR_MAX_WORKERS=1`
2. 增加系统内存或使用更小的音频文件
3. Demucs 模型需要约 1-2GB 内存

### 问题: ffmpeg 未找到
**解决方案**: 安装 ffmpeg 并添加到系统 PATH

### 问题: 进程显示系统 Python 路径
**这不是问题**：参见上方"Windows 本地开发环境"章节

## 参考文档

- 第三层设计文档: `notes/server/3rd/AudioSeparator-design-detail.md` v2.0
- 第二层设计文档: `notes/server/2nd/AudioSeparator-design.md` v1.5
- 第一层架构文档: `notes/server/1st/Base-Design.md` v2.2

## 开发状态

- [x] Phase 1: 基础设施搭建
- [ ] Phase 2: Spleeter 模型封装
- [ ] Phase 3: 音频分离逻辑
- [ ] Phase 4: 并发控制
- [ ] Phase 5: 测试实现
- [ ] Phase 6: 文档和代码审查

