# AIAdaptor 服务 GoZero 框架迁移指南

**文档版本**: 1.0  
**创建日期**: 2025-11-05  
**目标框架**: go-zero v1.9.2  
**预计完成时间**: 16-20 工时  
**迁移负责人**: [待分配]

---

## 📋 目录

1. [迁移背景和原因](#1-迁移背景和原因)
2. [迁移范围和影响分析](#2-迁移范围和影响分析)
3. [分阶段迁移计划](#3-分阶段迁移计划)
4. [技术适配指南](#4-技术适配指南)
5. [质量保证要求](#5-质量保证要求)
6. [风险和注意事项](#6-风险和注意事项)
7. [参考资料](#7-参考资料)

---

## 1. 迁移背景和原因

### 1.1 架构要求

根据 `notes/server/1st/Base-Design.md` v2.2（第 127 行）明确要求：

```markdown
* **后端语言与框架**:
  * **Go**: GoZero（Gateway、Task、Processor 服务）
  * **Python**: gRPC + TensorFlow（Audio-Separator 服务）
```

**当前状态**: AIAdaptor 服务使用原生 gRPC 实现，违反架构设计要求。

### 1.2 技术栈对比

| 组件 | 当前实现 (原生 gRPC) | 目标实现 (go-zero) |
|------|---------------------|-------------------|
| **项目结构** | 自定义目录结构 | go-zero 标准结构 (goctl 生成) |
| **配置管理** | 环境变量 | YAML 配置文件 (etc/aiadaptor.yaml) |
| **依赖注入** | 手动管理 | ServiceContext 模式 |
| **日志系统** | 标准库 log | go-zero logx |
| **Redis 客户端** | go-redis v9 | go-zero redis.Redis |
| **gRPC 代码生成** | protoc | goctl rpc protoc |

### 1.3 迁移收益

1. **架构一致性**: 与 Task、Processor、Gateway 服务保持一致
2. **开发效率**: 利用 goctl 自动生成代码，减少样板代码
3. **可维护性**: 统一的项目结构和编码规范
4. **监控和日志**: go-zero 内置的监控和日志功能

---

## 2. 迁移范围和影响分析

### 2.1 需要迁移的模块清单

#### 2.1.1 核心模块

| 模块路径 | 文件数 | 代码行数 | 迁移复杂度 | 说明 |
|---------|-------|---------|-----------|------|
| `internal/logic/` | 5 | ~800 | 中 | 业务逻辑层，需适配 ServiceContext |
| `internal/adapters/` | 11 | ~2000 | 低 | 适配器层，逻辑不变，仅调整日志 |
| `internal/voice_cache/` | 1 | ~300 | 中 | 音色缓存管理，需适配 go-zero redis |
| `internal/config/` | 2 | ~200 | 高 | 配置管理，需完全重写 |
| `internal/utils/` | 若干 | ~100 | 低 | 工具函数，基本不变 |
| `main.go` | 1 | ~100 | 高 | 服务入口，需完全重写 |

#### 2.1.2 已实现的适配器列表

**ASR 适配器** (3个):
- `internal/adapters/asr/aliyun.go` - 阿里云 ASR
- `internal/adapters/asr/azure.go` - Azure ASR
- `internal/adapters/asr/google.go` - Google ASR

**翻译适配器** (1个):
- `internal/adapters/translation/google.go` - Google 翻译

**LLM 适配器** (2个):
- `internal/adapters/llm/openai.go` - OpenAI GPT
- `internal/adapters/llm/gemini.go` - Google Gemini

**声音克隆适配器** (1个):
- `internal/adapters/voice_cloning/aliyun_cosyvoice.go` - 阿里云 CosyVoice

**适配器基础设施** (2个):
- `internal/adapters/interface.go` - 适配器接口定义
- `internal/adapters/registry.go` - 适配器注册表

### 2.2 go.mod 依赖变更清单

#### 2.2.1 新增依赖

```go
require (
    github.com/zeromicro/go-zero v1.9.2  // go-zero 框架
    github.com/google/uuid v1.6.0        // UUID 生成（如需要）
)
```

#### 2.2.2 依赖变更

| 原依赖 | 新依赖 | 变更原因 |
|-------|-------|---------|
| `github.com/redis/go-redis/v9` | `github.com/zeromicro/go-zero/core/stores/redis` | 使用 go-zero 内置 Redis 客户端 |
| 标准库 `log` | `github.com/zeromicro/go-zero/core/logx` | 使用 go-zero 日志系统 |

#### 2.2.3 保留依赖

```go
require (
    google.golang.org/grpc v1.70.0
    google.golang.org/protobuf v1.36.0
    github.com/aliyun/aliyun-oss-go-sdk v3.0.2+incompatible
)
```

### 2.3 不可修改的 goctl 生成文件清单

**重要**: 以下文件由 goctl 生成，带有 "DO NOT EDIT" 标记，**禁止手动修改**：

1. `internal/server/aiadaptorServer.go` - gRPC 服务器实现
2. `aiadaptorservice/aiadaptor.go` - 服务接口定义
3. `proto/aiadaptor.pb.go` - Protocol Buffers 消息类
4. `proto/aiadaptor_grpc.pb.go` - gRPC 服务类

**修改方式**: 如需调整，修改 `proto/aiadaptor.proto` 文件，然后重新运行 `goctl rpc protoc`。

---

## 3. 分阶段迁移计划

### Phase 1: 基础设施搭建

**目标**: 使用 goctl 生成 go-zero 项目骨架

**任务清单**:
- [ ] 安装 goctl 工具 (v1.9.2)
- [ ] 备份现有代码到 `server/mcp/ai_adaptor-backup/`
- [ ] 创建新目录 `server/mcp/ai_adaptor-gozero/`
- [ ] 使用 goctl 生成项目结构
- [ ] 修复 proto 文件 go_package 选项
- [ ] 验证项目编译通过

**预计工作量**: 2 小时

**验收标准**:
- `go mod tidy` 通过
- `go build` 通过
- 目录结构符合 go-zero 规范

**详细步骤**:

```bash
# 1. 安装 goctl
go install github.com/zeromicro/go-zero/tools/goctl@v1.9.2

# 2. 备份现有代码
cd server/mcp
cp -r ai_adaptor ai_adaptor-backup

# 3. 创建新目录
mkdir ai_adaptor-gozero
cd ai_adaptor-gozero

# 4. 复制 proto 文件
mkdir proto
cp ../ai_adaptor/proto/aiadaptor.proto proto/

# 5. 修改 proto 文件的 go_package 选项
# 将 option go_package = "video-in-chinese/server/mcp/ai_adaptor/proto";
# 改为 option go_package = "./proto";

# 6. 使用 goctl 生成项目
goctl rpc protoc proto/aiadaptor.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style=goZero

# 7. 初始化 go.mod
go mod init video-in-chinese/server/mcp/ai_adaptor
go mod tidy

# 8. 验证编译
go build -o aiadaptor.exe .
```

**生成的项目结构**:

```
server/mcp/ai_adaptor-gozero/
├── etc/
│   └── aiadaptor.yaml              # go-zero 配置文件
├── internal/
│   ├── config/
│   │   └── config.go               # 配置结构体定义
│   ├── logic/
│   │   ├── asrLogic.go             # ASR 逻辑（待实现）
│   │   ├── polishLogic.go          # 文本润色逻辑（待实现）
│   │   ├── translateLogic.go       # 翻译逻辑（待实现）
│   │   ├── optimizeLogic.go        # 译文优化逻辑（待实现）
│   │   └── cloneVoiceLogic.go      # 声音克隆逻辑（待实现）
│   ├── server/
│   │   └── aiadaptorServer.go      # goctl 生成，不可修改
│   └── svc/
│       └── serviceContext.go       # 服务上下文（依赖注入）
├── proto/
│   ├── aiadaptor.pb.go
│   └── aiadaptor_grpc.pb.go
├── aiadaptorservice/
│   └── aiadaptor.go                # goctl 生成，不可修改
├── aiadaptor.go                    # 主程序入口
├── aiadaptor.proto
├── go.mod
└── go.sum
```

### Phase 2: 配置管理迁移

**目标**: 从环境变量迁移到 YAML 配置文件

**任务清单**:
- [ ] 创建 `etc/aiadaptor.yaml` 配置文件
- [ ] 更新 `internal/config/config.go` 添加自定义字段
- [ ] 迁移 Redis 配置
- [ ] 迁移加密密钥配置
- [ ] 验证配置加载

**预计工作量**: 2 小时

**验收标准**:
- 配置文件格式正确
- 所有环境变量已迁移到 YAML
- 配置加载测试通过

**配置文件示例** (`etc/aiadaptor.yaml`):

```yaml
Name: aiadaptor.rpc
ListenOn: 0.0.0.0:50051

# Redis 配置
Redis:
  Host: localhost:6379
  Type: node
  Pass: ""

# API 密钥加密密钥
ApiKeyEncryptionSecret: "your-32-byte-secret-key-here"

# OSS 配置（用于声音克隆）
OSS:
  Endpoint: "oss-cn-hangzhou.aliyuncs.com"
  AccessKeyId: ""
  AccessKeySecret: ""
  BucketName: "your-bucket-name"
```

**Config 结构体示例** (`internal/config/config.go`):

```go
package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	
	Redis struct {
		Host string
		Type string
		Pass string
	}
	
	ApiKeyEncryptionSecret string
	
	OSS struct {
		Endpoint        string
		AccessKeyId     string
		AccessKeySecret string
		BucketName      string
	}
}
```

### Phase 3: 存储层迁移

**目标**: 迁移 Redis 客户端和加密管理

**任务清单**:
- [ ] 创建 `internal/storage/redis.go` (使用 go-zero redis.Redis)
- [ ] 创建 `internal/storage/crypto.go` (加密解密逻辑)
- [ ] 迁移音色缓存管理器 `internal/voice_cache/manager.go`
- [ ] 更新 ServiceContext 集成存储层
- [ ] 验证 Redis 操作

**预计工作量**: 3 小时

**验收标准**:
- Redis 客户端初始化成功
- 加密解密功能正常
- 音色缓存读写正常

**go-redis v9 到 go-zero redis.Redis API 映射**:

| go-redis v9 | go-zero redis.Redis | 说明 |
|------------|---------------------|------|
| `client.Set(ctx, key, value, ttl)` | `redis.Setex(key, value, int(ttl.Seconds()))` | 设置带过期时间的键值 |
| `client.Get(ctx, key)` | `redis.Get(key)` | 获取键值 |
| `client.HSet(ctx, key, field, value)` | `redis.Hset(key, field, value)` | 设置 Hash 字段 |
| `client.HGetAll(ctx, key)` | `redis.Hgetall(key)` | 获取所有 Hash 字段 |
| `client.Del(ctx, key)` | `redis.Del(key)` | 删除键 |
| `client.LPush(ctx, key, value)` | `redis.Lpush(key, value)` | 左推入列表 |
| `client.RPop(ctx, key)` | `redis.Rpop(key)` | 右弹出列表 |

**注意**: go-zero redis.Redis 的方法**不需要** `context.Context` 参数。

**ServiceContext 示例** (`internal/svc/serviceContext.go`):

```go
package svc

import (
	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
	"video-in-chinese/server/mcp/ai_adaptor/internal/storage"
	"video-in-chinese/server/mcp/ai_adaptor/internal/voice_cache"
	
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config       config.Config
	RedisClient  *redis.Redis
	CryptoManager *storage.CryptoManager
	VoiceCacheManager *voice_cache.Manager
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 Redis 客户端
	rdb := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})
	
	// 初始化加密管理器
	cryptoManager := storage.NewCryptoManager(c.ApiKeyEncryptionSecret)
	
	// 初始化音色缓存管理器
	voiceCacheManager := voice_cache.NewManager(rdb)
	
	return &ServiceContext{
		Config:            c,
		RedisClient:       rdb,
		CryptoManager:     cryptoManager,
		VoiceCacheManager: voiceCacheManager,
	}
}
```

### Phase 4: 适配器层迁移

**目标**: 迁移 7 个已完成的适配器

**任务清单**:
- [ ] 迁移适配器接口定义 `internal/adapters/interface.go`
- [ ] 迁移适配器注册表 `internal/adapters/registry.go`
- [ ] 迁移 ASR 适配器 (3个: aliyun, azure, google)
- [ ] 迁移翻译适配器 (1个: google)
- [ ] 迁移 LLM 适配器 (2个: openai, gemini)
- [ ] 迁移声音克隆适配器 (1个: aliyun_cosyvoice)
- [ ] 更新日志调用为 logx

**预计工作量**: 4 小时

**验收标准**:
- 所有适配器编译通过
- 日志系统已切换到 logx
- 适配器注册表功能正常

**日志系统迁移指南**:

| 原代码 (标准库 log) | 新代码 (go-zero logx) |
|-------------------|---------------------|
| `log.Printf("info: %s", msg)` | `logx.Infof("info: %s", msg)` |
| `log.Printf("error: %v", err)` | `logx.Errorf("error: %v", err)` |
| `log.Println("debug")` | `logx.Info("debug")` |

**适配器迁移示例** (以 `aliyun.go` 为例):

```go
// 原代码
import "log"

func (a *AliyunASRAdapter) ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error) {
	log.Printf("[AliyunASR] Starting ASR: %s", audioPath)
	// ...
}

// 新代码
import "github.com/zeromicro/go-zero/core/logx"

func (a *AliyunASRAdapter) ASR(audioPath, apiKey, endpoint string) ([]*pb.Speaker, error) {
	logx.Infof("[AliyunASR] Starting ASR: %s", audioPath)
	// ...
}
```

### Phase 5: 业务逻辑层迁移

**目标**: 迁移 5 个 logic 模块

**任务清单**:
- [ ] 迁移 `asr_logic.go` (ASR 服务逻辑)
- [ ] 迁移 `polish_logic.go` (文本润色服务逻辑)
- [ ] 迁移 `translate_logic.go` (翻译服务逻辑)
- [ ] 迁移 `optimize_logic.go` (译文优化服务逻辑)
- [ ] 迁移 `clone_voice_logic.go` (声音克隆服务逻辑)
- [ ] 更新 godoc 注释为 go-zero 风格
- [ ] 验证业务逻辑

**预计工作量**: 4 小时

**验收标准**:
- 所有 logic 文件编译通过
- 使用 logx 日志系统
- 通过 ServiceContext 访问依赖
- 代码注释完整，符合 GoDoc 规范

**Logic 层迁移要点**:

1. **结构体定义**: 嵌入 `logx.Logger`

```go
type AsrLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}
```

2. **构造函数**: 使用 `logx.WithContext(ctx)`

```go
func NewAsrLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AsrLogic {
	return &AsrLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}
```

3. **依赖访问**: 通过 `l.svcCtx` 访问

```go
// 访问 Redis
voiceId, err := l.svcCtx.VoiceCacheManager.GetVoiceId(ctx, speakerId)

// 访问加密管理器
decryptedKey, err := l.svcCtx.CryptoManager.Decrypt(encryptedKey)
```

4. **日志记录**: 使用 `l.Infof()` / `l.Errorf()`

```go
l.Infof("[ASR] Processing audio: %s", in.AudioPath)
l.Errorf("[ASR] Failed to process: %v", err)
```

### Phase 6: 测试迁移

**目标**: 迁移 30 个测试用例到 go-zero 环境

**任务清单**:
- [ ] 迁移 Phase 1 基础设施测试 (10个)
- [ ] 迁移 Phase 2 配置管理测试 (5个)
- [ ] 迁移 Phase 6 单元测试 (18个)
- [ ] 迁移 Phase 6 集成测试 (6个)
- [ ] 迁移 Phase 6 Mock 测试 (6个)
- [ ] 生成覆盖率报告
- [ ] 更新测试文档

**预计工作量**: 5 小时

**验收标准**:
- 所有测试用例通过
- 业务逻辑覆盖率 > 80%
- 测试文档更新完成

**测试迁移要点**:

1. **Redis 测试**: 使用 go-zero redis.Redis

```go
// 原代码
rdb := redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

// 新代码
rdb := redis.MustNewRedis(redis.RedisConf{
	Host: "localhost:6379",
	Type: "node",
})
```

2. **ServiceContext 测试**: 创建测试用的 ServiceContext

```go
func newTestServiceContext() *svc.ServiceContext {
	c := config.Config{
		Redis: struct {
			Host string
			Type string
			Pass string
		}{
			Host: "localhost:6379",
			Type: "node",
			Pass: "",
		},
		ApiKeyEncryptionSecret: "test-secret-key-32-bytes-long",
	}
	return svc.NewServiceContext(c)
}
```

---

## 4. 技术适配指南

### 4.1 go-zero 框架核心概念

#### 4.1.1 ServiceContext (服务上下文)

**作用**: 依赖注入容器，管理所有服务依赖（Redis、数据库、外部客户端等）

**生命周期**: 服务启动时创建一次，所有请求共享

**使用场景**:
- 初始化 Redis 客户端
- 初始化加密管理器
- 初始化适配器注册表
- 初始化音色缓存管理器

#### 4.1.2 logx (日志系统)

**特性**:
- 结构化日志
- 支持日志级别 (Info, Error, Slow)
- 自动记录请求上下文 (trace_id, span_id)
- 支持日志轮转

**使用方式**:
```go
// 在 Logic 层
l.Infof("message: %s", msg)
l.Errorf("error: %v", err)

// 在其他地方
logx.Infof("message: %s", msg)
logx.Errorf("error: %v", err)
```

#### 4.1.3 配置管理

**配置文件**: `etc/aiadaptor.yaml`

**加载方式**:
```go
var c config.Config
conf.MustLoad(*configFile, &c)
```

**配置结构体**: 继承 `zrpc.RpcServerConf`

```go
type Config struct {
	zrpc.RpcServerConf
	// 自定义字段
	Redis struct {
		Host string
		Type string
		Pass string
	}
}
```

### 4.2 代码注释规范

**GoDoc 规范**:

1. **包注释**: 在 package 语句前添加

```go
// Package logic implements business logic for the AIAdaptor service.
//
// This package contains the core business logic for AI service orchestration.
package logic
```

2. **函数注释**: 说明功能、参数、返回值

```go
// ASR implements the speech recognition workflow.
//
// Workflow:
//  1. Read user configuration from Redis
//  2. Decrypt API key
//  3. Select ASR adapter based on provider
//  4. Call adapter to perform ASR
//  5. Return speaker-separated results
//
// Parameters:
//   - in: ASRRequest containing audio_path
//
// Returns:
//   - ASRResponse containing speakers and sentences
//   - error if any step fails
func (l *AsrLogic) ASR(in *proto.ASRRequest) (*proto.ASRResponse, error) {
	// ...
}
```

3. **结构体注释**: 说明用途和设计决策

```go
// AsrLogic encapsulates the business logic for ASR service.
//
// This struct holds the context and service dependencies needed to execute
// the ASR workflow. It is created per-request and is not reused.
type AsrLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}
```

---

## 5. 质量保证要求

### 5.1 每个 Phase 完成后的验证步骤

**Phase 1-5 验证**:

```bash
# 1. 依赖整理
go mod tidy

# 2. 代码格式化
gofmt -s -w .

# 3. 静态检查
go vet ./...

# 4. 编译验证
go build -o aiadaptor.exe .

# 5. 运行测试（如有）
go test -v ./...
```

**Phase 6 验证**:

```bash
# 1. 运行所有测试
go test -v ./...

# 2. 生成覆盖率报告
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 3. 检查覆盖率
go tool cover -func=coverage.out | grep total
```

### 5.2 代码审查检查清单

- [ ] 所有 goctl 生成的文件未被修改
- [ ] 所有日志调用使用 logx
- [ ] 所有 Redis 操作使用 go-zero redis.Redis
- [ ] 所有依赖通过 ServiceContext 访问
- [ ] 所有函数有完整的 GoDoc 注释
- [ ] 所有错误有适当的日志记录
- [ ] 配置文件格式正确
- [ ] go.mod 依赖版本正确

### 5.3 测试覆盖率要求

| 模块 | 覆盖率目标 |
|------|-----------|
| **业务逻辑层** (internal/logic/) | > 80% |
| **适配器层** (internal/adapters/) | > 70% |
| **存储层** (internal/storage/) | > 80% |
| **音色缓存** (internal/voice_cache/) | > 80% |

---

## 6. 风险和注意事项

### 6.1 goctl 生成文件的 "DO NOT EDIT" 约束

**风险**: 手动修改 goctl 生成的文件会导致代码被覆盖

**解决方案**:
- 修改 proto 文件，然后重新运行 `goctl rpc protoc`
- 业务逻辑放在 Logic 层，不要放在 Server 层

### 6.2 proto 文件 go_package 路径问题

**问题**: goctl 根据 `go_package` 生成嵌套目录，导致导入路径错误

**原配置**:
```proto
option go_package = "video-in-chinese/server/mcp/ai_adaptor/proto";
```

**修改后**:
```proto
option go_package = "./proto";
```

**结果**: 生成的导入路径正确 (`video-in-chinese/server/mcp/ai_adaptor/proto`)

### 6.3 并发安全和错误处理的保持

**要求**: 迁移过程中必须保持原有的并发安全和错误处理逻辑

**检查点**:
- 音色缓存的并发访问是否安全
- 适配器注册表的并发访问是否安全
- 所有错误是否有适当的日志记录
- 所有外部 API 调用是否有超时控制

### 6.4 Redis 客户端 API 差异

**注意**: go-zero redis.Redis 的方法**不需要** `context.Context` 参数

**错误示例**:
```go
// 错误：go-zero redis.Redis 不需要 ctx 参数
redis.Set(ctx, key, value)
```

**正确示例**:
```go
// 正确：直接调用，不传 ctx
redis.Setex(key, value, ttl)
```

---

## 7. 参考资料

### 7.1 官方文档

- **go-zero 官方文档**: https://go-zero.dev/
- **go-zero GitHub**: https://github.com/zeromicro/go-zero
- **goctl 工具文档**: https://go-zero.dev/docs/tutorials/cli/overview

### 7.2 项目文档

- **Base-Design.md v2.2**: `notes/server/1st/Base-Design.md`
- **AIAdaptor-design-detail.md**: `notes/server/3rd/AIAdaptor-design-detail.md`
- **Task 服务 GoZero 重构日志**: `server/mcp/task-gozero/GOZERO_REFACTORING_LOG.md`

### 7.3 示例代码

- **Task 服务 GoZero 实现**: `server/mcp/task-gozero/`
- **原 AIAdaptor 实现**: `server/mcp/ai_adaptor-backup/`

---

**文档维护者**: 开发团队  
**最后更新**: 2025-11-05  
**反馈渠道**: 请在项目 Issue 中提交文档问题或改进建议

