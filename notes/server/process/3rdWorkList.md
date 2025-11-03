# 第三层文档实现工作清单

**文档版本**: 1.0
**创建日期**: 2025-11-02
**关联文档**:
- 第一层：`notes/server/1st/Base-Design.md` v2.2
- 第二层：各服务的 `{服务名}-design.md`
- 规范文档：`notes/server/1st/design-rules.md`

---

## 工作清单说明

本清单严格按照 `design-rules.md` 的要求制定，确保第三层文档的实现符合以下原则：

1. **核心理念**：第三层文档的核心是"代码本身"（在 Git 中）和它的"说明书"（在 .md 文件中，解释"为什么这么写"）
2. **文档定位**：专注于"核心实现决策与上下文"，解释"为什么这么写"，而不是"写了什么"
3. **代码片段使用规则**：
   - 目的：仅用于阐明决策，不是为了被复制粘贴
   - 篇幅：严格限制在 20 行以内
   - 形式：优先使用伪代码，其次才是高度简化的示例代码
   - 前置条件：必须是在代码注释和文字都难以清晰表达其核心思想时才可使用
4. **部署文档约束**：Dockerfile、docker-compose.yml、requirements.txt 等部署文档将在项目完成后、到达部署阶段时专门撰写，不在第三层文档中包含

---

## 工作顺序与优先级

根据服务间的依赖关系和架构层次，第三层文档的实现顺序如下：

### 阶段一：基础服务层（无外部依赖）
1. **AudioSeparator-design-detail.md**（Python 微服务，独立服务）
2. **AIAdaptor-design-detail.md**（Go gRPC 微服务，依赖外部 AI API）

### 阶段二：核心业务层（依赖基础服务）
3. **Processor-design-detail.md**（Go 后台服务，依赖 AIAdaptor 和 AudioSeparator）

### 阶段三：任务管理层（依赖核心业务）
4. **Task-design-detail.md**（Go gRPC 微服务，依赖 Redis）

### 阶段四：接入层（依赖所有服务）
5. **Gateway-design-detail.md**（Go RESTful API，依赖 Task 服务）

---

## 第三层文档模板要求

根据 `design-rules.md` 第 779-856 行的规定，每个第三层文档必须包含以下章节：

### 1. 项目结构
- 仅列出关键文件和目录
- 附简要职责说明
- 不需要列出所有文件（避免维护负担）

### 2. 核心实现决策与上下文（Core Implementation Rationale）
- **核心理念**：解释"为什么这么写"，而不是"写了什么"
- 说明算法选择理由
- 说明库选择理由
- 说明性能优化决策
- 说明错误处理策略

**代码片段使用规则**：
- **目的**：仅用于阐明决策，不是为了被复制粘贴
- **篇幅**：严格限制在 20 行以内
- **形式**：优先使用伪代码，其次才是高度简化的示例代码
- **前置条件**：必须是在代码注释和文字都难以清晰表达其核心思想时才可使用

### 3. 依赖库清单（及选型原因）
- 列出所有第三方依赖
- 说明选择该库的原因
- 说明版本要求

### 4. 构建要求说明
> ⚠️ **注意**：根据"部署阶段文档约束"规则，完整的 Dockerfile 不应包含在第三层文档中，应在部署阶段专门撰写。

- 说明构建工具要求（Go 版本、Python 版本等）
- 说明构建依赖（系统库、编译工具等）
- 说明运行要求（环境变量、配置文件等）

### 5. 测试策略与示例
- 说明测试策略（单元测试、集成测试）
- 提供测试示例
- 说明测试覆盖率要求

### 6. 待实现任务清单（TODO List）
- 列出待实现的功能
- 列出已知的技术债务
- 列出未来的优化方向

---

## 阶段一：基础服务层

### 1. AudioSeparator-design-detail.md

**关联第二层文档**: `notes/server/2nd/AudioSeparator-design.md` v1.4

**实现要点**:

#### 1.1 项目结构
- 列出关键文件：`main.py`、`separator_service.py`、`spleeter_wrapper.py`
- 说明各文件职责

#### 1.2 核心实现决策与上下文
- **Spleeter 模型选择理由**：为什么选择 2stems 模型而非 4stems/5stems
- **模型懒加载策略**：为什么首次调用时加载，而非服务启动时加载
- **并发控制策略**：为什么最大并发数设置为 1
- **错误处理策略**：如何处理模型加载失败、内存不足、处理超时

