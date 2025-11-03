# 视频翻译服务开发任务清单

**文档版本**: 1.0  
**创建日期**: 2025-11-03  
**最后更新**: 2025-11-03  
**当前开发阶段**: 阶段一-A (AudioSeparator 服务)  
**参考文档**: `plan.md` v1.4

---

## 📌 任务状态标记规则

- **未开始**: `- [ ]`
- **进行中**: `- [/]`
- **已完成**: `- [x]`
- **模块开发完成**: 在任务项后标记 `✅ 开发完成`
- **模块测试通过**: 在模块标题后标记 `✅ 测试通过，模块完成`

**重要提示**: 
- 每完成一个任务点必须立即更新状态
- 每个模块开发完成后必须先通过测试才能标记为完成
- 本文档是开发进度的唯一真实来源（Single Source of Truth）

---

## 🎯 开发顺序概览

1. **阶段一 (并行)**: AudioSeparator + AIAdaptor
2. **阶段二**: Task
3. **阶段三**: Processor
4. **阶段四**: Gateway
5. **阶段五**: 系统集成测试

---

## 📋 阶段一-A: AudioSeparator 服务 (Python)

**服务端口**: 50052  
**技术栈**: Python 3.9+, gRPC, Spleeter, TensorFlow  
**依赖**: ffmpeg, libsndfile1  
**参考文档**: `AudioSeparator-design-detail.md` v2.0

### Phase 1: 基础设施搭建

- [ ] 创建项目目录结构 `server/mcp/audio_separator/`
- [ ] 创建 `main.py` (gRPC 服务入口)
- [ ] 创建 `separator_service.py` (gRPC 服务实现)
- [ ] 创建 `spleeter_wrapper.py` (Spleeter 模型封装)
- [ ] 创建 `config.py` (配置管理)
- [ ] 创建 `proto/audioseparator.proto` (gRPC 接口定义)
- [ ] 配置 gRPC 服务器监听端口 50052
- [ ] 实现从环境变量读取配置
- [ ] 验证 Python 版本要求 (3.9+)
- [ ] 验证系统依赖 (ffmpeg, libsndfile1)

### Phase 2: Spleeter 模型封装

- [ ] 实现懒加载逻辑 (首次调用时加载模型)
- [ ] 实现模型缓存 (字典存储: {stems: model})
- [ ] 实现模型加载错误处理
- [ ] 实现内存不足错误处理
- [ ] 实现日志记录 (INFO/WARN/ERROR)

### Phase 3: 音频分离逻辑 (9步)

- [ ] 步骤1: 参数验证 (audio_path, output_dir, stems)
- [ ] 步骤2: 输出目录创建 (如果不存在)
- [ ] 步骤3: 处理上下文初始化 (开始时间记录)
- [ ] 步骤4: 模型加载 (调用 SpleeterWrapper.get_model)
- [ ] 步骤5: 音频分离 (调用 model.separate)
- [ ] 步骤6: 输出路径构建 (vocals.wav, accompaniment.wav)
- [ ] 步骤7: 输出文件验证 (检查文件是否存在)
- [ ] 步骤8: 处理耗时计算 (结束时间 - 开始时间)
- [ ] 步骤9: 成功响应返回 (vocals_path, accompaniment_path, processing_time)

### Phase 4: 并发控制

- [ ] 实现最大并发数控制 (AUDIO_SEPARATOR_MAX_WORKERS=1)
- [ ] 实现超时处理 (10分钟超时)
- [ ] 实现资源清理 (处理失败时清理临时文件)

### Phase 5: 测试实现

#### 单元测试
- [ ] 创建 `tests/test_spleeter_wrapper.py`
- [ ] 测试模型懒加载
- [ ] 测试模型缓存
- [ ] 测试错误处理 (模型加载失败)

#### 集成测试
- [ ] 创建 `tests/test_separator_service.py`
- [ ] 测试完整的音频分离流程 (10秒音频)
- [ ] 测试并发控制 (多个请求)
- [ ] 测试超时处理 (超大音频文件)

#### 性能测试
- [ ] 验证10分钟音频处理时间 <15分钟 (CPU模式)
- [ ] 验证内存占用 <2GB
- [ ] 验证模型加载时间 <60秒

### Phase 6: 文档和代码审查

- [ ] 编写代码注释 (解释关键决策)
- [ ] 编写 API 文档 (gRPC 接口说明)
- [ ] 编写测试报告 (测试覆盖率、性能指标)
- [ ] Code Review (Python 工程师互审)

### 验收标准

- [ ] 可分离10分钟音频，处理时间 <15分钟 (CPU模式)
- [ ] 模型懒加载正常工作 (首次调用加载，后续复用)
- [ ] 并发控制有效 (maxWorkers=1 时无并发冲突)
- [ ] 单元测试覆盖率 >80%
- [ ] 集成测试通过 (真实音频文件)
- [ ] 代码通过 Code Review

