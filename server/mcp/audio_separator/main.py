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
    
    # 检查 ffmpeg
    try:
        result = subprocess.run(
            ['ffmpeg', '-version'],
            capture_output=True,
            text=True,
            timeout=5
        )
        if result.returncode == 0:
            version_line = result.stdout.split('\n')[0]
            logging.info(f"ffmpeg check passed: {version_line}")
        else:
            logging.warning("ffmpeg check failed: command returned non-zero exit code")
    except FileNotFoundError:
        logging.error("ffmpeg not found. Please install ffmpeg.")
        sys.exit(1)
    except Exception as e:
        logging.warning(f"ffmpeg check failed: {str(e)}")
    
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

