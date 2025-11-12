# AudioSeparator 服务

AudioSeparator 是一个 Python gRPC 微服务，负责音频分离（人声 + 背景音）。

**当前引擎**: Demucs v4.0.1 (Hybrid Transformer)

## 技术栈

- Python 3.11
- gRPC
- **Demucs v4.0.1** (Hybrid Transformer)
- **PyTorch 2.9+** (替代 TensorFlow)
- Soundfile (音频 I/O)

## 项目结构

```
audio_separator/
├── main.py                      # gRPC 服务入口
├── separator_service.py         # gRPC 服务实现
├── demucs_wrapper.py            # Demucs 模型封装（当前使用）
├── spleeter_wrapper.py          # Spleeter 模型封装（已废弃）
├── config.py                    # 配置管理
├── proto/
│   ├── audioseparator.proto     # gRPC 接口定义
│   ├── audioseparator_pb2.py    # 自动生成的 Python gRPC 代码
│   └── audioseparator_pb2_grpc.py
├── requirements_demucs.txt      # Demucs 依赖清单（当前使用）
├── requirements.txt             # Spleeter 依赖清单（已废弃）
└── README.md                    # 本文件
```

## 环境要求

### Python 版本
- **要求版本**: Python 3.11
- **虚拟环境**: 使用项目根目录的 `server/.venv`

### 系统依赖
- **ffmpeg**: 音频格式转换（Demucs 必需）
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

### 1. 使用项目虚拟环境
本服务使用项目根目录的统一虚拟环境：
```powershell
# Windows
D:\Go-Project\video-In-Chinese\server\.venv\Scripts\python.exe
```

### 2. 安装 Demucs 依赖
```powershell
cd D:\Go-Project\video-In-Chinese\server\mcp\audio_separator
D:\Go-Project\video-In-Chinese\server\.venv\Scripts\pip.exe install -r requirements_demucs.txt
```

**首次安装注意**：
- PyTorch 安装较大（约 100+ MB）
- 首次运行时会自动下载 Demucs 模型（约 80 MB）

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
| `AUDIO_SEPARATOR_MODEL_NAME` | `htdemucs` | Demucs 模型名称 |
| `AUDIO_SEPARATOR_MODEL_PATH` | `/models` | 模型文件路径（未使用） |
| `AUDIO_SEPARATOR_MAX_WORKERS` | `1` | 最大并发处理数 |
| `AUDIO_SEPARATOR_TIMEOUT` | `600` | 超时时间（秒） |
| `AUDIO_SEPARATOR_USE_GPU` | `false` | 是否使用 GPU（需要 NVIDIA GPU + CUDA） |
| `LOG_LEVEL` | `info` | 日志级别 (debug/info/warn/error) |

### 可用模型

| 模型 | 说明 | 性能 |
|------|------|------|
| `htdemucs` | Hybrid Transformer (默认) | 最佳质量 |
| `htdemucs_ft` | Fine-tuned 版本 | 更好质量，慢 4x |
| `mdx_extra` | MDX Challenge 2nd | 良好，较快 |

### 示例配置文件 (.env)
```bash
AUDIO_SEPARATOR_GRPC_PORT=50052
AUDIO_SEPARATOR_MAX_WORKERS=1
AUDIO_SEPARATOR_TIMEOUT=600
LOG_LEVEL=info
```

## 运行服务

### 开发模式（Windows）
```powershell
# 设置 UTF-8（必须）
[Console]::InputEncoding = [Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
chcp 65001 > $null

# 启动服务
cd D:\Go-Project\video-In-Chinese\server\mcp\audio_separator
D:\Go-Project\video-In-Chinese\server\.venv\Scripts\python.exe main.py
```

**成功标志**：
```
AudioSeparator service started on port 50052
DemucsWrapper initialized with default_model=htdemucs
```

### 生产模式（Docker）
```bash
docker build -t audio-separator:demucs .
docker run -p 50052:50052 audio-separator:demucs
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

### Demucs (htdemucs) - CPU 模式

**实测数据**（128.78 秒音频）：
- 处理时间：40.36 秒
- 内存占用：~200 MB（子进程）
- 模型加载：< 1 秒（缓存命中）
- 首次模型下载：2-5 分钟

**输出**：
- 5 个文件：vocals, drums, bass, other, accompaniment
- 每个文件：21.66 MB（立体声 44.1kHz）

### GPU 模式（如果启用）
- 处理时间：预计快 5-10x
- 需要：NVIDIA GPU + CUDA
- 内存占用：1-2GB VRAM

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

### 问题 1: 模型下载失败
**症状**：
```
Failed to download model: Connection timeout
```

**解决方案**：
1. 检查网络连接（需要访问 `dl.fbaipublicfiles.com`）
2. 重试启动（会自动重新下载）
3. 模型会缓存到：`~/.cache/torch/hub/checkpoints/`

### 问题 2: 端口 50052 已被占用
**解决方案**：
```powershell
netstat -ano | findstr :50052
taskkill /F /PID [PID]
```

### 问题 3: 内存不足 (OOM)
**解决方案**:
1. 确保 `AUDIO_SEPARATOR_MAX_WORKERS=1`
2. Demucs 需要约 200-300 MB 内存（CPU）
3. 如果仍不足，考虑使用更快但更小的模型（`mdx_extra`）

### 问题 4: ffmpeg 未找到
**解决方案**: 安装 ffmpeg 并添加到系统 PATH

### 问题 5: 进程显示系统 Python 路径
**这不是问题**：参见上方"Windows 本地开发环境"章节

### 问题 6: accompaniment 文件缺失
**确认**：
- Demucs 会生成 5 个文件：vocals, drums, bass, other, accompaniment
- accompaniment 是 drums + bass + other 的混合
- 如果缺失，检查 demucs_wrapper.py 是否包含混合逻辑

## 参考文档

### 官方文档
- **Demucs GitHub**: https://github.com/adefossez/demucs
- **DeepWiki**: facebookresearch/demucs
- **Context7**: /adefossez/demucs (Trust Score: 8.5)

### 项目文档
- **服务说明**: [docs/AudioSeparator服务说明.md](../../../docs/AudioSeparator服务说明.md)
- **虚拟环境分析**: [docs/VENV_SUBPROCESS_ANALYSIS.md](../../../docs/VENV_SUBPROCESS_ANALYSIS.md)
- **Spleeter 诊断**: [docs/Spleeter音频分离失败问题诊断报告.md](../../../docs/Spleeter音频分离失败问题诊断报告.md)
- **Demucs 评估**: [docs/Demucs技术升级方案评估.md](../../../docs/Demucs技术升级方案评估.md)

## 开发状态

- [x] Phase 1: 基础设施搭建
- [x] Phase 2: Demucs 模型封装（替换 Spleeter）
- [x] Phase 3: 音频分离逻辑（含背景音混合）
- [x] Phase 4: 并发控制
- [x] Phase 5: 测试验证
- [x] Phase 6: 文档完善

**当前版本**: v2.0 (Demucs)  
**上次更新**: 2025-11-10