---

## 📋 阶段一-B: AIAdaptor 服务 (Go)

**服务端口**: 50053  
**技术栈**: Go 1.21+, gRPC, go-redis  
**依赖**: Redis  
**参考文档**: `AIAdaptor-design-detail.md` v2.0

### Phase 1: 基础设施搭建

- [ ] 创建项目目录结构 `server/mcp/ai_adaptor/`
- [ ] 创建 `main.go` (gRPC 服务入口)
- [ ] 创建 `internal/logic/` (业务逻辑)
- [ ] 创建 `internal/adapters/` (适配器实现)
- [ ] 创建 `internal/voice_cache/` (音色缓存管理)
- [ ] 创建 `internal/config/` (配置管理)
- [ ] 实现 gRPC 服务入口 (监听端口 50053)
- [ ] 实现适配器注册表 (接口+注册表模式)
- [ ] 配置 Redis 连接 (读取 app:settings 和 voice_cache)
- [ ] 验证 Go 版本要求 (1.21+)

### Phase 2: 配置管理

- [ ] 实现 Redis 连接 (go-redis 客户端)
- [ ] 实现 API 密钥解密 (AES-256-GCM)
- [ ] 实现配置缓存策略 (10分钟过期，避免频繁访问 Redis)
- [ ] 实现配置热更新 (监听 Redis 配置变更)

### Phase 3: 音色缓存管理器

- [ ] 实现音色注册逻辑 (调用阿里云 CosyVoice API)
- [ ] 实现音色轮询逻辑 (指数退避，最多5次，每次间隔 1/2/4/8/16秒)
- [ ] 实现音色缓存 (Redis, Key: voice_cache:{speaker_id}, TTL: 24小时)
- [ ] 实现音色缓存失效处理 (404错误时自动重新注册)

### Phase 4: 适配器实现

#### 主工程师负责
- [ ] 实现阿里云 ASR 适配器 (internal/adapters/asr/aliyun.go)
- [ ] 实现 Azure ASR 适配器 (internal/adapters/asr/azure.go)
- [ ] 实现 Google ASR 适配器 (internal/adapters/asr/google.go)
- [ ] 实现阿里云 CosyVoice 适配器 (internal/adapters/voice_cloning/aliyun_cosyvoice.go)
  - [ ] 音色注册 (RegisterVoice)
  - [ ] 音色轮询 (PollVoiceStatus)
  - [ ] 音频合成 (SynthesizeAudio)
  - [ ] 音色缓存管理 (使用 voice_cache 管理器)

#### 副工程师负责
- [ ] 实现 DeepL 翻译适配器 (internal/adapters/translation/deepl.go)
- [ ] 实现 Google 翻译适配器 (internal/adapters/translation/google.go)
- [ ] 实现 Azure 翻译适配器 (internal/adapters/translation/azure.go)
- [ ] 实现 OpenAI LLM 适配器 (internal/adapters/llm/openai.go)
- [ ] 实现 Claude LLM 适配器 (internal/adapters/llm/claude.go)
- [ ] 实现 Gemini LLM 适配器 (internal/adapters/llm/gemini.go)

### Phase 5: 服务逻辑实现

- [ ] 实现 ASR 服务逻辑 (internal/logic/asr_logic.go)
- [ ] 实现翻译服务逻辑 (internal/logic/translate_logic.go)
- [ ] 实现文本润色服务逻辑 (internal/logic/polish_logic.go)
- [ ] 实现译文优化服务逻辑 (internal/logic/optimize_logic.go)
- [ ] 实现声音克隆服务逻辑 (internal/logic/clone_voice_logic.go)

### Phase 6: 测试实现

#### 单元测试
- [ ] 测试适配器注册表 (注册、选择、调用)
- [ ] 测试 ASR 适配器 (Mock API 响应)
- [ ] 测试翻译适配器 (Mock API 响应)
- [ ] 测试 LLM 适配器 (Mock API 响应)
- [ ] 测试声音克隆适配器 (Mock API 响应)
- [ ] 测试音色注册 (正常、失败、重试)
- [ ] 测试音色轮询 (成功、超时、失败)
- [ ] 测试音色缓存 (写入、读取、失效)

#### 集成测试
- [ ] 测试配置读取和解密
- [ ] 测试音色缓存写入和读取
- [ ] 测试阿里云 ASR (真实音频文件，可选)
- [ ] 测试 DeepL 翻译 (真实文本，可选)
- [ ] 测试 OpenAI LLM (真实 Prompt，可选)
- [ ] 测试阿里云 CosyVoice (真实音频合成，可选)

#### Mock 测试
- [ ] 测试根据配置动态选择适配器

### Phase 7: 文档和代码审查