#### 1.3 依赖库清单（及选型原因）
- grpcio：官方 gRPC Python 实现
- spleeter：开源音频分离模型
- tensorflow：Spleeter 依赖，选择 CPU 版本降低部署门槛

#### 1.4 构建要求说明
- Python 版本要求：3.9+
- 系统依赖：ffmpeg, libsndfile1
- 环境变量：AUDIO_SEPARATOR_MAX_WORKERS

> 📋 **完整的 Dockerfile、requirements.txt**将在部署阶段专门撰写（参考 design-rules.md 第 181-330 行）。

#### 1.5 测试策略与示例
- 单元测试：测试 Spleeter 模型加载
- 集成测试：测试完整的音频分离流程
- 性能测试：测试处理速度和内存占用

#### 1.6 待实现任务清单
- [ ] 实现 gRPC 服务入口（main.py）
- [ ] 实现 Spleeter 模型封装（spleeter_wrapper.py）
- [ ] 实现音频分离逻辑（separator_service.py）
- [ ] 实现错误处理和降级策略
- [ ] 编写单元测试和集成测试

---

### 2. AIAdaptor-design-detail.md

**关联第二层文档**: `notes/server/2nd/AIAdaptor-design.md` v1.5

**实现要点**:

#### 2.1 项目结构
- 列出关键文件：`main.go`、`internal/logic/`、`internal/adapters/`、`internal/voice_cache/`
- 说明各文件职责

#### 2.2 核心实现决策与上下文
- **适配器模式实现**：为什么使用接口 + 注册表模式
- **音色缓存策略**：为什么使用 Redis 缓存音色 ID
- **音色轮询策略**：为什么采用指数退避轮询
- **API 密钥解密策略**：为什么每次调用都从 Redis 读取并解密
- **错误处理策略**：如何处理 API 调用失败、音色注册失败、缓存失效

#### 2.3 依赖库清单（及选型原因）
- go-zero：微服务框架
- grpc：gRPC 服务框架
- go-redis：Redis 客户端
- crypto/aes：API 密钥加密解密

#### 2.4 构建要求说明
- Go 版本要求：1.21+
- 环境变量：AIADAPTOR_GRPC_PORT、REDIS_HOST、REDIS_PORT、API_KEY_ENCRYPTION_SECRET

> 📋 **完整的 Dockerfile**将在部署阶段专门撰写（参考 design-rules.md 第 181-330 行）。

#### 2.5 测试策略与示例
- 单元测试：测试适配器接口实现
- 集成测试：测试完整的 AI 服务调用流程
- Mock 测试：测试适配器选择逻辑

#### 2.6 待实现任务清单
- [ ] 实现 gRPC 服务入口（main.go）
- [ ] 实现适配器注册表（internal/adapters/registry.go）
- [ ] 实现 ASR 适配器（internal/adapters/asr/）
- [ ] 实现翻译适配器（internal/adapters/translation/）
- [ ] 实现 LLM 适配器（internal/adapters/llm/）
- [ ] 实现声音克隆适配器（internal/adapters/voice_cloning/）
- [ ] 实现音色缓存管理器（internal/voice_cache/）
- [ ] 实现 API 密钥解密逻辑（internal/config/）
- [ ] 编写单元测试和集成测试

---

## 阶段二：核心业务层

### 3. Processor-design-detail.md

**关联第二层文档**: `notes/server/2nd/Processor-design.md` v2.5

**实现要点**:

#### 3.1 项目结构
- 列出关键文件：`main.go`、`internal/logic/task_pull_loop.go`、`internal/logic/processor_logic.go`、`internal/composer/`、`internal/mediautil/`
- 说明各文件职责

#### 3.2 核心实现决策与上下文
- **任务拉取策略**：为什么使用定期轮询而非阻塞式拉取
- **并发控制策略**：为什么使用 Channel 信号量而非消息队列
- **时长对齐算法**：为什么采用混合策略（静音填充 + 语速加速）
- **音频拼接策略**：为什么按时间轴顺序拼接
- **错误处理策略**：如何处理 AI 服务调用失败、音频处理失败、文件 I/O 失败

