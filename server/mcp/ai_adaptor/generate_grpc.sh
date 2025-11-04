#!/bin/bash

# AIAdaptor gRPC 代码生成脚本
# 用途: 从 proto 文件生成 Go gRPC 代码

set -e

echo "=== AIAdaptor gRPC Code Generation ==="

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Please install protoc: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# 检查 protoc-gen-go-grpc 是否安装
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 生成 gRPC 代码
echo "Generating gRPC code from proto/aiadaptor.proto..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/aiadaptor.proto

echo "✓ gRPC code generated successfully"
echo "  - proto/aiadaptor.pb.go (Protocol Buffers messages)"
echo "  - proto/aiadaptor_grpc.pb.go (gRPC service)"