- [ ] 编写代码注释 (解释适配器模式设计决策)
- [ ] 编写 API 文档 (gRPC 接口说明)
- [ ] 编写测试报告 (测试覆盖率、集成测试结果)
- [ ] Code Review (Go 工程师互审)

### 验收标准

- [ ] 可调用至少3个厂商的 ASR 服务 (阿里云、Azure、Google)
- [ ] 可调用至少2个厂商的翻译服务 (DeepL、Google)
- [ ] 可调用至少2个厂商的 LLM 服务 (OpenAI、Claude)
- [ ] 声音克隆功能可正常注册音色并合成音频
- [ ] 适配器选择逻辑正常工作 (根据 Redis 配置动态选择)
- [ ] 音色缓存正常工作 (首次注册后缓存，后续直接使用)
- [ ] 单元测试覆盖率 >80%
- [ ] 集成测试通过 (Redis 连接、配置读取)
- [ ] 代码通过 Code Review

---

## 📋 阶段二: Task 服务 (Go)

**服务端口**: 50050  
**技术栈**: Go 1.21+, gRPC, go-redis  
**依赖**: Redis  
**参考文档**: `Task-design-detail.md` v1.0

### Phase 1: 基础设施搭建

- [ ] 创建项目目录结构 `server/mcp/task/`
- [ ] 创建 `main.go` (gRPC 服务入口)
- [ ] 创建 `internal/logic/` (业务逻辑)
- [ ] 创建 `internal/storage/` (存储层)
- [ ] 创建 `internal/svc/` (服务上下文)
- [ ] 实现 gRPC 服务入口 (监听端口 50050)
- [ ] 配置 Redis 连接 (go-redis 客户端)
- [ ] 验证 Go 版本要求 (1.21+)

### Phase 2: 存储层实现

- [ ] 实现 Redis 队列操作 (internal/storage/redis.go)
  - [ ] LPUSH: 推入任务到队列 (Key: task:pending)
  - [ ] RPOP: 拉取任务从队列 (由 Processor 调用)
  - [ ] LLEN: 查询队列长度
- [ ] 实现 Redis Hash 操作 (任务状态存储)
  - [ ] HSET: 设置任务字段 (Key: task:{task_id})
  - [ ] HGETALL: 读取任务所有字段
  - [ ] 定义字段: task_id, status, original_file_path, result_file_path, error_message, created_at, updated_at
- [ ] 实现文件操作封装 (internal/storage/file.go)
  - [ ] 文件移动 (os.Rename)
  - [ ] 跨文件系统降级策略 (os.Rename 失败时使用 io.Copy + os.Remove)
  - [ ] 文件存在性检查

### Phase 3: 业务逻辑实现

- [ ] 实现 CreateTask 逻辑 (internal/logic/create_task_logic.go)
  - [ ] 步骤1: 生成任务 ID (UUID v4)
  - [ ] 步骤2: 构建正式文件路径 ({LOCAL_STORAGE_PATH}/videos/{task_id}/original.mp4)
  - [ ] 步骤3: 创建任务目录 (如果不存在)
  - [ ] 步骤4: 文件交接 (临时文件→正式文件，使用 os.Rename)
  - [ ] 步骤5: 创建任务记录 (Redis Hash，初始状态: PENDING)
  - [ ] 步骤6: 推入任务到队列 (Redis LPUSH, Key: task:pending)
  - [ ] 步骤7: 返回任务 ID
- [ ] 实现 GetTaskStatus 逻辑 (internal/logic/get_task_status_logic.go)
  - [ ] 步骤1: 从 Redis 读取任务状态 (HGETALL task:{task_id})
  - [ ] 步骤2: 检查任务是否存在 (如果不存在返回 NOT_FOUND 错误)
  - [ ] 步骤3: 返回任务状态 (status, result_file_path, error_message)

### Phase 4: 测试实现

#### 单元测试
- [ ] 测试文件交接 (os.Rename 正常情况)
- [ ] 测试跨文件系统降级 (Mock 跨文件系统错误)
- [ ] 测试文件不存在错误处理
- [ ] 测试 LPUSH 和 LLEN
- [ ] 测试 HSET 和 HGETALL
- [ ] 使用 testcontainers-go 启动真实 Redis 容器

#### 集成测试
- [ ] 测试 CreateTask (临时文件→正式文件→队列推入)
- [ ] 测试 GetTaskStatus (读取任务状态)
- [ ] 验证 Redis 中的数据结构正确

#### Mock 测试
- [ ] Mock gRPC Server
- [ ] 测试 CreateTask 和 GetTaskStatus 接口

### Phase 5: 文档和代码审查

- [ ] 编写代码注释 (解释文件移动策略、队列设计)
- [ ] 编写 API 文档 (gRPC 接口说明)
- [ ] 编写测试报告 (测试覆盖率、集成测试结果)
- [ ] Code Review (Go 工程师审查)

