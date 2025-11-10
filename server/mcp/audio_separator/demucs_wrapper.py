"""
Demucs 模型封装模块

实现懒加载、模型缓存和错误处理
基于官方文档: https://github.com/facebookresearch/demucs
参考: DeepWiki facebookresearch/demucs/4-python-api

注意: demucs 4.0.1 没有 api.py，使用底层 API
- pretrained.get_model() 加载模型  
- apply.apply_model() 应用模型
- audio 模块处理音频I/O
"""

import logging
import threading
import time
import os
import torch
from typing import Dict

# Demucs 底层 API
# 来源: DeepWiki facebookresearch/demucs
from demucs.pretrained import get_model
from demucs.apply import apply_model
from demucs.audio import AudioFile
import soundfile as sf  # 使用 soundfile 保存音频


logger = logging.getLogger(__name__)


class DemucsWrapper:
    """
    Demucs 模型封装类
    
    核心功能:
    1. 懒加载: 首次调用时才加载模型
    2. 模型缓存: 使用字典缓存已加载的模型 {model_name: separator}
    3. 错误处理: 捕获模型加载和分离过程中的异常
    
    设计决策:
    - 为什么懒加载? 服务启动时间快速
    - 为什么缓存? 避免重复加载，后续请求响应时间降低到毫秒级
    - 为什么使用 Demucs? SOTA 性能，Hybrid Transformer 架构，更好处理非音乐场景
    
    官方文档:
    - Python API: https://github.com/adefossez/demucs/blob/main/docs/api.md
    - Installation: https://github.com/adefossez/demucs/blob/main/README.md
    """
    
    def __init__(
        self,
        default_model: str = 'htdemucs',
        device: str = 'cpu',
        segment: float = None,
        shifts: int = 1,
    ):
        """
        初始化 DemucsWrapper
        
        Args:
            default_model: 默认模型名称
                - 'htdemucs': Hybrid Transformer Demucs (默认，SOTA)
                - 'htdemucs_ft': Fine-tuned版本（更高质量但慢4倍）
                - 'mdx_extra': MDX Challenge 2nd place
            device: 计算设备 ('cpu' 或 'cuda')
            segment: 每个片段的长度（秒），None表示自动
            shifts: 时间偏移次数（提高质量，默认1）
            
        参考:
        - DeepWiki: facebookresearch/demucs/5.1-models-and-variants
        - DeepWiki: facebookresearch/demucs/5.2-model-loading-system
        """
        self.default_model = default_model
        self.device = torch.device(device)
        self.segment = segment
        self.shifts = shifts
        self.models: Dict[str, any] = {}  # 缓存已加载的模型
        self._lock = threading.RLock()
        
        logger.info(
            "DemucsWrapper initialized with default_model=%s, device=%s, segment=%s, shifts=%s",
            default_model,
            device,
            segment,
            shifts,
        )
    
    def get_model(self, model_name: str = None):
        """
        获取 Demucs 模型（懒加载 + 缓存）
        
        Args:
            model_name: 模型名称，None表示使用默认模型
        
        Returns:
            model: Demucs 模型实例
        
        Raises:
            RuntimeError: 模型加载失败
            
        参考:
        - DeepWiki: facebookresearch/demucs/5.2-model-loading-system
        """
        if model_name is None:
            model_name = self.default_model
        
        # 检查缓存
        with self._lock:
            if model_name in self.models:
                logger.debug("Model cache hit: %s", model_name)
                return self.models[model_name]
        
        # 懒加载模型
        logger.info(f"Loading Demucs model: {model_name} (first use will download from remote)...")
        start_time = time.time()
        
        try:
            # 使用 Demucs pretrained API
            # 参考: DeepWiki facebookresearch/demucs/5.2-model-loading-system
            model = get_model(model_name)
            model.to(self.device)
            model.eval()  # 设置为评估模式
            
            load_time = time.time() - start_time
            logger.info(f"Demucs model loaded successfully: {model_name}, time={load_time:.2f}s")
            
            # 缓存模型
            with self._lock:
                self.models[model_name] = model
            return model
            
        except Exception as e:
            logger.error(f"Failed to load Demucs model: {model_name}, error={str(e)}")
            raise RuntimeError(f"Model loading failed: {str(e)}") from e
    
    def separate(
        self,
        audio_path: str,
        output_dir: str,
        model_name: str = None,
        timeout_seconds: int | None = None,
    ) -> Dict[str, str]:
        """
        分离音频文件
        
        Args:
            audio_path: 输入音频文件路径
            output_dir: 输出目录路径
            model_name: 模型名称（None表示使用默认模型）
            timeout_seconds: 超时时间（秒）
        
        Returns:
            Dict[str, str]: 分离后的文件路径字典
                默认4 stems: {'vocals': path, 'drums': path, 'bass': path, 'other': path}
        
        Raises:
            FileNotFoundError: 输入音频文件不存在
            RuntimeError: 音频分离失败或超时
            
        参考:
        - DeepWiki: facebookresearch/demucs/6-audio-processing-pipeline
        """
        # 验证输入文件存在
        if not os.path.exists(audio_path):
            raise FileNotFoundError(f"Audio file not found: {audio_path}")
        
        # 获取模型（懒加载 + 缓存）
        model = self.get_model(model_name)
        
        # 执行音频分离
        logger.info(f"Starting audio separation: audio_path={audio_path}, model={model_name or self.default_model}")
        start_time = time.time()
        
        try:
            # 1. 加载音频
            # 参考: DeepWiki facebookresearch/demucs/6-audio-processing-pipeline
            wav = AudioFile(audio_path).read(
                seek_time=0,
                duration=None,
                streams=0,
                samplerate=model.samplerate,
                channels=model.audio_channels,
            )
            # AudioFile.read() 返回 Tensor，shape: (channels, samples)
            ref = wav.mean(0)  # 参考信号（用于归一化）
            wav = (wav - ref.mean()) / ref.std()  # 归一化
            
            # 转移到指定设备并添加 batch 维度
            wav = wav.to(self.device)
            wav = wav.unsqueeze(0)  # shape: (1, channels, samples)
            
            # 2. 应用模型
            # 参考: DeepWiki facebookresearch/demucs/6-audio-processing-pipeline
            with torch.no_grad():
                sources = apply_model(
                    model,
                    wav,
                    device=self.device,
                    shifts=self.shifts,
                    split=True,  # 分段处理以节省内存
                    overlap=0.25,  # 25%重叠
                    segment=self.segment,
                )
            
            # sources shape: (1, num_sources, channels, samples)
            sources = sources * ref.std() + ref.mean()  # 反归一化
            
            separation_time = time.time() - start_time
            logger.info(f"Audio separation completed: time={separation_time:.2f}s")

            # 超时判定
            if timeout_seconds and separation_time > timeout_seconds:
                raise RuntimeError(
                    f"Separation exceeded timeout ({separation_time:.2f}s > {timeout_seconds}s)"
                )
            
            # 3. 创建输出目录
            input_filename = os.path.splitext(os.path.basename(audio_path))[0]
            output_subdir = os.path.join(output_dir, input_filename)
            os.makedirs(output_subdir, exist_ok=True)
            
            # 4. 保存分离后的音频文件
            # Demucs 模型的 sources 顺序：通常是 drums, bass, other, vocals
            # 参考: DeepWiki facebookresearch/demucs/5.1-models-and-variants
            output_paths = {}
            sources = sources[0]  # 移除 batch 维度: (num_sources, channels, samples)
            
            # 保存所有单独的 stems
            stems_dict = {}
            for i, source_name in enumerate(model.sources):
                stem_audio = sources[i]  # (channels, samples)
                stem_path = os.path.join(output_subdir, f'{source_name}.wav')
                
                # 保存音频（使用 soundfile，避免 torchcodec 依赖）
                # stem_audio shape: (channels, samples)
                # soundfile 需要 (samples, channels)
                stem_np = stem_audio.cpu().numpy().T  # 转置: (samples, channels)
                
                # 使用 soundfile 保存
                sf.write(
                    stem_path,
                    stem_np,
                    model.samplerate,
                    subtype='PCM_16',  # 16-bit PCM
                )
                
                output_paths[source_name] = stem_path
                stems_dict[source_name] = stem_audio
                logger.debug(f"Saved {source_name}: {stem_path}")
            
            # 5. 创建完整的背景音（accompaniment = drums + bass + other）
            # 这样才等同于 Spleeter 的 accompaniment（所有非人声内容）
            logger.info("Creating full accompaniment (drums + bass + other)...")
            
            accompaniment_audio = torch.zeros_like(stems_dict['vocals'])
            for stem_name in ['drums', 'bass', 'other']:
                if stem_name in stems_dict:
                    accompaniment_audio += stems_dict[stem_name]
            
            # 保存完整背景音
            accompaniment_path = os.path.join(output_subdir, 'accompaniment.wav')
            accompaniment_np = accompaniment_audio.cpu().numpy().T  # (samples, channels)
            sf.write(
                accompaniment_path,
                accompaniment_np,
                model.samplerate,
                subtype='PCM_16',
            )
            
            output_paths['accompaniment'] = accompaniment_path
            logger.info(f"Saved full accompaniment: {accompaniment_path}")
            
            logger.info(f"All stems saved to: {output_subdir}")
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

