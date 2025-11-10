"""
AudioSeparator gRPC 服务实现

实现 SeparateAudio 接口的关键逻辑步骤。
参考文档: AudioSeparator-design.md v1.5 第 6 章
"""

import logging
import os
import time
from typing import Dict

import grpc
from concurrent import futures

from proto import audioseparator_pb2
from proto import audioseparator_pb2_grpc

from demucs_wrapper import DemucsWrapper
from config import AudioSeparatorConfig


logger = logging.getLogger(__name__)


class AudioSeparatorServicer(audioseparator_pb2_grpc.AudioSeparatorServicer):
    """AudioSeparator gRPC 服务实现"""

    def __init__(self, config: AudioSeparatorConfig):
        """初始化服务并准备模型包装器"""
        self.config = config
        # 使用 Demucs 替换 Spleeter
        # 参考: DeepWiki facebookresearch/demucs/4-python-api
        device = 'cuda' if config.use_gpu else 'cpu'
        self.demucs = DemucsWrapper(
            default_model='htdemucs',  # Hybrid Transformer Demucs (SOTA)
            device=device,
            segment=None,  # 自动确定片段长度
            shifts=1,  # 默认1次时间偏移
        )
        logger.info(
            "AudioSeparatorServicer initialized with Demucs (device=%s, model=htdemucs)",
            device,
        )

    def SeparateAudio(self, request, context):
        """分离音频为多个 stems"""
        try:
            audio_path, output_dir, stems, task_id = self._normalize_request(request)
        except ValueError as error:
            logger.warning("Invalid request: %s", error)
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(error))
            return audioseparator_pb2.SeparateAudioResponse(
                success=False,
                error_message=str(error),
            )

        try:
            os.makedirs(output_dir, exist_ok=True)
            logger.info("Output directory resolved: %s", output_dir)
        except Exception as error:  # pragma: no cover - OS errors hard to simulate
            logger.error("Failed to create output directory: %s", error)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to create output directory: {error}")
            return audioseparator_pb2.SeparateAudioResponse(
                success=False,
                error_message=f"Failed to create output directory: {error}",
            )

        start_time = time.time()
        logger.info(
            "Audio separation started: task_id=%s, stems=%s, output_dir=%s",
            task_id,
            stems,
            output_dir,
        )

        try:
            # 使用 Demucs 分离
            # Demucs 默认输出4个stems，我们需要转换为与Spleeter兼容的格式
            output_paths = self.demucs.separate(
                audio_path=audio_path,
                output_dir=output_dir,
                model_name=None,  # 使用默认模型
                timeout_seconds=self.config.timeout_seconds,
            )
            self._validate_output_files(stems, output_paths)

            processing_time_ms = int((time.time() - start_time) * 1000)

            log_details = {
                stem: f"{os.path.getsize(path) / (1024 * 1024):.2f}MB"
                for stem, path in output_paths.items()
            }
            logger.info(
                "Audio separation completed: task_id=%s, time=%sms, stems=%s",
                task_id,
                processing_time_ms,
                log_details,
            )

            # Demucs 输出: drums, bass, vocals, other, accompaniment
            # accompaniment 是 drums + bass + other 的混合（完整背景音）
            response = audioseparator_pb2.SeparateAudioResponse(
                success=True,
                vocals_path=output_paths.get('vocals', ''),
                accompaniment_path=output_paths.get('accompaniment', ''),  # 完整背景音
                processing_time_ms=processing_time_ms,
            )

            for stem_name, stem_path in output_paths.items():
                # 将全部 stems 信息回传客户端，便于调用方自定义后续流程
                stem_entry = response.stems.add()
                stem_entry.name = stem_name
                stem_entry.path = stem_path

            return response

        except FileNotFoundError as error:
            logger.warning("Audio file not found: %s", error)
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(error))
            return audioseparator_pb2.SeparateAudioResponse(
                success=False,
                error_message=str(error),
            )

        except RuntimeError as error:
            logger.error("Audio separation failed: %s", error)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(error))
            return audioseparator_pb2.SeparateAudioResponse(
                success=False,
                error_message=str(error),
            )

        except Exception as error:  # pragma: no cover - defensive programming
            logger.error("Unexpected error: %s", error, exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Unexpected error: {error}")
            return audioseparator_pb2.SeparateAudioResponse(
                success=False,
                error_message=f"Unexpected error: {error}",
            )

    def _normalize_request(self, request) -> tuple[str, str, int, str]:
        """验证并规范化请求参数"""
        audio_path = request.audio_path.strip()
        if not audio_path:
            raise ValueError("audio_path is required")

        if not os.path.exists(audio_path):
            raise ValueError(f"Audio file not found: {audio_path}")

        task_id = self._extract_task_id(audio_path)

        stems = self.config.resolve_stems(request.stems)
        # 通过配置层的规范化逻辑限制输出目录，防止写入系统敏感位置
        output_dir = self.config.resolve_output_dir(request.output_dir, task_id)

        return audio_path, output_dir, stems, task_id

    def _validate_output_files(self, stems: int, output_paths: Dict[str, str]):
        """验证分离结果的输出文件"""
        if not output_paths:
            raise RuntimeError("No output files were produced by the separation pipeline")

        # Demucs 总是输出4个stems: drums, bass, vocals, other
        # 参考: DeepWiki facebookresearch/demucs/1-overview
        min_file_size = 1024  # bytes

        for stem_name, stem_path in output_paths.items():
            if not stem_path:
                raise RuntimeError(f"Missing output path for stem '{stem_name}'")

            if not os.path.exists(stem_path):
                raise RuntimeError(f"Output file not found: {stem_path}")

            file_size = os.path.getsize(stem_path)
            if file_size < min_file_size:
                raise RuntimeError(
                    f"Output file too small: {stem_path}, size={file_size} bytes. "
                    f"Expected >= {min_file_size} bytes."
                )

        # Demucs 必须包含 vocals（我们主要需要的）
        if 'vocals' not in output_paths:
            raise RuntimeError("Demucs separation must include 'vocals' output")

    @staticmethod
    def _extract_task_id(audio_path: str) -> str:
        """从音频路径提取任务 ID"""
        parts = audio_path.split(os.sep)
        if len(parts) >= 2 and parts[-2]:
            return parts[-2]
        return "unknown"


def serve(config: AudioSeparatorConfig):
    """启动 gRPC 服务"""
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=config.max_workers),
        options=[
            ('grpc.max_send_message_length', 100 * 1024 * 1024),  # 100MB
            ('grpc.max_receive_message_length', 100 * 1024 * 1024),  # 100MB
        ],
    )

    servicer = AudioSeparatorServicer(config)
    audioseparator_pb2_grpc.add_AudioSeparatorServicer_to_server(servicer, server)

    server.add_insecure_port(f'[::]:{config.grpc_port}')
    server.start()
    logger.info("AudioSeparator service started on port %s", config.grpc_port)

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down AudioSeparator service...")
        server.stop(grace=5)