### 验收标准

- [ ] CreateTask 可创建任务并推入 Redis 队列
- [ ] GetTaskStatus 可查询任务状态 (4种状态: PENDING/PROCESSING/COMPLETED/FAILED)
- [ ] 文件交接逻辑正常 (临时文件→正式文件)
- [ ] Redis 队列和 Hash 结构正确 (与 Processor 期望一致)
- [ ] 单元测试覆盖率 >80%
- [ ] 集成测试通过 (真实 Redis 容器)
- [ ] 代码通过 Code Review

---

## 📋 阶段三: Processor 服务 (Go)

**服务端口**: 无 (后台服务)
**技术栈**: Go 1.21+, gRPC Client, go-redis, ffmpeg
**依赖**: Redis, AIAdaptor, AudioSeparator, ffmpeg
**参考文档**: `Processor-design-detail.md` v2.0

### Phase 1: 基础设施搭建

- [ ] 创建项目目录结构 `server/mcp/processor/`
- [ ] 创建 `main.go` (后台服务入口)
- [ ] 创建 `internal/logic/` (业务逻辑)
- [ ] 创建 `internal/composer/` (音频合成)
- [ ] 创建 `internal/mediautil/` (媒体工具)
- [ ] 创建 `internal/storage/` (存储层)
- [ ] 配置 Redis 连接 (读取任务队列、任务状态、应用配置)
- [ ] 配置 gRPC 客户端 (AIAdaptor, AudioSeparator)
- [ ] 确认 Task 服务的 Redis 数据结构
  - [ ] 确认队列 Key 名称 (task:pending)
  - [ ] 确认任务状态 Hash Key 格式 (task:{task_id})
  - [ ] 确认任务状态字段
  - [ ] 确认状态枚举值 (PENDING, PROCESSING, COMPLETED, FAILED)
- [ ] 验证 Go 版本要求 (1.21+)
- [ ] 验证系统依赖 (ffmpeg >= 4.0)

### Phase 2: Composer 包实现

#### 主工程师负责
- [ ] 实现音频拼接 (internal/composer/concatenate.go)
  - [ ] 将所有克隆音频片段按时间顺序拼接
  - [ ] 使用 ffmpeg 拼接音频片段
  - [ ] 错误处理和日志记录
- [ ] 实现时长对齐 (internal/composer/align.go)
  - [ ] 策略1: 静音填充 (当翻译音频比原音频短)
  - [ ] 策略2: 语速加速 (当翻译音频比原音频长，加速比率 ≤1.3)
  - [ ] 策略3: LLM 重译 (当加速后仍超长)
  - [ ] 策略4: 截断降级 (最后的降级策略)
  - [ ] 实现策略选择决策树

#### 副工程师负责
- [ ] 实现音频合并 (internal/composer/merge.go)
  - [ ] 如果有背景音，将人声和背景音合并
  - [ ] 使用 ffmpeg 的 amix 滤镜
  - [ ] 如果没有背景音，直接返回人声

### Phase 3: Mediautil 包实现

- [ ] 实现音频提取 (internal/mediautil/extract.go)
  - [ ] 从视频中提取音频
  - [ ] 使用 ffmpeg 命令
- [ ] 实现音视频合并 (internal/mediautil/merge.go)
  - [ ] 合并视频 + 新音轨
  - [ ] 使用 ffmpeg 命令

### Phase 4: 主流程编排 (14步)

- [ ] 实现任务拉取循环 (internal/logic/task_pull_loop.go)
  - [ ] 定期轮询 Redis 队列 (每5秒检查一次)
  - [ ] 尝试获取 worker 槽位 (使用 Channel 信号量)
  - [ ] 如果达到并发上限，跳过本次拉取
  - [ ] 如果拉取到任务，启动新 Goroutine 处理
- [ ] 实现14步处理流程 (internal/logic/processor_logic.go)
  - [ ] 步骤1: 状态更新 (立即更新 Redis 任务状态为 PROCESSING)
  - [ ] 步骤2: 读取应用配置 (从 Redis 读取 app:settings 并解密 API 密钥)
  - [ ] 步骤3: 文件准备 (从本地存储读取原始视频)
  - [ ] 步骤4: 音频提取 (调用 mediautil.Extract)
  - [ ] 步骤5: 音频分离 (可选，调用 AudioSeparator 服务)
  - [ ] 步骤6: ASR (语音识别，调用 AIAdaptor.ASR)
  - [ ] 步骤6.5: 音频片段切分 (根据 ASR 返回的时间戳切分音频)
  - [ ] 步骤7: 文本润色 (可选，调用 AIAdaptor.Polish)
  - [ ] 步骤8: 翻译 (调用 AIAdaptor.Translate)
  - [ ] 步骤9: 译文优化 (可选，调用 AIAdaptor.Optimize)
  - [ ] 步骤10: 声音克隆 (调用 AIAdaptor.CloneVoice)
  - [ ] 步骤11: 音频合成 (调用 composer 包)
    - [ ] 子步骤1: 音频拼接 (composer.Concatenate)
    - [ ] 子步骤2: 时长对齐 (composer.Align)
    - [ ] 子步骤3: 音频合并 (composer.Merge)
  - [ ] 步骤12: 视频合成 (调用 mediautil.Merge)
  - [ ] 步骤13: 更新任务状态为 COMPLETED
  - [ ] 步骤14: 异常处理 (任何步骤失败则更新状态为 FAILED)

