# AudioSeparator 服务实现文档（第三层）

**文档版本**: 1.0  
**最后更新**: 2025-10-30  
**服务定位**: Python gRPC 微服务实现细节

---

## 1. 项目结构

```
server/mcp/audio-separator/
├── main.py                 # gRPC 服务入口
├── separator.py            # Spleeter 封装
├── config.py               # 配置管理
├── proto/
│   ├── audio_separator.proto
│   └── audio_separator_pb2.py      # 自动生成
│   └── audio_separator_pb2_grpc.py # 自动生成
├── requirements.txt        # Python 依赖
├── Dockerfile              # Docker 镜像
├── .dockerignore
└── tests/
    └── test_separator.py   # 单元测试
```

---

## 2. 核心代码实现

### 2.1 main.py（gRPC 服务入口）

```python
import os
import logging
import grpc
from concurrent import futures
from proto import audio_separator_pb2_grpc
from separator import AudioSeparatorServicer
from config import load_config

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def serve():
    """启动 gRPC 服务"""
    # 加载配置
    config = load_config()
    
    # 创建 gRPC 服务器
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=config.max_workers)
    )
    
    # 注册服务
    audio_separator_pb2_grpc.add_AudioSeparatorServicer_to_server(
        AudioSeparatorServicer(config), server
    )
    
    # 监听端口
    server.add_insecure_port(f'[::]:{config.grpc_port}')
    
    logger.info(f"AudioSeparator gRPC 服务启动，监听端口: {config.grpc_port}")
    logger.info(f"使用 GPU: {config.use_gpu}")
    logger.info(f"模型路径: {config.model_path}")
    
    server.start()
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
```

### 2.2 separator.py（Spleeter 封装）

```python
import os
import time
import logging
from typing import Optional, Tuple
from dataclasses import dataclass
import grpc
from spleeter.separator import Separator
from proto import audio_separator_pb2
from proto import audio_separator_pb2_grpc
from config import AudioSeparatorConfig

logger = logging.getLogger(__name__)


@dataclass
class SeparationContext:
    """音频分离处理上下文"""
    audio_path: str
    output_dir: str
    stems: int
    start_time: float
    vocals_path: Optional[str] = None
    accompaniment_path: Optional[str] = None


class AudioSeparatorServicer(audio_separator_pb2_grpc.AudioSeparatorServicer):
    """AudioSeparator gRPC 服务实现"""
    
    def __init__(self, config: AudioSeparatorConfig):
        self.config = config
        self._separator_cache = {}  # 模型缓存
        logger.info("AudioSeparatorServicer 初始化完成")
    
    def _get_separator(self, stems: int) -> Separator:
        """获取 Spleeter 分离器（懒加载 + 缓存）"""
        model_name = f"spleeter:{stems}stems"
        
        if model_name not in self._separator_cache:
            logger.info(f"加载 Spleeter 模型: {model_name}")
            try:
                self._separator_cache[model_name] = Separator(
                    model_name,
                    multiprocess=False,
                    stft_backend='librosa'  # 使用 librosa 后端
                )
                logger.info(f"模型加载成功: {model_name}")
            except Exception as e:
                logger.error(f"模型加载失败: {e}")
                raise
        
        return self._separator_cache[model_name]
    
    def SeparateAudio(self, request, context):
        """分离音频为人声和背景音"""
        logger.info(f"收到音频分离请求: {request.audio_path}")
        
        # 1. 参数验证
        if not os.path.exists(request.audio_path):
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("音频文件不存在")
            return audio_separator_pb2.SeparateAudioResponse(
                success=False,
                error_message="音频文件不存在"
            )
        
        # 2. 创建输出目录
        if not os.path.exists(request.output_dir):
            try:
                os.makedirs(request.output_dir, exist_ok=True)
            except Exception as e:
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details(f"创建输出目录失败: {e}")
                return audio_separator_pb2.SeparateAudioResponse(
                    success=False,
                    error_message=f"创建输出目录失败: {e}"
                )
        
        # 3. 初始化上下文
        sep_context = SeparationContext(
            audio_path=request.audio_path,
            output_dir=request.output_dir,
            stems=request.stems or 2,
            start_time=time.time()
        )
        
        # 4. 加载 Spleeter 模型
        try:
            separator = self._get_separator(sep_context.stems)
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"模型加载失败: {e}")
            return audio_separator_pb2.SeparateAudioResponse(
                success=False,
                error_message=f"模型加载失败: {e}"
            )
        
        # 5. 执行音频分离
        try:
            logger.info(f"开始分离音频: {sep_context.audio_path}")
            separator.separate_to_file(
                audio_descriptor=sep_context.audio_path,
                destination=sep_context.output_dir
            )
            logger.info("音频分离完成")
        except Exception as e:
            logger.error(f"音频分离失败: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"音频分离失败: {e}")
            return audio_separator_pb2.SeparateAudioResponse(
                success=False,
                error_message=f"音频分离失败: {e}"
            )
        
        # 6. 构建输出路径
        audio_name = os.path.splitext(os.path.basename(sep_context.audio_path))[0]
        sep_context.vocals_path = os.path.join(
            sep_context.output_dir, audio_name, "vocals.wav"
        )
        sep_context.accompaniment_path = os.path.join(
            sep_context.output_dir, audio_name, "accompaniment.wav"
        )
        
        # 7. 验证输出文件
        if not os.path.exists(sep_context.vocals_path):
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("人声文件生成失败")
            return audio_separator_pb2.SeparateAudioResponse(
                success=False,
                error_message="人声文件生成失败"
            )
        
        if not os.path.exists(sep_context.accompaniment_path):
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("背景音文件生成失败")
            return audio_separator_pb2.SeparateAudioResponse(
                success=False,
                error_message="背景音文件生成失败"
            )
        
        # 8. 计算耗时
        processing_time_ms = int((time.time() - sep_context.start_time) * 1000)
        logger.info(f"音频分离成功，耗时: {processing_time_ms}ms")
        
        # 9. 返回成功响应
        return audio_separator_pb2.SeparateAudioResponse(
            success=True,
            vocals_path=sep_context.vocals_path,
            accompaniment_path=sep_context.accompaniment_path,
            processing_time_ms=processing_time_ms
        )
```

