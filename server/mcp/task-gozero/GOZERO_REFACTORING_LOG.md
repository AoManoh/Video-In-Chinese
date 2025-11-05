# Task 服务 go-zero 框架重构日志

**创建日期**: 2025-11-05
**重构状态**: 进行中 (Phase 4.1 已完成)

---

## 重构原因

发现架构偏差：Task 服务使用原生 gRPC 实现，但 Base-Design.md v2.2 明确要求使用 go-zero 框架。

**架构要求** (Base-Design.md v2.2 第127行):
```markdown
* **后端语言与框架**:
  * **Go**: GoZero（Gateway、Task、Processor 服务）
  * **Python**: gRPC + TensorFlow（Audio-Separator 服务）
```

---

## 重构范围

### 技术栈变更

| 组件 | 原实现 | go-zero 实现 |
|------|--------|--------------|
| 项目结构 | 自定义目录结构 | go-zero 标准结构 (goctl 生成) |
| 配置管理 | 环境变量 | YAML 配置文件 (etc/task.yaml) |
| 依赖注入 | 手动管理 | ServiceContext 模式 |
| 日志系统 | 标准库 log | go-zero logx |
| Redis 客户端 | go-redis v9 | go-zero redis.Redis |
| gRPC 代码生成 | protoc | goctl rpc protoc |

### 保留内容

- **业务逻辑**: CreateTask 7步工作流程、GetTaskStatus 3步查询流程
- **测试用例**: 全部 41 个测试用例 (需迁移到 go-zero 环境)
- **数据结构**: Redis 队列 (task:pending) 和 Hash (task:{task_id}) 结构保持不变

---

## Phase 4.0: 准备阶段 ✅ 已完成

### 任务清单
- [x] 备份现有代码到 `server/mcp/task-backup/`
- [x] 安装 goctl 工具 (v1.9.2)
- [x] 创建重构检查清单文档 (REFACTORING_CHECKLIST.md)

### 完成时间
2025-11-05

---

## Phase 4.1: 基础设施搭建 ✅ 已完成

### 任务清单
- [x] 使用 goctl 生成 go-zero 项目结构
- [x] 配置 `etc/task.yaml` (Redis + LocalStoragePath)
- [x] 更新 `internal/config/config.go` (添加 Redis 和 LocalStoragePath 字段)
- [x] 创建存储层 `internal/storage/redis.go` (使用 go-zero redis.Redis)
- [x] 创建存储层 `internal/storage/file.go` (使用 go-zero logx)
- [x] 更新 `internal/svc/serviceContext.go` (集成 RedisClient 和 FileStorage)
- [x] 修复 proto 文件 go_package 选项 (从 `video-in-chinese/task/proto` 改为 `./proto`)
- [x] 重新生成 gRPC 代码 (确保导入路径正确)
- [x] 验证项目编译 (go mod tidy, go vet, go build 全部通过)

### 技术要点

#### 1. go-zero redis.Redis API 差异

**问题**: go-zero redis.Redis 的 API 与 go-redis v9 不同

**解决方案**:
- `Ping()` 返回 `bool` 而不是 `error`
  ```go
  // 原实现 (go-redis v9)
  _, err := client.Ping(ctx).Result()
  
  // go-zero 实现
  ok := client.Ping()
  if !ok {
      return nil, fmt.Errorf("failed to connect to Redis")
  }
  ```

- `HmsetCtx()` 需要 `map[string]string` 而不是 `map[string]interface{}`
  ```go
  // 在 SetTaskFields 中添加类型转换
  stringFields := make(map[string]string, len(fields))
  for k, v := range fields {
      stringFields[k] = fmt.Sprintf("%v", v)
  }
  err := r.client.HmsetCtx(ctx, key, stringFields)
  ```

#### 2. goctl 生成的文件不可修改

**问题**: goctl 生成的文件带有 "DO NOT EDIT" 标记,不应手动修改

**受影响文件**:
- `internal/server/taskServiceServer.go`
- `taskservice/taskService.go`
- `proto/task.pb.go`
- `proto/task_grpc.pb.go`

**解决方案**: 修改 proto 文件的 `go_package` 选项,然后重新生成代码

#### 3. proto 文件 go_package 路径问题

**问题**: goctl 根据 `go_package` 生成嵌套目录,导致导入路径错误

**原配置**:
```proto
option go_package = "video-in-chinese/task/proto";
```

**修改后**:
```proto
option go_package = "./proto";
```

**结果**: 生成的导入路径正确 (`video-in-chinese/task/proto`)

#### 4. client 目录问题

**问题**: goctl 生成的 `client/taskservice/taskService.go` 文件导入路径错误

**解决方案**: 删除 client 目录 (服务端不需要客户端代码)

### 项目结构

```
server/mcp/task-gozero/
├── etc/
│   └── task.yaml (Redis + LocalStoragePath 配置)
├── internal/
│   ├── config/
│   │   └── config.go (添加了 Redis 和 LocalStoragePath 字段)
│   ├── logic/
│   │   ├── createTaskLogic.go (待实现业务逻辑)
│   │   └── getTaskStatusLogic.go (待实现业务逻辑)
│   ├── server/
│   │   └── taskServiceServer.go (goctl 生成,不可修改)
│   ├── storage/
│   │   ├── file.go (使用 logx,从 config 获取 baseDir)
│   │   └── redis.go (使用 go-zero redis.Redis)
│   └── svc/
│       └── serviceContext.go (集成 RedisClient 和 FileStorage)
├── proto/
│   ├── task.pb.go
│   └── task_grpc.pb.go
├── taskservice/
│   └── taskService.go (goctl 生成,不可修改)
├── task.go (主程序入口)
├── task.proto (go_package = "./proto")
├── go.mod (module: video-in-chinese/task)
└── go.sum
```