#### 3.3 依赖库清单（及选型原因）
- go-zero：微服务框架
- grpc：gRPC 客户端
- go-redis：Redis 客户端
- ffmpeg：音视频处理（通过 exec.Command 调用）

#### 3.4 构建要求说明
- Go 版本要求：1.21+
- 系统依赖：ffmpeg（>= 4.0）
- 环境变量：PROCESSOR_MAX_CONCURRENCY、LOCAL_STORAGE_PATH、REDIS_HOST、REDIS_PORT

> 📋 **完整的 Dockerfile**将在部署阶段专门撰写（参考 design-rules.md 第 181-330 行）。

#### 3.5 测试策略与示例
- 单元测试：测试音频拼接、时长对齐、音频合成逻辑
- 集成测试：测试完整的 18 步处理流程
- Mock 测试：测试 AI 服务调用逻辑

#### 3.6 待实现任务清单
- [ ] 实现后台服务入口（main.go）
- [ ] 实现任务拉取循环（internal/logic/task_pull_loop.go）
- [ ] 实现主流程编排逻辑（internal/logic/processor_logic.go）
- [ ] 实现音频拼接（internal/composer/concatenate.go）
- [ ] 实现时长对齐（internal/composer/align.go）
- [ ] 实现音频合成（internal/composer/merge.go）
- [ ] 实现媒体工具包（internal/mediautil/）
- [ ] 实现并发控制逻辑
- [ ] 实现 GC 定时任务
- [ ] 编写单元测试和集成测试

---

## 阶段三：任务管理层

### 4. Task-design-detail.md

**关联第二层文档**: `notes/server/2nd/Task-design.md` v1.5

**实现要点**:

#### 4.1 项目结构
- 列出关键文件：`main.go`、`internal/logic/task_logic.go`、`internal/svc/service_context.go`
- 说明各文件职责

#### 4.2 核心实现决策与上下文
- **文件交接策略**：为什么使用 os.Rename 而非 io.Copy
- **任务 ID 生成策略**：为什么使用 UUID 而非自增 ID
- **Redis 队列选择**：为什么使用 List 而非 Stream
- **错误处理策略**：如何处理文件交接失败、Redis 写入失败

#### 4.3 依赖库清单（及选型原因）
- go-zero：微服务框架
- grpc：gRPC 服务框架
- go-redis：Redis 客户端
- google/uuid：UUID 生成

#### 4.4 构建要求说明
- Go 版本要求：1.21+
- 环境变量：TASK_GRPC_PORT、REDIS_HOST、REDIS_PORT、LOCAL_STORAGE_PATH

> 📋 **完整的 Dockerfile**将在部署阶段专门撰写（参考 design-rules.md 第 181-330 行）。

#### 4.5 测试策略与示例
- 单元测试：测试文件交接逻辑
- 集成测试：测试完整的任务创建流程
- Mock 测试：测试 Redis 操作逻辑

#### 4.6 待实现任务清单
- [ ] 实现 gRPC 服务入口（main.go）
- [ ] 实现任务创建逻辑（internal/logic/create_task_logic.go）
- [ ] 实现任务状态查询逻辑（internal/logic/get_task_status_logic.go）
- [ ] 实现文件交接逻辑
- [ ] 实现 Redis 队列操作
- [ ] 编写单元测试和集成测试

---

## 阶段四：接入层

### 5. Gateway-design-detail.md

**关联第二层文档**: `notes/server/2nd/Gateway-design.md` v5.8

**实现要点**:

#### 5.1 项目结构
- 列出关键文件：`main.go`、`internal/handler/`、`internal/logic/`、`internal/svc/service_context.go`
- 说明各文件职责

#### 5.2 核心实现决策与上下文
- **文件流式处理策略**：为什么使用 io.Copy 而非 ioutil.ReadAll
- **磁盘空间预检策略**：为什么使用 `availableSpace >= fileSize * 3 + 500MB` 公式
- **API Key 加密策略**：为什么使用 AES-256-GCM 而非 AES-256-CBC
- **API Key 脱敏策略**：为什么使用 `前缀-***-后6位` 格式
- **乐观锁策略**：为什么使用版本号而非时间戳
- **错误处理策略**：如何处理文件上传失败、磁盘空间不足、Redis 写入失败

