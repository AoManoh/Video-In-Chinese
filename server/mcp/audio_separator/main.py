#!/usr/bin/env python3
"""
AudioSeparator 服务入口

启动 gRPC 服务器，监听端口 50052
参考文档: AudioSeparator-design-detail.md v2.0
"""

import sys
import logging

from config import AudioSeparatorConfig, setup_logging
from separator_service import serve


def check_python_version():
    """
    验证 Python 版本要求
    
    要求: Python 3.9+
    原因: Spleeter 和 TensorFlow 2.13+ 要求 Python 3.9+
    """
    if sys.version_info < (3, 9):
        print(f"Error: Python 3.9+ is required. Current version: {sys.version}")
        sys.exit(1)
    
    logging.info(f"Python version check passed: {sys.version}")


def check_system_dependencies():
    """
    验证系统依赖

    要求:
    - ffmpeg: 音频格式转换
    - libsndfile1: 音频文件读写
    """
    import subprocess
    import os

    # 检查 ffmpeg
    try:
        # 尝试使用绝对路径（Windows）
        ffmpeg_paths = [
            'ffmpeg',  # 尝试从 PATH
            r'C:\Environment\ffmpeg-8.0-essentials_build\bin\ffmpeg.exe',  # 常见安装位置
            r'C:\Windows\System32\ffmpeg.exe',  # Windows System32
        ]

        ffmpeg_found = False
        for ffmpeg_cmd in ffmpeg_paths:
            try:
                result = subprocess.run(
                    [ffmpeg_cmd, '-version'],
                    capture_output=True,
                    text=True,
                    timeout=5
                )
                if result.returncode == 0:
                    version_line = result.stdout.split('\n')[0]
                    logging.info(f"ffmpeg check passed: {version_line} (using {ffmpeg_cmd})")
                    ffmpeg_found = True
                    # 将 ffmpeg 路径添加到环境变量，供后续使用
                    if os.path.dirname(ffmpeg_cmd):
                        ffmpeg_dir = os.path.dirname(ffmpeg_cmd)
                        if ffmpeg_dir not in os.environ.get('PATH', ''):
                            os.environ['PATH'] = ffmpeg_dir + os.pathsep + os.environ.get('PATH', '')
                            logging.info(f"Added {ffmpeg_dir} to PATH")
                    break
            except FileNotFoundError:
                continue

        if not ffmpeg_found:
            logging.error("ffmpeg not found in any expected location. Please install ffmpeg.")
            sys.exit(1)

    except Exception as e:
        logging.error(f"ffmpeg check failed with exception: {type(e).__name__}: {str(e)}")
        sys.exit(1)
    
    # 检查 libsndfile1（通过尝试导入 soundfile）
    try:
        import soundfile
        logging.info(f"libsndfile1 check passed: soundfile version {soundfile.__version__}")
    except ImportError:
        logging.error("libsndfile1 not found. Please install libsndfile1.")
        sys.exit(1)


def main():
    """主函数"""
    # 步骤1: 验证 Python 版本
    check_python_version()
    
    # 步骤2: 加载配置
    try:
        config = AudioSeparatorConfig()
    except ValueError as e:
        print(f"Configuration error: {str(e)}")
        sys.exit(1)
    
    # 步骤3: 配置日志
    setup_logging(config)
    logger = logging.getLogger(__name__)
    logger.info("=" * 60)
    logger.info("AudioSeparator Service Starting...")
    logger.info("=" * 60)
    
    # 步骤4: 验证系统依赖
    check_system_dependencies()
    
    # 步骤5: 启动 gRPC 服务器
    try:
        serve(config)
    except Exception as e:
        logger.error(f"Failed to start service: {str(e)}", exc_info=True)
        sys.exit(1)


if __name__ == '__main__':
    main()

