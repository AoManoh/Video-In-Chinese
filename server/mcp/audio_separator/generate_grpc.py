#!/usr/bin/env python3
"""
生成 gRPC Python 代码的脚本

从 proto/audioseparator.proto 生成:
- proto/audioseparator_pb2.py (Protocol Buffers 消息类)
- proto/audioseparator_pb2_grpc.py (gRPC 服务类)
"""

import subprocess
import sys
import os


def generate_grpc_code():
    """生成 gRPC Python 代码"""
    print("Generating gRPC Python code from proto file...")
    
    # 确保在正确的目录
    script_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(script_dir)
    
    # 运行 protoc 命令
    cmd = [
        sys.executable, '-m', 'grpc_tools.protoc',
        '-I./proto',
        '--python_out=./proto',
        '--grpc_python_out=./proto',
        './proto/audioseparator.proto'
    ]
    
    try:
        subprocess.run(cmd, check=True, capture_output=True, text=True)
        print("gRPC code generated successfully.")
        print(f"  - proto/audioseparator_pb2.py")
        print(f"  - proto/audioseparator_pb2_grpc.py")
        
        # 修复导入路径
        fix_import_paths()
        
        return True
    except subprocess.CalledProcessError as e:
        print("Failed to generate gRPC code:")
        print(f"  Error: {e.stderr}")
        return False
    except Exception as e:
        print(f"Unexpected error: {str(e)}")
        return False


def fix_import_paths():
    """修复生成的 gRPC 代码中的导入路径"""
    grpc_file = 'proto/audioseparator_pb2_grpc.py'
    
    if not os.path.exists(grpc_file):
        print(f"Warning: {grpc_file} not found, skipping import path fix")
        return
    
    print("Fixing import paths in generated code...")
    
    try:
        with open(grpc_file, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # 替换导入语句
        old_import = 'import audioseparator_pb2 as audioseparator__pb2'
        new_import = 'from proto import audioseparator_pb2 as audioseparator__pb2'
        
        if old_import in content:
            content = content.replace(old_import, new_import)
            
            with open(grpc_file, 'w', encoding='utf-8') as f:
                f.write(content)
            
            print(f"Import paths fixed in {grpc_file}")
        else:
            print(f"  No import path fix needed in {grpc_file}")
    
    except Exception as e:
        print(f"Warning: Failed to fix import paths: {str(e)}")


if __name__ == '__main__':
    success = generate_grpc_code()
    sys.exit(0 if success else 1)
