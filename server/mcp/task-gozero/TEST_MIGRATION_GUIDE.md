# Task 服务测试迁移交接文档

**文档版本**: 1.0  
**创建日期**: 2025-11-05  
**目标框架**: go-zero v1.9.2  
**预计完成时间**: P0 (1-2小时), P1 (1-2小时), P2 (2-3小时)  
**迁移负责人**: [待分配]

---

## 📋 目录

1. [测试迁移现状](#1-测试迁移现状)
2. [优先级分级的测试迁移计划](#2-优先级分级的测试迁移计划)
3. [测试环境搭建指南](#3-测试环境搭建指南)
4. [测试迁移步骤](#4-测试迁移步骤)
5. [验收标准](#5-验收标准)
6. [预计工作量](#6-预计工作量)

---

## 1. 测试迁移现状

### 1.1 已完成内容

**业务逻辑迁移** (Phase 4.2 已完成):

1. **CreateTask 7步工作流程** (`internal/logic/createTaskLogic.go`, 143行):
   - 生成任务ID (UUID v4)
   - 构建正式文件路径
   - 创建任务目录
   - 文件交接 (临时→正式)
   - 创建Redis记录 (Hash)
   - 推入Redis队列 (LPUSH)
   - 返回任务ID

2. **GetTaskStatus 3步查询流程** (`internal/logic/getTaskStatusLogic.go`, 137行):
   - 读取Redis状态 (HGETALL)
   - 检查任务存在性
   - 返回任务状态

**质量验证**:
- ✅ `go mod tidy`: 通过
- ✅ `go vet ./...`: 通过 (无警告)
- ✅ `go build`: 通过 (生成 task.exe)
- ✅ 代码注释: 完整的 GoDoc 风格

### 1.2 待迁移测试用例清单

**原测试代码位置**: `server/mcp/task-backup/internal/`

| 测试文件 | 测试用例数 | 代码行数 | 说明 |
|---------|-----------|---------|------|
| `storage/file_test.go` | 12 | ~300 | 文件操作单元测试 |
| `storage/redis_test.go` | 11 | ~350 | Redis 操作单元测试 |
| `logic/create_task_logic_test.go` | 6 | ~200 | CreateTask 逻辑测试 |
| `logic/get_task_status_logic_test.go` | 7 | ~250 | GetTaskStatus 逻辑测试 |
| `integration_test.go` | 5 | ~300 | 集成测试 |
| **总计** | **41** | **~1400** | - |

**测试类型分布**:
- 单元测试: 36 个 (88%)
- 集成测试: 5 个 (12%)

**外部依赖**:
- Docker Desktop (用于 testcontainers-go 启动 Redis 容器)
- Redis 7-alpine 镜像

---

## 2. 优先级分级的测试迁移计划

### 2.1 P0 (立即执行) - 核心业务逻辑测试

**目标**: 验证 GoZero 框架适配的正确性

**测试范围**:

| 测试文件 | 测试用例 | 说明 |
|---------|---------|------|
| `create_task_logic_test.go` | 6 个 | CreateTask 7步工作流程验证 |
| `get_task_status_logic_test.go` | 7 个 | GetTaskStatus 3步查询流程验证 |
| **小计** | **13 个** | - |

**测试用例详细清单**:

**CreateTask 逻辑测试** (6个):
1. `TestCreateTaskLogic_Success` - 正常流程测试
2. `TestCreateTaskLogic_InvalidTempFilePath` - 无效临时文件路径
3. `TestCreateTaskLogic_FileMoveFailed` - 文件移动失败
4. `TestCreateTaskLogic_RedisSetFailed` - Redis 写入失败
5. `TestCreateTaskLogic_RedisPushFailed` - Redis 队列推送失败
6. `TestCreateTaskLogic_DirectoryCreationFailed` - 目录创建失败

**GetTaskStatus 逻辑测试** (7个):
1. `TestGetTaskStatusLogic_Success_Pending` - 查询 PENDING 状态
2. `TestGetTaskStatusLogic_Success_Processing` - 查询 PROCESSING 状态
3. `TestGetTaskStatusLogic_Success_Completed` - 查询 COMPLETED 状态
4. `TestGetTaskStatusLogic_Success_Failed` - 查询 FAILED 状态
5. `TestGetTaskStatusLogic_TaskNotFound` - 任务不存在
6. `TestGetTaskStatusLogic_RedisGetFailed` - Redis 读取失败
7. `TestGetTaskStatusLogic_InvalidStatus` - 无效状态字符串

**预计工作量**: 1-2 小时

### 2.2 P1 (建议执行) - 集成测试

**目标**: 验证完整的 gRPC 服务流程

**测试范围**:

| 测试文件 | 测试用例 | 说明 |
|---------|---------|------|
| `integration_test.go` | 5 个 | 端到端集成测试 |

**测试用例详细清单**:

1. `TestIntegration_CreateAndGetTask` - 创建任务并查询状态
2. `TestIntegration_MultipleTasksSequential` - 顺序创建多个任务
3. `TestIntegration_TaskStatusTransition` - 任务状态转换
4. `TestIntegration_FileHandoff` - 文件交接验证
5. `TestIntegration_RedisQueueOperation` - Redis 队列操作验证

**预计工作量**: 1-2 小时

### 2.3 P2 (可选) - 存储层单元测试

**目标**: 验证存储层实现的正确性

**测试范围**:

| 测试文件 | 测试用例 | 说明 |
|---------|---------|------|
| `storage/file_test.go` | 12 个 | 文件操作单元测试 |
| `storage/redis_test.go` | 11 个 | Redis 操作单元测试 |
| **小计** | **23 个** | - |

**说明**: 这些测试已在原版本验证通过，存储层逻辑未变，可选择性迁移。

**预计工作量**: 2-3 小时

---

## 3. 测试环境搭建指南

### 3.1 Docker Desktop 安装和配置

**Windows 系统**:

1. 下载 Docker Desktop: https://www.docker.com/products/docker-desktop
2. 安装并启动 Docker Desktop
3. 验证安装:

```powershell
docker --version
# 输出: Docker version 24.0.x, build xxxxx
```

**Linux 系统**:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install docker.io

# 启动 Docker 服务
sudo systemctl start docker
sudo systemctl enable docker

# 验证安装
docker --version
```

### 3.2 testcontainers-go 使用说明

**依赖安装**:

```bash
cd server/mcp/task-gozero
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/wait
```

**基本用法**:

```go
import (
	"context"
	"testing"
	
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupRedisContainer(t *testing.T) (string, func()) {
	ctx := context.Background()
	
	// 创建 Redis 容器
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start Redis container: %v", err)
	}
	
	// 获取容器地址
	host, err := redisC.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	
	port, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}
	
	redisAddr := fmt.Sprintf("%s:%s", host, port.Port())
	
	// 返回地址和清理函数
	cleanup := func() {
		if err := redisC.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
	
	return redisAddr, cleanup
}
```

### 3.3 Redis 测试容器启动脚本

**测试辅助函数** (`internal/logic/test_helpers.go`):

```go
package logic

import (
	"context"
	"fmt"
	"testing"
	
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zeromicro/go-zero/core/stores/redis"
	
	"video-in-chinese/task/internal/config"
	"video-in-chinese/task/internal/svc"
)

// SetupTestRedis 启动 Redis 测试容器并返回 ServiceContext
func SetupTestRedis(t *testing.T) (*svc.ServiceContext, func()) {
	ctx := context.Background()
	
	// 启动 Redis 容器
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start Redis container: %v", err)
	}
	
	// 获取 Redis 地址
	host, _ := redisC.Host(ctx)
	port, _ := redisC.MappedPort(ctx, "6379")
	redisAddr := fmt.Sprintf("%s:%s", host, port.Port())
	
	// 创建测试配置
	c := config.Config{
		Redis: redis.RedisConf{
			Host: redisAddr,
			Type: "node",
		},
		LocalStoragePath: t.TempDir(), // 使用临时目录
	}
	
	// 创建 ServiceContext
	svcCtx := svc.NewServiceContext(c)
	
	// 清理函数
	cleanup := func() {
		if err := redisC.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate Redis container: %v", err)
		}
	}
	
	return svcCtx, cleanup
}
```

---

## 4. 测试迁移步骤

### 4.1 P0 测试迁移 (CreateTask 和 GetTaskStatus 逻辑测试)

**步骤 1**: 创建测试文件

```bash
cd server/mcp/task-gozero/internal/logic
touch createTaskLogic_test.go
touch getTaskStatusLogic_test.go
```

**步骤 2**: 复制原测试代码

```bash
# 从备份目录复制测试代码
cp ../../task-backup/internal/logic/create_task_logic_test.go createTaskLogic_test.go
cp ../../task-backup/internal/logic/get_task_status_logic_test.go getTaskStatusLogic_test.go
```

**步骤 3**: 适配 go-zero 框架

**关键适配点**:

1. **导入路径更新**:

```go
// 原代码
import (
	"video-in-chinese/task/internal/svc"
	"video-in-chinese/task/proto"
)

// 新代码 (相同，无需修改)
import (
	"video-in-chinese/task/internal/svc"
	"video-in-chinese/task/proto"
)
```

2. **Redis 客户端适配**:

```go
// 原代码 (go-redis v9)
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
	Addr: redisAddr,
})

// 新代码 (go-zero redis.Redis)
import "github.com/zeromicro/go-zero/core/stores/redis"

rdb := redis.MustNewRedis(redis.RedisConf{
	Host: redisAddr,
	Type: "node",
})
```

3. **ServiceContext 创建**:

```go
// 原代码
svcCtx := &svc.ServiceContext{
	RedisClient: redisClient,
	FileStorage: fileStorage,
}

// 新代码 (使用配置创建)
c := config.Config{
	Redis: redis.RedisConf{
		Host: redisAddr,
		Type: "node",
	},
	LocalStoragePath: t.TempDir(),
}
svcCtx := svc.NewServiceContext(c)
```

4. **日志系统适配**:

```go
// 原代码
import "log"
log.Printf("test message")

// 新代码
import "github.com/zeromicro/go-zero/core/logx"
logx.Infof("test message")
```

**步骤 4**: 运行测试

```bash
# 运行单个测试文件
go test -v ./internal/logic/createTaskLogic_test.go ./internal/logic/createTaskLogic.go

# 运行所有 logic 测试
go test -v ./internal/logic/...

# 运行测试并显示覆盖率
go test -v -cover ./internal/logic/...
```

### 4.2 完整迁移示例 (TestCreateTaskLogic_Success)

**原测试代码** (`task-backup/internal/logic/create_task_logic_test.go`):

```go
func TestCreateTaskLogic_Success(t *testing.T) {
	// 启动 Redis 容器
	redisAddr, cleanup := setupRedisContainer(t)
	defer cleanup()
	
	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()
	
	// 创建 ServiceContext
	svcCtx := &svc.ServiceContext{
		RedisClient: storage.NewRedisClient(rdb),
		FileStorage: storage.NewFileStorage(t.TempDir()),
	}
	
	// 创建临时文件
	tempFile := createTempFile(t)
	
	// 创建 Logic
	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	
	// 调用 CreateTask
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{
		TempFilePath: tempFile,
	})
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.TaskId)
	
	// 验证 Redis 记录
	fields, err := rdb.HGetAll(context.Background(), "task:"+resp.TaskId).Result()
	assert.NoError(t, err)
	assert.Equal(t, "PENDING", fields["status"])
}
```

**迁移后的测试代码** (`task-gozero/internal/logic/createTaskLogic_test.go`):

```go
package logic