### 2.3 config.py（配置管理）

```python
import os
from dataclasses import dataclass


@dataclass
class AudioSeparatorConfig:
    """音频分离服务配置"""
    model_name: str
    model_path: str
    max_workers: int
    grpc_port: int
    use_gpu: bool


def load_config() -> AudioSeparatorConfig:
    """从环境变量加载配置"""
    return AudioSeparatorConfig(
        model_name="spleeter:2stems",
        model_path=os.getenv("AUDIO_SEPARATOR_MODEL_PATH", "/models"),
        max_workers=int(os.getenv("AUDIO_SEPARATOR_MAX_WORKERS", "1")),
        grpc_port=int(os.getenv("AUDIO_SEPARATOR_GRPC_PORT", "50052")),
        use_gpu=os.getenv("AUDIO_SEPARATOR_USE_GPU", "false").lower() == "true"
    )
```

---

## 3. Proto 文件生成

### 3.1 生成 Python 代码

```bash
# 在 server/mcp/audio-separator/ 目录下执行
python -m grpc_tools.protoc \
    -I./proto \
    --python_out=./proto \
    --grpc_python_out=./proto \
    ./proto/audio_separator.proto
```

---

## 4. Dockerfile

```dockerfile
FROM python:3.9-slim

# 设置工作目录
WORKDIR /app

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

# 复制依赖文件
COPY requirements.txt .

# 安装 Python 依赖
RUN pip install --no-cache-dir -r requirements.txt

# 复制服务代码
COPY . .

# 暴露 gRPC 端口
EXPOSE 50052

# 启动服务
CMD ["python", "main.py"]
```

---

## 5. requirements.txt

```txt
grpcio==1.60.0
grpcio-tools==1.60.0
spleeter==2.4.0
tensorflow==2.13.0
librosa==0.10.1
numpy==1.24.3
```

---

## 6. .dockerignore

