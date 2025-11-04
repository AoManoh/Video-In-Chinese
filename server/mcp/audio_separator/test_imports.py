#!/usr/bin/env python3
"""
测试导入是否正确

验证所有模块可以正确导入（不需要实际运行服务）
"""

import sys
import os

# 添加当前目录到 Python 路径
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

def test_imports():
    """测试所有模块导入"""
    print("=" * 60)
    print("Testing AudioSeparator Service Imports")
    print("=" * 60)
    
    # 测试 1: 导入 proto 模块
    print("\n[1/5] Testing proto imports...")
    try:
        from proto import audioseparator_pb2
        from proto import audioseparator_pb2_grpc
        print("✓ Proto imports successful")
    except ImportError as e:
        print(f"✗ Proto imports failed: {str(e)}")
        return False
    
    # 测试 2: 导入 config 模块
    print("\n[2/5] Testing config import...")
    try:
        from config import AudioSeparatorConfig, setup_logging
        print("✓ Config import successful")
    except ImportError as e:
        print(f"✗ Config import failed: {str(e)}")
        return False
    
    # 测试 3: 测试配置类实例化
    print("\n[3/5] Testing config instantiation...")
    try:
        # 设置测试环境变量
        os.environ['AUDIO_SEPARATOR_GRPC_PORT'] = '50052'
        os.environ['LOG_LEVEL'] = 'info'
        
        config = AudioSeparatorConfig()
        print(f"✓ Config instantiation successful")
        print(f"  - gRPC Port: {config.grpc_port}")
        print(f"  - Model Name: {config.model_name}")
        print(f"  - Max Workers: {config.max_workers}")
        print(f"  - Timeout: {config.timeout_seconds}s")
    except Exception as e:
        print(f"✗ Config instantiation failed: {str(e)}")
        return False
    
    # 测试 4: 导入 spleeter_wrapper（可能会失败，因为 spleeter 未安装）
    print("\n[4/5] Testing spleeter_wrapper import...")
    try:
        from spleeter_wrapper import SpleeterWrapper
        print("✓ SpleeterWrapper import successful")
    except ImportError as e:
        print(f"⚠ SpleeterWrapper import failed (expected if spleeter not installed): {str(e)}")
        print("  This is OK for import testing, but required for actual service")
    
    # 测试 5: 导入 separator_service
    print("\n[5/5] Testing separator_service import...")
    try:
        from separator_service import AudioSeparatorServicer, serve
        print("✓ SeparatorService import successful")
    except ImportError as e:
        print(f"✗ SeparatorService import failed: {str(e)}")
        return False
    
    # 测试 6: 验证 gRPC 消息类
    print("\n[6/6] Testing gRPC message classes...")
    try:
        # 创建测试请求
        request = audioseparator_pb2.SeparateAudioRequest(
            audio_path="/test/audio.wav",
            output_dir="/test/output",
            stems=2
        )
        print(f"✓ SeparateAudioRequest created: audio_path={request.audio_path}")
        
        # 创建测试响应
        response = audioseparator_pb2.SeparateAudioResponse(
            success=True,
            vocals_path="/test/vocals.wav",
            accompaniment_path="/test/accompaniment.wav",
            processing_time_ms=1000
        )
        print(f"✓ SeparateAudioResponse created: success={response.success}")
    except Exception as e:
        print(f"✗ gRPC message creation failed: {str(e)}")
        return False
    
    print("\n" + "=" * 60)
    print("✓ All import tests passed!")
    print("=" * 60)
    print("\nNext steps:")
    print("1. Install Python dependencies: pip install -r requirements.txt")
    print("2. Install system dependencies: ffmpeg, libsndfile1")
    print("3. Run the service: python main.py")
    print("=" * 60)
    
    return True


if __name__ == '__main__':
    success = test_imports()
    sys.exit(0 if success else 1)