### Phase 5: 并发控制和错误处理

- [ ] 实现并发控制逻辑
  - [ ] 使用带缓冲的 Channel 作为信号量
  - [ ] 获取槽位: 尝试向 Channel 发送值
  - [ ] 释放槽位: 从 Channel 接收值 (使用 defer 确保释放)
- [ ] 实现错误处理逻辑
  - [ ] 快速失败: 任何步骤失败立即中止
  - [ ] 状态更新: 更新 Redis 任务状态为 FAILED
  - [ ] 错误记录: 记录详细错误信息到 Redis 的 error_message 字段
  - [ ] 文件清理: 删除中间文件
  - [ ] 资源释放: 释放 worker 槽位

### Phase 6: GC 定时任务

- [ ] 实现 GC 定时任务 (清理过期任务目录)
  - [ ] 每小时扫描一次 {LOCAL_STORAGE_PATH}/videos 目录
  - [ ] 删除创建时间超过3小时的任务目录
  - [ ] 记录删除日志 (INFO 级别)

### Phase 7: 测试实现

#### 单元测试
- [ ] 测试音频拼接 (使用真实音频文件)
- [ ] 测试时长对齐 (4个策略: 静音填充、语速加速、LLM 重译、截断)
- [ ] 测试音频合并 (人声+背景音)
- [ ] 测试音频提取 (使用真实视频文件)
- [ ] 测试音视频合并

#### 集成测试
- [ ] 测试任务拉取 (RPOP 队列)
- [ ] 测试任务状态更新 (HSET)
- [ ] 测试应用配置读取 (app:settings)
- [ ] Mock AIAdaptor 服务
- [ ] Mock AudioSeparator 服务
- [ ] 测试14步流程 (使用 Mock 服务)

#### 端到端测试
- [ ] 使用真实的 AIAdaptor 和 AudioSeparator 服务
- [ ] 测试完整的视频翻译流程 (10秒视频)
- [ ] 验证输出视频质量

### Phase 8: 性能优化和文档

- [ ] 并发性能测试 (测试不同并发数: 1、2、4)
- [ ] 内存优化 (使用 pprof 分析内存占用)
- [ ] 磁盘 I/O 优化
- [ ] 编写代码注释 (解释14步流程、并发控制策略、时长对齐决策树)
- [ ] 编写测试报告 (测试覆盖率、性能指标)
- [ ] Code Review (Go 工程师审查)

### 验收标准

- [ ] 可从 Redis 队列拉取任务
- [ ] 14步处理流程可完整执行 (使用 Mock AI 服务)
- [ ] 步骤2 (读取应用配置) 和步骤6.5 (音频片段切分) 正常工作
- [ ] 步骤11 (音频合成) 的3个子步骤正常工作
- [ ] 时长对齐的4个策略决策树正确实现
- [ ] 并发控制有效 (maxConcurrency=1 时无并发冲突)
- [ ] 错误处理正确 (任何步骤失败都能正确更新状态和清理资源)
- [ ] GC 定时任务正常工作 (清理过期任务目录)
- [ ] 单元测试覆盖率 >80%
- [ ] 集成测试通过 (Mock gRPC 服务)
- [ ] 端到端测试通过 (真实 AI 服务)
- [ ] 代码通过 Code Review

---

## 📋 阶段四: Gateway 服务 (Go)

**服务端口**: 8080
**技术栈**: Go 1.21+, RESTful API, gRPC Client, go-redis
**依赖**: Redis, Task 服务
**参考文档**: `Gateway-design-detail.md` v1.0

### Phase 1: 基础设施搭建

- [ ] 创建项目目录结构 `server/mcp/gateway/`
- [ ] 创建 `main.go` (RESTful API 服务入口)
- [ ] 创建 `internal/handler/` (HTTP 处理器)
- [ ] 创建 `internal/logic/` (业务逻辑)
- [ ] 创建 `internal/middleware/` (中间件)
- [ ] 创建 `internal/svc/` (服务上下文)
- [ ] 实现 RESTful API 服务入口 (监听端口 8080)
- [ ] 配置 gRPC 客户端 (Task 服务，地址: task:50050)
- [ ] 配置 Redis 连接 (配置管理，读取 app:settings)
- [ ] 验证 Go 版本要求 (1.21+)

