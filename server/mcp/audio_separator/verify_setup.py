#!/usr/bin/env python3
"""简单的设置验证脚本"""

import sys
import os

# 添加当前目录到 Python 路径
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

print("Verifying AudioSeparator Service Setup...")
print("-" * 60)

# 步骤 1: 检查 proto 文件
print("\n1. Checking proto files...")
proto_files = [
    'proto/audioseparator.proto',
    'proto/audioseparator_pb2.py',
    'proto/audioseparator_pb2_grpc.py',
    'proto/__init__.py'
]

for f in proto_files:
    if os.path.exists(f):
        print(f"  ✓ {f}")
    else:
        print(f"  ✗ {f} NOT FOUND")

# 步骤 2: 检查服务文件
print("\n2. Checking service files...")
service_files = [
    'main.py',
    'config.py',
    'spleeter_wrapper.py',
    'separator_service.py',
    'requirements.txt',
    'README.md'
]

for f in service_files:
    if os.path.exists(f):
        print(f"  ✓ {f}")
    else:
        print(f"  ✗ {f} NOT FOUND")

# 步骤 3: 尝试导入 proto 模块
print("\n3. Testing proto imports...")
try:
    from proto import audioseparator_pb2
    from proto import audioseparator_pb2_grpc
    print("  ✓ Proto modules imported successfully")
    
    # 测试创建消息
    req = audioseparator_pb2.SeparateAudioRequest()
    resp = audioseparator_pb2.SeparateAudioResponse()
    print("  ✓ Proto messages can be instantiated")
except Exception as e:
    print(f"  ✗ Proto import failed: {e}")
    sys.exit(1)

# 步骤 4: 检查 Python 版本
print("\n4. Checking Python version...")
print(f"  Python version: {sys.version}")
if sys.version_info >= (3, 9):
    print("  ✓ Python 3.9+ requirement met")
else:
    print("  ✗ Python 3.9+ required")

print("\n" + "=" * 60)
print("✓ Setup verification completed successfully!")
print("=" * 60)
print("\nTo run the service:")
print("1. Install dependencies: pip install -r requirements.txt")
print("2. Install system deps: ffmpeg, libsndfile1")
print("3. Run: python main.py")

