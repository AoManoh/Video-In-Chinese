"""
AudioSeparator 服务配置管理模块

负责从环境变量读取配置，提供默认值
参考文档: AudioSeparator-design-detail.md v2.0
"""

import logging
import os
from pathlib import Path


class AudioSeparatorConfig:
    """AudioSeparator 服务配置类"""
    
    def __init__(self):
        """从环境变量初始化配置"""
        # gRPC 服务端口
        self.grpc_port = int(os.getenv('AUDIO_SEPARATOR_GRPC_PORT', '50052'))
        
        # Spleeter 模型配置
        self.model_name = os.getenv('AUDIO_SEPARATOR_MODEL_NAME', 'spleeter:2stems')
        self.model_path = os.getenv('AUDIO_SEPARATOR_MODEL_PATH', '/models')
        self.allowed_stems = (2, 4, 5)
        self.default_stems = self._parse_default_stems(self.model_name)
        
        # 并发控制
        self.max_workers = int(os.getenv('AUDIO_SEPARATOR_MAX_WORKERS', '1'))
        
        # 超时时间（秒）
        self.timeout_seconds = int(os.getenv('AUDIO_SEPARATOR_TIMEOUT', '600'))
        
        # GPU 配置
        self.use_gpu = os.getenv('AUDIO_SEPARATOR_USE_GPU', 'false').lower() == 'true'

        # 输出根目录（可选）
        self.output_root = os.getenv('AUDIO_SEPARATOR_OUTPUT_ROOT')
        if self.output_root:
            self.output_root = str(Path(self.output_root).expanduser().resolve())
        
        # 日志级别
        log_level_str = os.getenv('LOG_LEVEL', 'info').upper()
        self.log_level = getattr(logging, log_level_str, logging.INFO)
        
        # 验证配置
        self._validate()
    
    def _validate(self):
        """验证配置的有效性"""
        if self.grpc_port < 1024 or self.grpc_port > 65535:
            raise ValueError(f"Invalid gRPC port: {self.grpc_port}. Must be between 1024 and 65535.")
        
        if self.max_workers < 1:
            raise ValueError(f"Invalid max_workers: {self.max_workers}. Must be >= 1.")
        
        if self.timeout_seconds < 60:
            raise ValueError(f"Invalid timeout: {self.timeout_seconds}. Must be >= 60 seconds.")
        
        # 验证 model_name 格式
        valid_models = [f'spleeter:{stem}stems' for stem in self.allowed_stems]
        if self.model_name not in valid_models:
            raise ValueError(f"Invalid model_name: {self.model_name}. Must be one of {valid_models}.")

    @staticmethod
    def _parse_default_stems(model_name: str) -> int:
        """从模型名称中提取默认 stems（解析失败时返回 2）"""
        try:
            suffix = model_name.split(':', maxsplit=1)[1]
            if suffix.endswith('stems'):
                return int(suffix.replace('stems', ''))
        except (IndexError, ValueError):
            logging.getLogger(__name__).warning(
                "Unable to parse stems from model name: %s", model_name
            )
        return 2

    def resolve_stems(self, requested_stems: int) -> int:
        """将请求的 stems 映射为受支持的合法值"""
        if requested_stems == 0:
            return self.default_stems
        if requested_stems not in self.allowed_stems:
            raise ValueError(
                f"Invalid stems: {requested_stems}. Must be one of {self.allowed_stems} or 0."
            )
        return requested_stems

    def resolve_output_dir(self, requested_dir: str, task_id: str) -> str:
        """根据配置解析安全的输出目录"""
        if not requested_dir:
            raise ValueError("output_dir is required")

        resolved = Path(requested_dir).expanduser().resolve()

        if self.output_root:
            root = Path(self.output_root)
            if not resolved.is_relative_to(root):
                logging.getLogger(__name__).warning(
                    "Requested output directory %s is outside configured root %s; "
                    "falling back to root/task_id",
                    resolved,
                    root,
                )
                resolved = root / task_id
                resolved = resolved.resolve()
        return str(resolved)
    
    def __str__(self):
        """返回配置的字符串表示"""
        return (
            f"AudioSeparatorConfig(\n"
            f"  grpc_port={self.grpc_port},\n"
            f"  model_name={self.model_name},\n"
            f"  model_path={self.model_path},\n"
            f"  default_stems={self.default_stems},\n"
            f"  max_workers={self.max_workers},\n"
            f"  timeout_seconds={self.timeout_seconds},\n"
            f"  use_gpu={self.use_gpu},\n"
            f"  output_root={self.output_root},\n"
            f"  log_level={logging.getLevelName(self.log_level)}\n"
            f")"
        )


def setup_logging(config: AudioSeparatorConfig):
    """
    配置日志系统
    
    Args:
        config: AudioSeparatorConfig 实例
    """
    logging.basicConfig(
        level=config.log_level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    
    # 设置 TensorFlow 日志级别（避免过多的 INFO 日志）
    os.environ['TF_CPP_MIN_LOG_LEVEL'] = '2'  # 只显示 WARNING 和 ERROR
    
    logger = logging.getLogger(__name__)
    logger.info("Logging configured successfully")
    logger.info(f"Configuration: {config}")