### Phase 2: 核心工具函数实现

- [ ] 实现磁盘空间预检 (internal/logic/disk_check.go)
  - [ ] 使用 syscall.Statfs 获取磁盘信息
  - [ ] 公式: availableSpace >= fileSize * 3 + 500MB
  - [ ] 如果空间不足返回 507 Insufficient Storage
- [ ] 实现路径安全检查 (internal/logic/path_check.go)
  - [ ] 检查路径遍历攻击 (../)
  - [ ] 检查符号链接 (filepath.EvalSymlinks)
  - [ ] 如果检测到异常返回 400 Bad Request
- [ ] 实现 API Key 加密/解密 (internal/logic/crypto.go)
  - [ ] 使用 AES-256-GCM 加密
  - [ ] 密钥从环境变量 API_KEY_ENCRYPTION_SECRET 读取 (32字节)
  - [ ] 生成随机 nonce (12字节)
  - [ ] 返回格式: base64(nonce + ciphertext)
- [ ] 实现 MIME Type 检测 (internal/logic/mime_check.go)
  - [ ] 通过文件头检测 (而非扩展名)
  - [ ] 白名单: video/mp4, video/avi, video/mkv, video/mov
  - [ ] 如果不匹配返回 400 Bad Request

### Phase 3: 配置管理功能实现

- [ ] 实现 GetSettings 逻辑 (internal/logic/settings/get_settings_logic.go)
  - [ ] 步骤1: 从 Redis 读取配置 (Key: app:settings)
  - [ ] 步骤2: 如果不存在，返回默认配置
  - [ ] 步骤3: 解密 API 密钥
  - [ ] 步骤4: 脱敏 API 密钥 (格式: 前缀-***-后6位)
  - [ ] 步骤5: 返回配置 (包含 version 字段用于乐观锁)
- [ ] 实现 UpdateSettings 逻辑 (internal/logic/settings/update_settings_logic.go)
  - [ ] 步骤1: 参数验证
  - [ ] 步骤2: 使用 Lua 脚本原子性更新 Redis
  - [ ] 步骤3: 返回更新后的配置

### Phase 4: 文件上传功能实现

- [ ] 实现 UploadTask 逻辑 (internal/logic/task/upload_task_logic.go)
  - [ ] 步骤1: 解析 multipart/form-data
  - [ ] 步骤2: 检查文件大小 (MAX_UPLOAD_SIZE_MB=2048)
  - [ ] 步骤3: 检查磁盘空间
  - [ ] 步骤4: MIME Type 验证
  - [ ] 步骤5: 生成临时文件路径
  - [ ] 步骤6: 流式保存文件
  - [ ] 步骤7: 调用 Task 服务的 CreateTask 接口
  - [ ] 步骤8: 返回任务 ID
  - [ ] 步骤9: 异常处理 (defer 清理临时文件)
- [ ] 实现临时文件清理逻辑

### Phase 5: 任务状态查询功能实现

- [ ] 实现 GetTaskStatus 逻辑 (internal/logic/task/get_task_status_logic.go)
  - [ ] 步骤1: 调用 Task 服务的 GetTaskStatus 接口
  - [ ] 步骤2: 状态枚举映射
  - [ ] 步骤3: 如果任务完成，生成下载 URL
  - [ ] 步骤4: 返回任务状态

### Phase 6: 文件下载功能实现

- [ ] 实现 DownloadFile 逻辑 (internal/logic/task/download_file_logic.go)
  - [ ] 步骤1: 路径拼接
  - [ ] 步骤2: 路径安全检查
  - [ ] 步骤3: 文件存在性检查
  - [ ] 步骤4: MIME Type 检测
  - [ ] 步骤5: 设置响应头
  - [ ] 步骤6: 流式传输文件
  - [ ] 步骤7: Range 请求支持 (P1，断点续传)

### Phase 7: 中间件实现

- [ ] 实现 CORS 中间件 (internal/middleware/cors_middleware.go)
- [ ] 实现日志中间件 (internal/middleware/logging_middleware.go)
- [ ] 实现错误处理中间件 (internal/middleware/error_middleware.go)
- [ ] 实现限流中间件 (P2, internal/middleware/rate_limit_middleware.go)

### Phase 8: 测试和优化