#### 5.3 依赖库清单（及选型原因）
- go-zero：微服务框架
- grpc：gRPC 客户端
- go-redis：Redis 客户端
- crypto/aes：API 密钥加密解密

#### 5.4 构建要求说明
- Go 版本要求：1.21+
- 环境变量：GATEWAY_PORT、TASK_RPC_ADDRESS、REDIS_HOST、REDIS_PORT、LOCAL_STORAGE_PATH、API_KEY_ENCRYPTION_SECRET

> 📋 **完整的 Dockerfile**将在部署阶段专门撰写（参考 design-rules.md 第 181-330 行）。

#### 5.5 测试策略与示例
- 单元测试：测试文件流式处理逻辑
- 集成测试：测试完整的文件上传流程
- Mock 测试：测试 gRPC 客户端调用逻辑

#### 5.6 待实现任务清单
- [ ] 实现 RESTful API 服务入口（main.go）
- [ ] 实现文件上传 Handler（internal/handler/upload_task_handler.go）
- [ ] 实现任务状态查询 Handler（internal/handler/get_task_status_handler.go）
- [ ] 实现文件下载 Handler（internal/handler/download_file_handler.go）
- [ ] 实现配置读取 Handler（internal/handler/get_settings_handler.go）
- [ ] 实现配置更新 Handler（internal/handler/update_settings_handler.go）
- [ ] 实现文件流式处理逻辑
- [ ] 实现磁盘空间预检逻辑
- [ ] 实现 API Key 加密解密逻辑
- [ ] 实现乐观锁逻辑
- [ ] 编写单元测试和集成测试

---

## 工作流程规范

根据 `design-rules.md` 第 415-636 行的规定，每个第三层文档的实现必须遵循以下流程：

### 1. 工作前审视流程

在开始任何第三层文档设计或更新工作之前，必须执行以下审视流程：

#### 1.1 回顾上级文档
- 完整阅读对应的第二层文档（如 `Processor-design.md`）
- 确保理解接口契约和逻辑步骤

#### 1.2 确保方向一致
- 检查第二层文档的版本号和更新日期
- 确认第二层文档的核心设计决策
- 确认服务边界和职责划分
- 确认接口定义和数据结构

#### 1.3 总体说明更新计划
在开始工作前，必须提供总体说明，包括：
- **更新目标**：本次更新的核心目标是什么？
- **主要变更**：将进行哪些主要变更？
- **与现有文档的差异**：与第二层文档的主要差异是什么？
- **更新计划**：将按照什么顺序执行更新？

### 2. 工作后审查流程

在完成任何第三层文档设计或更新工作后，必须执行以下审查流程：

#### 2.1 批判性审核
以批判性的眼光审核文档内容，提出以下问题：
- **完整性**：是否覆盖了所有必要的章节？
- **准确性**：是否与第二层文档一致？是否有逻辑错误？
- **清晰性**：是否易于理解？是否有歧义？
- **可执行性**：是否可以直接指导代码开发？

#### 2.2 检查层次分明
- **第三层文档**：是否专注于"为什么这么写"，而不是"写了什么"？

#### 2.3 检查内容准确性
- **实现决策**：是否与第二层文档的关键逻辑步骤一致？
- **依赖库清单**：是否完整？
- **构建要求**：是否准确？
- **测试策略**：是否合理？

#### 2.4 总体说明更新差异
在完成工作后，必须提供总体说明，包括：
- **更新内容总结**：本次更新了哪些内容？
- **主要变更点**：与之前版本的主要差异是什么？
- **遗留问题**：是否有待解决的问题？
- **后续工作**：下一步需要做什么？

---

## 文档回溯更新规则

根据 `design-rules.md` 第 566-636 行的规定，在代码开发过程中，如果发现第二层或第三层文档的设计有问题，必须遵循以下规则：

### 1. 破坏性变更

以下变更属于**破坏性变更**，必须发起紧急评审：
- **API 接口签名变更**：输入输出参数的增删改
- **gRPC Proto 定义变更**：message 或 service 的变更
- **核心数据结构变更**：Redis、数据库中的数据结构变更
- **服务间交互协议变更**：调用方式、调用顺序的变更
- **错误码定义变更**：错误码的增删改