### 质量验证

- `go mod tidy`: ✅ 通过
- `go vet ./...`: ✅ 通过 (无警告)
- `go build`: ✅ 通过 (编译成功)
- `gofmt -s -w .`: ✅ 通过

### 完成时间
2025-11-05

---

## Phase 4.2: 业务逻辑迁移 - ✅ 已完成

### 任务清单
- [x] 迁移 CreateTask 7步工作流程到 `internal/logic/createTaskLogic.go`
- [x] 迁移 GetTaskStatus 3步查询流程到 `internal/logic/getTaskStatusLogic.go`
- [x] 更新 godoc 注释为 go-zero 风格
- [x] 验证编译通过 (go build, go vet)

### 迁移内容

#### CreateTask 7步工作流程
1. 生成任务ID (UUID v4)
2. 构建正式文件路径
3. 创建任务目录
4. 文件交接 (临时→正式)
5. 创建Redis记录 (Hash)
6. 推入Redis队列 (LPUSH)
7. 返回任务ID

#### GetTaskStatus 3步查询流程
1. 读取Redis状态 (HGETALL)
2. 检查任务存在性
3. 返回任务状态

### 适配要点
- 使用 `logx.Logger` 替代标准库 `log`
- 使用 `l.svcCtx` 访问依赖 (RedisClient, FileStorage)
- 使用 `l.Infof()` / `l.Errorf()` 记录日志
- 保持原有业务逻辑和错误处理不变
- 保持原有Redis数据结构不变

### 质量验证
- `go mod tidy`: ✅ 通过
- `go vet ./...`: ✅ 通过 (无警告)
- `go build`: ✅ 通过 (编译成功，生成 task.exe)
- 代码注释: ✅ 完整的 GoDoc 风格注释

### 完成时间
2025-11-05

---

## Phase 4.3: 测试迁移和验证 - [ ] 待开始

### 任务清单
- [ ] 迁移文件存储测试 (12 个测试用例)
- [ ] 迁移 Redis 操作测试 (11 个测试用例)
- [ ] 迁移 CreateTask 逻辑测试 (6 个测试用例)
- [ ] 迁移 GetTaskStatus 逻辑测试 (7 个测试用例)
- [ ] 迁移集成测试 (5 个测试用例)
- [ ] 生成覆盖率报告 (目标: 41/41 通过, >80% 业务逻辑覆盖率)
- [ ] 更新测试文档

---

## Phase 4.4: 文档更新 - [ ] 待开始

### 任务清单
- [ ] 更新 API_DOCUMENTATION.md (go-zero 版本)
- [ ] 更新 CODE_REVIEW_REPORT.md (go-zero 版本)
- [ ] 更新 README.md (go-zero 版本)
- [ ] 创建 MIGRATION_TO_GOZERO.md (迁移文档)

---

## Phase 4.5: 清理和验收 - [ ] 待开始

### 任务清单
- [ ] 替换旧实现 (重命名 task-gozero 为 task)
- [ ] 最终质量检查 (go mod tidy, gofmt, go vet, go test, go build)
- [ ] 更新 development-todo.md 记录重构过程
- [ ] 生成最终完成报告

---

## go-zero 开发规范 (学习笔记)

### 1. RPC 服务开发流程
1. 使用 `goctl rpc protoc` 生成代码
2. 不要手动修改带 "DO NOT EDIT" 标记的文件
3. proto 文件的 `go_package` 应使用相对路径 (如 `./proto`)
4. Logic 层实现业务逻辑,通过 `l.svcCtx` 访问依赖
5. ServiceContext 用于依赖注入,在 `NewServiceContext` 中初始化所有依赖
6. 使用 `logx.WithContext(ctx)` 记录日志
7. Config 结构体继承 `zrpc.RpcServerConf` 并添加自定义配置字段

### 2. 项目结构规范
- `etc/`: 配置文件 (YAML 格式)
- `internal/config/`: 配置结构体定义
- `internal/logic/`: 业务逻辑实现
- `internal/server/`: gRPC 服务器实现 (goctl 生成,不可修改)
- `internal/svc/`: 服务上下文 (依赖注入)
- `proto/`: Protocol Buffers 定义和生成的代码
- `taskservice/`: 服务接口定义 (goctl 生成,不可修改)

### 3. 日志规范
- 使用 `logx.Infof()` 记录信息日志
- 使用 `logx.WithContext(ctx).Infof()` 记录带上下文的日志
- 使用 `logx.Errorf()` 记录错误日志

### 4. 配置管理规范
- 配置文件使用 YAML 格式
- Config 结构体继承 `zrpc.RpcServerConf`
- 自定义配置字段添加到 Config 结构体中
- 使用 `conf.MustLoad()` 加载配置

---

## 参考资料

- go-zero 官方文档: https://go-zero.dev/
- go-zero GitHub: https://github.com/zeromicro/go-zero
- goctl 工具文档: https://go-zero.dev/docs/tutorials/cli/overview
- Base-Design.md v2.2 (项目架构设计文档)
- Task-design-detail.md v1.0 (Task 服务详细设计文档)

