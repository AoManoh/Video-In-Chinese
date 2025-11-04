"""
Spleeter 模型封装模块

实现懒加载、模型缓存和错误处理
参考文档: AudioSeparator-design-detail.md v2.0 第2.2节
"""

import logging
import threading
import time
from typing import Dict

from spleeter.separator import Separator


logger = logging.getLogger(__name__)


class SpleeterWrapper:
    """
    Spleeter 模型封装类
    
    核心功能:
    1. 懒加载: 首次调用时才加载模型
    2. 模型缓存: 使用字典缓存已加载的模型 {stems: model}
    3. 错误处理: 捕获模型加载和分离过程中的异常
    
    设计决策:
    - 为什么懒加载? 服务启动时间从 30-60 秒降低到 1-2 秒
    - 为什么缓存? 避免重复加载，后续请求响应时间降低到毫秒级
    - 为什么字典存储? 支持多 stems 模式切换（2stems/4stems/5stems）
    """
    
    def __init__(
        self,
        model_path: str = '/models',
        default_model: str = 'spleeter:2stems',
        use_gpu: bool = False,
    ):
        """
        初始化 SpleeterWrapper
        
        Args:
            model_path: 模型文件路径
        """
        self.model_path = model_path
        self.default_model = default_model
        self.default_stems = self._parse_model_stems(default_model)
        self.use_gpu = use_gpu
        self.models: Dict[int, Separator] = {}  # 缓存已加载的模型 {stems: model}
        self._lock = threading.RLock()
        logger.info(
            "SpleeterWrapper initialized with model_path=%s, default_model=%s, use_gpu=%s",
            model_path,
            default_model,
            use_gpu,
        )

    @staticmethod
    def _parse_model_stems(model_name: str) -> int:
        """解析模型名称中的默认 stems 值"""
        try:
            suffix = model_name.split(':', maxsplit=1)[1]
            if suffix.endswith('stems'):
                return int(suffix.replace('stems', ''))
        except (IndexError, ValueError):
            logger.warning("Unable to parse stems from model name: %s", model_name)
        return 2
    
    def get_model(self, stems: int = 2) -> Separator:
        """
        获取 Spleeter 模型（懒加载 + 缓存）
        
        Args:
            stems: 分离模式（2/4/5）
        
        Returns:
            Separator: Spleeter 模型实例
        
        Raises:
            ValueError: stems 参数无效
            RuntimeError: 模型加载失败
        """
        # 参数验证
        if stems == 0:
            stems = self.default_stems

        if stems not in [2, 4, 5]:
            raise ValueError(f"Invalid stems: {stems}. Must be 2, 4, or 5.")

        # 检查缓存
        with self._lock:
            if stems in self.models:
                logger.debug("Model cache hit: %sstems", stems)
                return self.models[stems]
        
        # 懒加载模型
        logger.info(f"Loading Spleeter model: {stems}stems (this may take 30-60 seconds)...")
        start_time = time.time()
        
        try:
            # 创建 Separator 实例
            # params_descriptor: 模型配置（2stems/4stems/5stems）
            # multiprocess: 是否使用多进程（设置为 False，避免并发问题）
            separator_kwargs = {
                'multiprocess': False,
            }
            if self.model_path:
                separator_kwargs['model_root'] = self.model_path

            model = Separator(
                f'spleeter:{stems}stems',
                **separator_kwargs,
            )
            
            load_time = time.time() - start_time
            logger.info(f"Spleeter model loaded successfully: {stems}stems, time={load_time:.2f}s")
            
            # 缓存模型
            with self._lock:
                self.models[stems] = model
            return model
            
        except Exception as e:
            logger.error(f"Failed to load Spleeter model: {stems}stems, error={str(e)}")
            raise RuntimeError(f"Model loading failed: {str(e)}") from e
    
    def separate(
        self,
        audio_path: str,
        output_dir: str,
        stems: int = 2,
        timeout_seconds: int | None = None,
    ) -> Dict[str, str]:
        """
        分离音频文件
        
        Args:
            audio_path: 输入音频文件路径
            output_dir: 输出目录路径
            stems: 分离模式（2/4/5）
        
        Returns:
            Dict[str, str]: 分离后的文件路径字典
                - 2stems: {'vocals': path, 'accompaniment': path}
                - 4stems: {'vocals': path, 'drums': path, 'bass': path, 'other': path}
                - 5stems: {'vocals': path, 'drums': path, 'bass': path, 'piano': path, 'other': path}
        
        Raises:
            FileNotFoundError: 输入音频文件不存在
            RuntimeError: 音频分离失败
        """
        import os
        
        # 验证输入文件存在
        if not os.path.exists(audio_path):
            raise FileNotFoundError(f"Audio file not found: {audio_path}")
        
        # 获取模型（懒加载 + 缓存）
        model = self.get_model(stems)
        
        # 执行音频分离
        logger.info(f"Starting audio separation: audio_path={audio_path}, stems={stems}")
        start_time = time.time()
        
        try:
            # Spleeter 的 separate_to_file 方法
            # 参数:
            #   - audio_descriptor: 音频文件路径
            #   - destination: 输出目录
            #   - codec: 输出格式（默认 wav）
            #   - bitrate: 比特率（默认 128k）
            #   - filename_format: 文件名格式（默认 {filename}/{instrument}.{codec}）
            model.separate_to_file(
                audio_path,
                output_dir,
                codec='wav',
                filename_format='{instrument}.{codec}'
            )

            separation_time = time.time() - start_time
            logger.info(f"Audio separation completed: time={separation_time:.2f}s")

            if timeout_seconds and separation_time > timeout_seconds:
                raise RuntimeError(
                    f"Separation exceeded timeout ({separation_time:.2f}s > {timeout_seconds}s)"
                )
            
            # 构建输出文件路径
            output_paths = {}
            if stems == 2:
                output_paths['vocals'] = os.path.join(output_dir, 'vocals.wav')
                output_paths['accompaniment'] = os.path.join(output_dir, 'accompaniment.wav')
            elif stems == 4:
                output_paths['vocals'] = os.path.join(output_dir, 'vocals.wav')
                output_paths['drums'] = os.path.join(output_dir, 'drums.wav')
                output_paths['bass'] = os.path.join(output_dir, 'bass.wav')
                output_paths['other'] = os.path.join(output_dir, 'other.wav')
            elif stems == 5:
                output_paths['vocals'] = os.path.join(output_dir, 'vocals.wav')
                output_paths['drums'] = os.path.join(output_dir, 'drums.wav')
                output_paths['bass'] = os.path.join(output_dir, 'bass.wav')
                output_paths['piano'] = os.path.join(output_dir, 'piano.wav')
                output_paths['other'] = os.path.join(output_dir, 'other.wav')
            
            return output_paths
            
        except Exception as e:
            logger.error(f"Audio separation failed: error={str(e)}")
            raise RuntimeError(f"Audio separation failed: {str(e)}") from e
    
    def clear_cache(self):
        """清空模型缓存（用于测试或内存管理）"""
        with self._lock:
            cached = len(self.models)
            self.models.clear()
        logger.info("Cleared model cache: %s models were cached", cached)