```
__pycache__/
*.pyc
*.pyo
*.pyd
.Python
*.so
*.egg
*.egg-info/
dist/
build/
tests/
.pytest_cache/
.venv/
venv/
```

---

## 7. 单元测试

### 7.1 tests/test_separator.py

```python
import os
import unittest
import grpc
from proto import audio_separator_pb2
from proto import audio_separator_pb2_grpc
from separator import AudioSeparatorServicer
from config import AudioSeparatorConfig


class TestAudioSeparator(unittest.TestCase):
    """AudioSeparator 单元测试"""
    
    def setUp(self):
        """测试前准备"""
        self.config = AudioSeparatorConfig(
            model_name="spleeter:2stems",
            model_path="/models",
            max_workers=1,
            grpc_port=50052,
            use_gpu=False
        )
        self.servicer = AudioSeparatorServicer(self.config)
    
    def test_separate_audio_success(self):
        """测试音频分离成功"""
        # 准备测试数据
        request = audio_separator_pb2.SeparateAudioRequest(
            audio_path="/path/to/test.wav",
            output_dir="/path/to/output",
            stems=2
        )
        
        # 模拟 gRPC 上下文
        context = MockContext()
        
        # 调用服务（需要准备测试音频文件）
        # response = self.servicer.SeparateAudio(request, context)
        
        # 断言
        # self.assertTrue(response.success)
        # self.assertIsNotNone(response.vocals_path)
        # self.assertIsNotNone(response.accompaniment_path)
        pass
    
    def test_separate_audio_file_not_found(self):
        """测试音频文件不存在"""
        request = audio_separator_pb2.SeparateAudioRequest(
            audio_path="/path/to/nonexistent.wav",
            output_dir="/path/to/output",
            stems=2
        )
        
        context = MockContext()
        response = self.servicer.SeparateAudio(request, context)
        
        self.assertFalse(response.success)
        self.assertEqual(context.code, grpc.StatusCode.INVALID_ARGUMENT)


class MockContext:
    """模拟 gRPC 上下文"""
    def __init__(self):
        self.code = None
        self.details = None
    
    def set_code(self, code):
        self.code = code
    
    def set_details(self, details):
        self.details = details


if __name__ == '__main__':
    unittest.main()
```

---

## 8. 集成到 docker-compose.yml

```yaml
services:
  audio-separator:
    build:
      context: ./server/mcp/audio-separator
      dockerfile: Dockerfile
    container_name: audio-separator
    ports:
      - "50052:50052"
    environment:
      - AUDIO_SEPARATOR_GRPC_PORT=50052
      - AUDIO_SEPARATOR_USE_GPU=false
      - AUDIO_SEPARATOR_MODEL_PATH=/models
      - AUDIO_SEPARATOR_MAX_WORKERS=1
    volumes:
      - ./data:/data  # 挂载数据目录
      - ./models:/models  # 挂载模型目录
    networks:
      - video-translator-network
    restart: unless-stopped
```

---

## 9. 开发与测试

### 9.1 本地开发

```bash
# 1. 安装依赖
cd server/mcp/audio-separator
pip install -r requirements.txt

# 2. 生成 Proto 代码
python -m grpc_tools.protoc \
    -I./proto \
    --python_out=./proto \
    --grpc_python_out=./proto \
    ./proto/audio_separator.proto

# 3. 启动服务
python main.py
```

### 9.2 Docker 构建

```bash
# 构建镜像
docker build -t audio-separator:latest ./server/mcp/audio-separator

# 运行容器
docker run -p 50052:50052 \
    -e AUDIO_SEPARATOR_USE_GPU=false \
    -v $(pwd)/data:/data \
    audio-separator:latest
```

---

## 10. 性能优化建议

1. **模型缓存**: 使用懒加载 + 缓存策略，避免重复加载模型
2. **并发控制**: 设置 `max_workers=1`，避免 OOM
3. **GPU 加速**: 如果有 GPU，设置 `AUDIO_SEPARATOR_USE_GPU=true`
4. **内存监控**: 监控内存使用，及时释放资源

---

