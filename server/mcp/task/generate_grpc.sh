#!/bin/bash

# Task 服务 gRPC 代码生成脚本
# 使用 protoc 生成 Go 代码

echo "Generating gRPC code for Task service..."

# 进入 proto 目录
cd proto

# 生成 Go 代码
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    task.proto

echo "✓ gRPC code generated successfully"
echo "  - task.pb.go (Protobuf messages)"
echo "  - task_grpc.pb.go (gRPC service)"

