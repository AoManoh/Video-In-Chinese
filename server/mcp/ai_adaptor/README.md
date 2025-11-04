# AIAdaptor 服务

AIAdaptor 是视频翻译系统的 AI 服务适配器层，负责封装所有外部 AI API 调用，通过适配器模式实现业务逻辑与厂商解耦。

## 服务定位

- **统一接口**: 为 Processor 提供统一的 AI 服务 gRPC 接口（ASR、翻译、LLM、声音克隆）
- **适配器管理**: 封装厂商特定逻辑，支持多厂商切换
- **音色管理**: 管理声音克隆的音色注册、缓存、轮询（针对阿里云 CosyVoice）
- **配置读取**: 从 Redis 读取用户配置的 API 密钥和厂商选择
- **错误处理**: 统一错误处理和降级策略

## 技术栈

- **Go**: 1.21+
- **gRPC**: 1.60+
- **Redis**: go-redis/v9
- **加密**: AES-256-GCM

## 项目结构

```
server/mcp/ai_adaptor/
├── main.go                          # gRPC 服务入口
├── internal/
│   ├── logic/                       # 业务逻辑（Phase 5）
│   ├── adapters/                    # 适配器实现
│   │   ├── interface.go             # 适配器接口定义
│   │   ├── registry.go              # 适配器注册表
│   │   ├── asr/                     # ASR 适配器（Phase 4）
│   │   ├── translation/             # 翻译适配器（Phase 4）
│   │   ├── llm/                     # LLM 适配器（Phase 4）
│   │   └── voice_cloning/           # 声音克隆适配器（Phase 4）
│   ├── voice_cache/                 # 音色缓存管理器（Phase 3）
│   └── config/                      # 配置管理
│       ├── redis.go                 # Redis 配置读取
│       └── crypto.go                # API 密钥加密解密
├── proto/                           # gRPC 接口定义
│   ├── aiadaptor.proto              # Proto 文件
│   ├── aiadaptor.pb.go              # 生成的 Go 代码
│   └── aiadaptor_grpc.pb.go         # 生成的 gRPC 代码
├── go.mod                           # Go 模块定义
├── .env.example                     # 环境变量示例
└── README.md                        # 本文档
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- Redis 6.0+
- protoc (Protocol Buffers 编译器)

### 2. 安装依赖

```bash
cd server/mcp/ai_adaptor
go mod download
```

### 3. 生成 gRPC 代码

```bash
# 安装 protoc 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 生成 gRPC 代码
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/aiadaptor.proto
```

或使用脚本（Linux/Mac）:

```bash
chmod +x generate_grpc.sh
./generate_grpc.sh
```

### 4. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填写实际配置
```

**重要**: 必须设置 `API_KEY_ENCRYPTION_SECRET` 环境变量（32 字节十六进制字符串）

生成加密密钥:

```bash
openssl rand -hex 32
```

### 5. 启动服务

```bash
go run main.go
```

服务将监听端口 50053（可通过 `AI_ADAPTOR_GRPC_PORT` 环境变量配置）

## 开发进度

### Phase 1: 基础设施搭建 ✅ 已完成

- [x] 创建项目目录结构
- [x] 创建 proto 文件和生成 gRPC 代码
- [x] 实现适配器接口定义
- [x] 实现适配器注册表
- [x] 实现 Redis 配置管理
- [x] 实现 API 密钥加密解密
- [x] 实现 gRPC 服务入口
- [x] 创建配置文件和文档

### Phase 2: 配置管理（待实现）

- [ ] 实现配置缓存策略
- [ ] 实现配置热更新

### Phase 3: 音色缓存管理器（待实现）

- [ ] 实现音色注册逻辑
- [ ] 实现音色轮询逻辑
- [ ] 实现音色缓存失效处理

### Phase 4: 适配器实现（待实现）

- [ ] 实现 ASR 适配器（阿里云、Azure、Google）
- [ ] 实现翻译适配器（DeepL、Google、Azure）
- [ ] 实现 LLM 适配器（OpenAI、Claude、Gemini）
- [ ] 实现声音克隆适配器（阿里云 CosyVoice）

### Phase 5: 服务逻辑实现（待实现）

- [ ] 实现 ASR 服务逻辑
- [ ] 实现文本润色服务逻辑
- [ ] 实现翻译服务逻辑
- [ ] 实现译文优化服务逻辑
- [ ] 实现声音克隆服务逻辑

### Phase 6: 测试实现（待实现）

- [ ] 单元测试
- [ ] 集成测试
- [ ] 端到端测试

## gRPC 接口

### ASR (语音识别)

```protobuf
rpc ASR(ASRRequest) returns (ASRResponse);
```

### Polish (文本润色)

```protobuf
rpc Polish(PolishRequest) returns (PolishResponse);
```

### Translate (翻译)

```protobuf
rpc Translate(TranslateRequest) returns (TranslateResponse);
```

### Optimize (译文优化)

```protobuf
rpc Optimize(OptimizeRequest) returns (OptimizeResponse);
```

### CloneVoice (声音克隆)

```protobuf
rpc CloneVoice(CloneVoiceRequest) returns (CloneVoiceResponse);
```

详细接口定义请参阅 `proto/aiadaptor.proto`

## 配置说明

所有配置通过环境变量传入，详见 `.env.example`

### 必填配置

- `API_KEY_ENCRYPTION_SECRET`: API 密钥加密密钥（32 字节十六进制字符串）

### 可选配置

- `AI_ADAPTOR_GRPC_PORT`: gRPC 服务端口（默认 50053）
- `REDIS_HOST`: Redis 主机地址（默认 redis）
- `REDIS_PORT`: Redis 端口（默认 6379）
- `VOICE_CACHE_TTL`: 音色缓存过期时间（默认 0，不过期）
- `LOG_LEVEL`: 日志级别（默认 info）

## 参考文档

- 第二层设计文档: `notes/server/2nd/AIAdaptor-design.md`
- 第三层实现文档: `notes/server/3rd/AIAdaptor-design-detail.md`
- 开发任务清单: `notes/server/process/development-todo.md`

## 许可证

MIT