**处理流程**：
1. **暂停开发**：立即停止当前的代码开发工作
2. **发起紧急辩论评审**：在对话中说明问题和建议的变更方案
3. **达成新共识**：与项目发起者一起辩证性评估，达成新的设计共识
4. **更新文档**：由项目发起者更新第二层文档，并提交版本控制
5. **继续开发**：基于新的设计继续开发

### 2. 非破坏性优化

以下变更属于**非破坏性优化**，可以直接实现：
- **内部算法优化**：不影响接口的算法改进
- **性能优化**：缓存策略、并发优化等
- **日志增强**：增加日志输出
- **错误处理细化**：更详细的错误信息
- **代码重构**：不影响接口的代码结构调整

**处理流程**：
1. **直接优化**：在代码中直接实现优化
2. **代码提交时提供第三层文档更新说明**：在提交代码时，一并提供对第三层文档的更新说明
3. **由项目发起者一并提交**：项目发起者审查后，一并提交代码和文档

---

## 检查清单

根据 `design-rules.md` 第 1-228 行的 REVIEW-CHECKLIST.md，每个第三层文档完成后必须通过以下检查：

### 第三层文档审查清单

#### 1. 完整性检查
- [ ] 是否包含项目结构（关键文件和目录）？
- [ ] 是否包含核心实现决策与上下文？
- [ ] 是否包含依赖库清单（及选型原因）？
- [ ] 是否包含构建要求说明？
- [ ] 是否包含测试策略与示例？
- [ ] 是否包含待实现任务清单（TODO List）？

#### 2. 层次分明检查
- [ ] 核心实现决策是否专注于"为什么这么写"，而不是"写了什么"？
- [ ] 是否避免了大段的代码复制粘贴？
- [ ] 项目结构是否仅列出关键文件（而非所有文件）？

#### 3. 准确性检查
- [ ] 核心实现决策是否与第二层文档的关键逻辑步骤一致？
- [ ] 依赖库清单是否完整？
- [ ] 构建要求是否准确？
- [ ] 测试策略是否合理？

#### 4. 清晰性检查
- [ ] 核心实现决策是否易于理解？
- [ ] 算法选择理由是否充分？
- [ ] 库选择理由是否充分？
- [ ] 是否有歧义或模糊的表述？

#### 5. 代码片段使用规则检查
- [ ] 代码片段是否仅用于阐明决策（而非复制粘贴）？
- [ ] 代码片段是否严格限制在 20 行以内？
- [ ] 是否优先使用伪代码，其次才是简化示例代码？
- [ ] 代码片段是否只在代码注释和文字都难以清晰表达时才使用？

#### 6. 一致性检查
- [ ] 是否与第二层文档（{服务名}-design.md）一致？
- [ ] 是否与第一层文档（Base-Design.md）一致？
- [ ] 是否与相关的架构决策记录（ADR）一致？
- [ ] 各章节之间是否一致？

---

## 总结

本工作清单严格按照 `design-rules.md` 的要求制定，确保第三层文档的实现符合以下原则：

1. **核心理念**：专注于"核心实现决策与上下文"，解释"为什么这么写"
2. **代码片段使用规则**：严格限制在 20 行以内，优先使用伪代码
3. **部署文档约束**：Dockerfile、docker-compose.yml、requirements.txt 等部署文档将在部署阶段专门撰写
4. **工作流程规范**：遵循工作前审视流程和工作后审查流程
5. **文档回溯更新规则**：区分破坏性变更和非破坏性优化，采用不同的处理流程

**实施顺序**：
1. AudioSeparator-design-detail.md（Python 微服务，独立服务）
2. AIAdaptor-design-detail.md（Go gRPC 微服务，依赖外部 AI API）
3. Processor-design-detail.md（Go 后台服务，依赖 AIAdaptor 和 AudioSeparator）
4. Task-design-detail.md（Go gRPC 微服务，依赖 Redis）
5. Gateway-design-detail.md（Go RESTful API，依赖 Task 服务）

**下一步行动**：
- 按照上述顺序，逐个实现第三层文档
- 每个文档完成后，使用检查清单进行自查
- 发起评审，确保文档质量

---

**文档变更历史**

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 1.0 | 2025-11-02 | 初始版本，根据第一层和第二层文档制定第三层文档实现工作清单 |