import (
	"context"
	"os"
	"testing"
	
	"github.com/stretchr/testify/assert"
	
	"video-in-chinese/task/proto"
)

func TestCreateTaskLogic_Success(t *testing.T) {
	// 使用测试辅助函数启动 Redis 容器
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()
	
	// 创建临时文件
	tempFile := createTempFile(t, svcCtx.Config.LocalStoragePath)
	defer os.Remove(tempFile)
	
	// 创建 Logic
	logic := NewCreateTaskLogic(context.Background(), svcCtx)
	
	// 调用 CreateTask
	resp, err := logic.CreateTask(&proto.CreateTaskRequest{
		TempFilePath: tempFile,
	})
	
	// 验证结果
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.TaskId)
	
	// 验证 Redis 记录
	fields, err := svcCtx.RedisClient.Hgetall("task:" + resp.TaskId)
	assert.NoError(t, err)
	assert.Equal(t, "PENDING", fields["status"])
	
	// 验证文件已移动
	originalFilePath := svcCtx.FileStorage.GetOriginalFilePath(resp.TaskId)
	_, err = os.Stat(originalFilePath)
	assert.NoError(t, err, "Original file should exist")
}

// createTempFile 创建临时测试文件
func createTempFile(t *testing.T, baseDir string) string {
	tempFile, err := os.CreateTemp(baseDir, "test-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	// 写入一些测试数据
	_, err = tempFile.WriteString("test video content")
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	
	tempFile.Close()
	return tempFile.Name()
}
```

**关键变更点**:

1. ✅ 使用 `SetupTestRedis()` 辅助函数
2. ✅ 使用 `svcCtx.RedisClient.Hgetall()` (go-zero API)
3. ✅ 使用 `svcCtx.Config.LocalStoragePath` 获取存储路径
4. ✅ 添加文件移动验证

### 4.3 GetTaskStatus 测试迁移示例

**迁移后的测试代码** (`task-gozero/internal/logic/getTaskStatusLogic_test.go`):

```go
package logic

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	
	"video-in-chinese/task/proto"
)