#### 单元测试
- [ ] 测试磁盘空间预检 (Mock syscall.Statfs)
- [ ] 测试路径安全检查 (测试路径遍历、符号链接)
- [ ] 测试 API Key 加密/解密
- [ ] 测试 MIME Type 检测
- [ ] 测试 GetSettings (Redis 存在/不存在)
- [ ] 测试 UpdateSettings (正常更新、版本冲突、加密)
- [ ] 测试流式保存 (Mock 文件流)
- [ ] 测试文件大小检查
- [ ] 测试磁盘空间检查
- [ ] 测试 MIME Type 验证
- [ ] 测试临时文件清理
- [ ] 测试 GetTaskStatus (Mock Task 服务)
- [ ] 测试下载 URL 生成
- [ ] 测试流式传输 (Mock 文件系统)
- [ ] 测试路径安全检查
- [ ] 测试 Range 请求支持

#### 集成测试
- [ ] 测试文件上传→任务创建→状态查询→文件下载
- [ ] 使用真实的 Task 服务 (或 Mock gRPC Server)

#### 端到端测试
- [ ] 测试用户上传视频→查询状态→下载结果
- [ ] 验证流式处理 (内存占用 <100MB)

#### 性能测试
- [ ] 测试10个并发上传 (文件大小 100MB)
- [ ] 验证内存占用 <500MB
- [ ] 验证吞吐量 >10MB/s

#### 代码审查与优化
- [ ] Code Review (Go 工程师审查)
- [ ] 性能优化 (pprof 分析)
- [ ] 代码质量优化 (golangci-lint)

### Phase 9: 文档编写

- [ ] 编写代码注释 (解释流式处理策略、安全检查逻辑)
- [ ] 编写 API 文档 (RESTful 接口说明，包含请求/响应示例)
- [ ] 编写测试报告 (测试覆盖率、性能指标)

### 验收标准

- [ ] 可接收文件上传请求并调用 Task 服务 (连接地址: task:50050)
- [ ] 可查询任务状态并返回下载 URL
- [ ] 可提供文件下载 (流式传输)
- [ ] 配置管理功能正常 (加密存储+乐观锁)
- [ ] 磁盘空间预检正常 (避免磁盘耗尽)
- [ ] 路径安全检查正常 (防止路径遍历)
- [ ] MIME Type 验证正常 (防止恶意文件)
- [ ] 文件大小限制正确 (MAX_UPLOAD_SIZE_MB=2048)
- [ ] 单元测试覆盖率 >80%
- [ ] 集成测试通过 (真实 Task 服务或 Mock)
- [ ] 端到端测试通过 (完整 HTTP API 流程)
- [ ] 性能测试达标 (并发10个请求，内存 <500MB)
- [ ] 代码通过 Code Review

---

## 📋 阶段五: 系统集成测试

**负责人**: 全体工程师
**参考文档**: `plan.md` v1.4 第812-887行

### Phase 1: 集成测试准备

- [ ] 部署所有5个服务 (AudioSeparator, AIAdaptor, Task, Processor, Gateway)
- [ ] 配置 Docker Compose (定义服务依赖、网络、卷)
- [ ] 配置环境变量 (Redis 连接、API 密钥、存储路径)
- [ ] 启动 Redis 容器 (启用 AOF 持久化)
- [ ] 验证服务间通信 (gRPC 健康检查)
- [ ] 验证端口配置 (Task 服务监听 50050，Gateway 连接 task:50050)

### Phase 2: 端到端测试

- [ ] 测试完整的视频翻译流程
  - [ ] 步骤1: 用户通过 Web 上传视频 (10秒测试视频)
  - [ ] 步骤2: Gateway 接收上传并调用 Task 服务
  - [ ] 步骤3: Task 服务推入任务到 Redis 队列
  - [ ] 步骤4: Processor 拉取任务并执行14步流程
  - [ ] 步骤5: 用户轮询查询任务状态
  - [ ] 步骤6: 任务完成后用户下载结果视频
- [ ] 验证输出视频质量
  - [ ] 音频清晰度
  - [ ] 中文人声音色与原音色的相似度
  - [ ] 背景音保留情况 (如果启用音频分离)
  - [ ] 音画同步性

### Phase 3: 异常场景测试

- [ ] 测试磁盘空间不足场景 (Gateway 应拒绝上传)
- [ ] 测试 Redis 连接失败场景 (服务应返回 503 错误)
- [ ] 测试 AI 服务调用失败场景 (任务状态应更新为 FAILED)
- [ ] 测试应用配置缺失场景 (Processor 步骤2应失败并更新状态)
- [ ] 测试并发处理场景 (maxConcurrency=1，多个任务排队处理)
- [ ] 测试大文件上传场景 (2048MB 视频，验证流式处理)
- [ ] 测试时长对齐所有策略 (静音填充、语速加速、LLM 重译、截断)

### Phase 4: 性能测试

- [ ] 测试系统在 2C4G 服务器上的性能
  - [ ] CPU 占用 <80%
  - [ ] 内存占用 <3GB
  - [ ] 磁盘 I/O 合理
