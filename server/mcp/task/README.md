# Task 服务

**版本**: 1.0  
**服务端口**: 50050  
**技术栈**: Go 1.21+, gRPC, go-redis  
**依赖**: Redis

---

## 服务概述

Task 服务负责任务管理，提供任务创建和状态查询功能。主要职责：

1. **任务创建**：接收临时文件路径，生成任务ID，移动文件到正式目录，推入Redis队列
2. **状态查询**：根据任务ID查询任务状态、结果文件路径、错误信息

---

## 项目结构

```
server/mcp/task/
├── main.go                          # gRPC 服务入口
├── go.mod                           # Go 模块定义
├── go.sum                           # Go 依赖锁定
├── generate_grpc.sh                 # gRPC 代码生成脚本
├── internal/
│   ├── logic/
│   │   ├── create_task_logic.go     # 创建任务逻辑
│   │   └── get_task_status_logic.go # 查询任务状态逻辑
│   ├── storage/
│   │   ├── redis.go                 # Redis 操作封装
│   │   └── file.go                  # 文件操作封装
│   └── svc/
│       └── service_context.go       # 服务上下文（依赖注入）
└── proto/
    ├── task.proto                   # gRPC 接口定义
    ├── task.pb.go                   # 自动生成的 Protobuf 代码
    └── task_grpc.pb.go              # 自动生成的 gRPC 代码
```

---

## 环境变量配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `TASK_GRPC_PORT` | gRPC 服务端口 | `50050` |
| `REDIS_ADDR` | Redis 地址 | `localhost:6379` |
| `REDIS_PASSWORD` | Redis 密码 | 空 |
| `REDIS_DB` | Redis 数据库编号 | `0` |
| `LOCAL_STORAGE_PATH` | 本地存储路径 | `./storage` |

---

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 生成 gRPC 代码

```bash
cd proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    task.proto
```

或使用脚本（Linux/macOS）：

```bash
bash generate_grpc.sh
```

### 3. 编译服务

```bash
go build -o task.exe
```

### 4. 运行服务

```bash
# 设置环境变量（可选）
export TASK_GRPC_PORT=50050
export REDIS_ADDR=localhost:6379
export LOCAL_STORAGE_PATH=./storage

# 启动服务
./task.exe
```

---

## gRPC 接口

### CreateTask

创建新任务。

**请求**：
```protobuf
message CreateTaskRequest {
  string temp_file_path = 1;  // 临时文件路径
}
```

**响应**：
```protobuf
message CreateTaskResponse {
  string task_id = 1;  // 任务ID（UUID v4）
}
```

**执行流程**：
1. 生成任务 ID（UUID v4）
2. 构建正式文件路径（`{LOCAL_STORAGE_PATH}/videos/{task_id}/original.mp4`）
3. 创建任务目录
4. 文件交接（临时文件 → 正式文件，使用 `os.Rename`）
5. 创建任务记录（Redis Hash，初始状态: PENDING）
6. 推入任务到队列（Redis LPUSH, Key: `task:pending`）
7. 返回任务 ID

---

### GetTaskStatus

查询任务状态。

**请求**：
```protobuf
message GetTaskStatusRequest {
  string task_id = 1;  // 任务ID
}
```

**响应**：
```protobuf
message GetTaskStatusResponse {
  TaskStatus status = 1;         // 任务状态
  string result_file_path = 2;   // 结果文件路径（仅当status=COMPLETED时有效）
  string error_message = 3;      // 错误信息（仅当status=FAILED时有效）
  string created_at = 4;         // 创建时间（RFC3339格式）
  string updated_at = 5;         // 更新时间（RFC3339格式）
}
```

**任务状态枚举**：
- `PENDING (0)`: 待处理（已推入队列，等待Processor拉取）
- `PROCESSING (1)`: 处理中（Processor已拉取，正在处理）
- `COMPLETED (2)`: 已完成（处理成功，结果文件已生成）
- `FAILED (3)`: 失败（处理失败，包含错误信息）

---

## Redis 数据结构

### 任务队列

**Key**: `task:pending`  
**类型**: List  
**操作**:
- `LPUSH task:pending {task_id}`: 推入任务到队列头部
- `RPOP task:pending`: 从队列尾部拉取任务（由 Processor 调用）
- `LLEN task:pending`: 查询队列长度

---

### 任务状态

**Key**: `task:{task_id}`  
**类型**: Hash  
**字段**:
- `task_id`: 任务ID
- `status`: 任务状态（PENDING, PROCESSING, COMPLETED, FAILED）
- `original_file_path`: 原始文件路径
- `result_file_path`: 结果文件路径
- `error_message`: 错误信息
- `created_at`: 创建时间（RFC3339格式）
- `updated_at`: 更新时间（RFC3339格式）

---

## 文件移动策略

### 设计决策

使用 `os.Rename` 移动文件，而非 `io.Copy` 复制后删除。

**理由**：
1. **性能优势**: `os.Rename` 是原子操作，仅修改文件系统元数据，不涉及数据复制
2. **避免磁盘空间浪费**: `io.Copy` 需要双倍磁盘空间，`os.Rename` 无此问题
3. **原子性**: `os.Rename` 保证文件移动的原子性，避免中间状态

**性能对比**（1GB文件）：
- `os.Rename`: 耗时 < 1ms，磁盘空间占用 1GB
- `io.Copy + os.Remove`: 耗时 8秒，磁盘空间占用 2GB（复制过程中）

### 跨文件系统降级策略

**问题**: `os.Rename` 仅支持同一文件系统内移动，跨文件系统会失败

**缓解**: 检测 `os.Rename` 错误，如果是跨文件系统错误，降级为 `io.Copy + os.Remove`

---

## 开发状态

- ✅ Phase 1: 基础设施搭建（已完成）
  - ✅ 项目目录结构
  - ✅ gRPC 协议定义和代码生成
  - ✅ Redis 客户端封装
  - ✅ 文件操作封装
  - ✅ 服务上下文
  - ✅ 业务逻辑层（CreateTask, GetTaskStatus）
  - ✅ gRPC 服务入口
  - ✅ 编译验证通过

- ⏳ Phase 2: 存储层实现（待开始）
- ⏳ Phase 3: 业务逻辑实现（待开始）
- ⏳ Phase 4: 测试实现（待开始）
- ⏳ Phase 5: 文档和代码审查（待开始）

---

## 参考文档

- **设计文档**: `notes/server/3rd/Task-design-detail.md` v1.0
- **开发任务清单**: `notes/server/process/development-todo.md`
- **第一层架构**: `notes/server/1st/Base-Design.md` v2.2
- **第二层设计**: `notes/server/2nd/Task-design.md` v1.5

---

## 版本历史

- **v1.0 (2025-11-04)**: 初始版本，完成 Phase 1 基础设施搭建