func TestGetTaskStatusLogic_Success_Pending(t *testing.T) {
	// 启动 Redis 容器
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()
	
	// 准备测试数据
	taskID := "test-task-id-123"
	now := time.Now().Format(time.RFC3339)
	
	// 在 Redis 中创建任务记录
	err := svcCtx.RedisClient.Hset("task:"+taskID, "task_id", taskID)
	assert.NoError(t, err)
	err = svcCtx.RedisClient.Hset("task:"+taskID, "status", "PENDING")
	assert.NoError(t, err)
	err = svcCtx.RedisClient.Hset("task:"+taskID, "created_at", now)
	assert.NoError(t, err)
	err = svcCtx.RedisClient.Hset("task:"+taskID, "updated_at", now)
	assert.NoError(t, err)
	
	// 创建 Logic
	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	
	// 调用 GetTaskStatus
	resp, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{
		TaskId: taskID,
	})
	
	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, proto.TaskStatus_PENDING, resp.Status)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestGetTaskStatusLogic_TaskNotFound(t *testing.T) {
	// 启动 Redis 容器
	svcCtx, cleanup := SetupTestRedis(t)
	defer cleanup()
	
	// 创建 Logic
	logic := NewGetTaskStatusLogic(context.Background(), svcCtx)
	
	// 调用 GetTaskStatus (任务不存在)
	_, err := logic.GetTaskStatus(&proto.GetTaskStatusRequest{
		TaskId: "non-existent-task-id",
	})
	
	// 验证错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}
```

---

## 5. 验收标准

### 5.1 P0 测试验收标准

**必须满足**:
- ✅ 13 个测试用例全部通过
- ✅ 业务逻辑层覆盖率 > 80%
- ✅ `go vet ./...` 无警告
- ✅ `gofmt -s -w .` 格式正确

**验证命令**:

```bash
# 1. 运行 P0 测试
go test -v ./internal/logic/createTaskLogic_test.go ./internal/logic/getTaskStatusLogic_test.go

# 2. 检查覆盖率
go test -v -coverprofile=coverage.out ./internal/logic/
go tool cover -func=coverage.out | grep total

# 3. 静态检查
go vet ./...

# 4. 格式检查
gofmt -s -w .
```

### 5.2 P1 测试验收标准

**必须满足**:
- ✅ 5 个集成测试用例全部通过
- ✅ 端到端流程验证通过
- ✅ Redis 队列操作正确

### 5.3 P2 测试验收标准

**可选满足**:
- ✅ 23 个存储层测试用例全部通过
- ✅ 存储层覆盖率 > 80%

---

## 6. 预计工作量

| 优先级 | 测试用例数 | 预计工作量 | 说明 |
|-------|-----------|-----------|------|
| **P0** | 13 个 | 1-2 小时 | 核心业务逻辑测试，必须完成 |
| **P1** | 5 个 | 1-2 小时 | 集成测试，建议完成 |
| **P2** | 23 个 | 2-3 小时 | 存储层测试，可选完成 |
| **总计** | 41 个 | 4-7 小时 | 完整测试迁移 |

**建议执行顺序**:
1. P0 测试 (立即执行)
2. P1 测试 (建议执行)
3. P2 测试 (时间允许时执行)

---

**文档维护者**: 开发团队  
**最后更新**: 2025-11-05  
**反馈渠道**: 请在项目 Issue 中提交文档问题或改进建议