- [ ] 测试处理速度
  - [ ] 10分钟视频处理时间 <30分钟 (目标)
- [ ] 测试并发能力
  - [ ] 10个任务排队处理 (maxConcurrency=1)
  - [ ] 验证任务按顺序处理，无并发冲突

### Phase 5: 文档和演示

- [ ] 编写系统集成测试报告
  - [ ] 测试场景
  - [ ] 测试结果
  - [ ] 已知问题和缓解措施
- [ ] 录制演示视频
  - [ ] 展示完整的视频翻译流程
  - [ ] 展示输出视频质量
- [ ] 编写用户手册
  - [ ] 系统部署指南
  - [ ] API 使用说明
  - [ ] 常见问题 FAQ

### 验收标准

- [ ] 5个服务可成功部署并通信
- [ ] 端到端测试通过 (完整视频翻译流程，14步)
- [ ] 异常场景测试通过 (磁盘空间不足、Redis 连接失败、配置缺失等)
- [ ] 性能测试达标 (2C4G 服务器，内存 <3GB)
- [ ] 输出视频质量可接受 (音频清晰、音色相似、音画同步)
- [ ] 系统集成测试报告完成
- [ ] 演示视频录制完成
- [ ] 用户手册编写完成

---

## 📊 测试文件组织结构

所有测试文件统一存放在 `notes/server/test/` 目录下，按服务模块组织：

```
notes/server/test/
├── audioseparator/              # AudioSeparator 测试
│   ├── test_spleeter_wrapper.py
│   ├── test_separator_service_integration.py
│   └── test_separator_performance.py
├── aiadaptor/                   # AIAdaptor 测试
│   ├── test_adapter_registry.py
│   ├── test_asr_adapters.py
│   ├── test_translation_adapters.py
│   ├── test_llm_adapters.py
│   ├── test_voice_cloning_adapters.py
│   ├── test_voice_cache_manager.py
│   ├── test_redis_integration.py
│   ├── test_external_api_integration.py
│   └── test_adapter_selection.py
├── task/                        # Task 测试
│   ├── test_file_operations.py
│   ├── test_redis_operations.py
│   ├── test_create_task_integration.py
│   ├── test_get_task_status_integration.py
│   └── test_grpc_interface.py
├── processor/                   # Processor 测试
│   ├── test_composer_concatenate.py
│   ├── test_composer_align.py
│   ├── test_composer_merge.py
│   ├── test_mediautil_extract.py
│   ├── test_mediautil_merge.py
│   ├── test_redis_integration.py
│   ├── test_grpc_integration.py
│   └── test_processor_e2e.py
├── gateway/                     # Gateway 测试
│   ├── test_disk_check.py
│   ├── test_path_check.py
│   ├── test_crypto.py
│   ├── test_mime_check.py
│   ├── test_settings.py
│   ├── test_upload.py
│   ├── test_task_status.py
│   ├── test_download.py
│   ├── test_http_api_integration.py
│   ├── test_gateway_e2e.py
│   └── test_gateway_performance.py
└── integration/                 # 系统集成测试
    ├── test_system_e2e.py
    ├── test_video_quality.py
    ├── test_disk_full.py
    ├── test_redis_failure.py
    ├── test_ai_service_failure.py
    ├── test_config_missing.py
    ├── test_concurrent_processing.py
    ├── test_large_file.py
    ├── test_duration_alignment.py
    ├── test_system_performance.py
    ├── test_processing_speed.py
    └── test_concurrent_capacity.py
```

---

## 📈 进度统计

### 总体进度

| 阶段 | 服务 | 总任务数 | 已完成 | 进行中 | 未开始 | 完成率 |
|------|------|----------|--------|--------|--------|--------|
| 阶段一-A | AudioSeparator | 27 | 0 | 0 | 27 | 0% |
| 阶段一-B | AIAdaptor | 58 | 0 | 0 | 58 | 0% |
| 阶段二 | Task | 27 | 0 | 0 | 27 | 0% |
| 阶段三 | Processor | 48 | 0 | 0 | 48 | 0% |
| 阶段四 | Gateway | 58 | 0 | 0 | 58 | 0% |
| 阶段五 | 系统集成测试 | 18 | 0 | 0 | 18 | 0% |
| **总计** | **全部** | **236** | **0** | **0** | **236** | **0%** |

### 当前开发状态

- **当前阶段**: 阶段一-A (AudioSeparator 服务)
- **当前 Phase**: Phase 1 (基础设施搭建)
- **下一个里程碑**: M1-AudioSeparator 服务完成

---

## 📝 更新日志

### 2025-11-03
- 创建 development-todo.md 文档
- 初始化所有5个服务模块的开发任务清单
- 初始化所有测试清单
- 定义任务状态标记规则
- 设置当前开发阶段为 AudioSeparator 服务

---

**文档结束**

