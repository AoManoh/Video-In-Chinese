"""
AudioSeparator gRPC 客户端测试脚本

用于测试 AudioSeparator 服务的音频分离功能
"""

import grpc
import sys
import os
from pathlib import Path

# 添加 proto 目录到路径
sys.path.insert(0, str(Path(__file__).parent))

from proto import audioseparator_pb2
from proto import audioseparator_pb2_grpc


def test_separate_audio(audio_path: str, output_dir: str, stems: int = 4):
    """
    测试音频分离功能
    
    Args:
        audio_path: 输入音频文件路径
        output_dir: 输出目录路径
        stems: 分离模式 (2 或 4)
    """
    # 转换为绝对路径
    audio_path = os.path.abspath(audio_path)
    output_dir = os.path.abspath(output_dir)
    
    print("=" * 60)
    print("AudioSeparator 客户端测试")
    print("=" * 60)
    print(f"输入音频: {audio_path}")
    print(f"输出目录: {output_dir}")
    print(f"分离模式: {stems} stems")
    print()
    
    # 检查输入文件
    if not os.path.exists(audio_path):
        print(f"错误: 输入文件不存在: {audio_path}")
        return False
    
    # 创建输出目录
    os.makedirs(output_dir, exist_ok=True)
    
    # 连接到 gRPC 服务
    channel = grpc.insecure_channel('localhost:50052')
    stub = audioseparator_pb2_grpc.AudioSeparatorStub(channel)
    
    # 创建请求
    request = audioseparator_pb2.SeparateAudioRequest(
        audio_path=audio_path,
        output_dir=output_dir,
        stems=stems
    )
    
    print("正在连接到 AudioSeparator 服务...")
    print("正在分离音频（这可能需要几分钟）...")
    print()
    
    try:
        # 调用服务
        response = stub.SeparateAudio(request)
        
        # 显示结果
        print("=" * 60)
        print("分离结果")
        print("=" * 60)
        print(f"成功: {response.success}")
        print(f"处理时间: {response.processing_time_ms} ms ({response.processing_time_ms / 1000:.2f} 秒)")
        print()
        
        if response.success:
            print("输出文件:")
            print(f"  人声 (vocals): {response.vocals_path}")
            print(f"  伴奏 (accompaniment): {response.accompaniment_path}")
            print()
            
            if response.stems:
                print("所有 stems:")
                for stem in response.stems:
                    print(f"  {stem.name}: {stem.path}")
                    # 检查文件是否存在
                    if os.path.exists(stem.path):
                        file_size = os.path.getsize(stem.path) / (1024 * 1024)  # MB
                        print(f"    文件大小: {file_size:.2f} MB")
                    else:
                        print(f"    警告: 文件不存在!")
            
            print()
            print("=" * 60)
            print("测试成功!")
            print("=" * 60)
            return True
        else:
            print(f"分离失败: {response.error_message}")
            return False
            
    except grpc.RpcError as e:
        print(f"gRPC 错误: {e.code()}")
        print(f"详细信息: {e.details()}")
        return False
    except Exception as e:
        print(f"错误: {str(e)}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        channel.close()


def main():
    """主函数"""
    if len(sys.argv) < 3:
        print("用法: python test_client.py <audio_path> <output_dir> [stems]")
        print("示例: python test_client.py data/videos/xxx/intermediate/original_audio.wav data/videos/xxx/intermediate/separated 4")
        sys.exit(1)
    
    audio_path = sys.argv[1]
    output_dir = sys.argv[2]
    stems = int(sys.argv[3]) if len(sys.argv) > 3 else 4
    
    success = test_separate_audio(audio_path, output_dir, stems)
    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()

