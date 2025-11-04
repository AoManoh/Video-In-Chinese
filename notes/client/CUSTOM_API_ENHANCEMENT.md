# 自定义 API 服务支持增强说明

**版本**: 1.0  
**创建日期**: 2025-11-04  
**类型**: 功能增强

---

## 背景

用户反馈当前前端页面不支持用户集成的自定义 API 服务，例如通过代理服务使用 OpenAI 格式调用 Gemini 模型（如 gemini-balance 轮询服务）。用户希望能够通过简单配置域名 + 密钥的形式就能调用大模型。

---

## 现状分析

### 后端支持情况 ✅

经过代码审查，后端已经完整支持自定义 endpoint：

1. **OpenAI LLM Adapter** (`server/mcp/ai_adaptor/internal/adapters/llm/openai.go`)
   - 完美支持自定义 endpoint
   - 自动拼接 `/v1/chat/completions` 路径
   - 使用标准 OpenAI 认证方式（Bearer token）
   - 兼容所有 OpenAI 格式的代理服务

2. **Gemini LLM Adapter** (`server/mcp/ai_adaptor/internal/adapters/llm/gemini.go`)
   - 支持自定义 endpoint
   - 可用于自建 Gemini 代理服务

3. **Google Translation Adapter** (`server/mcp/ai_adaptor/internal/adapters/translation/google.go`)
   - 支持自定义 endpoint
   - 可配置企业网关

4. **配置管理器** (`server/mcp/ai_adaptor/internal/config/manager.go`)
   - 已有 `endpoint` 字段的解析和存储逻辑
   - 支持加密存储 API 密钥

### 前端问题 ❌

1. **缺少明确的自定义选项**：
   - 虽然有 `endpoint` 字段，但只标记为"可选"
   - 没有"自定义 API 服务"的明确选项
   - 用户不知道如何配置代理服务

2. **缺少说明和引导**：
   - endpoint 字段没有使用说明
   - 没有示例和提示
   - 用户不清楚自定义端点的用途

---

## 解决方案

### 1. 前端增强

#### 1.1 添加"自定义 OpenAI 格式 API"选项

**文件**: `client/src/views/SettingsView.vue`

**翻译服务**：
```vue
<el-option label="Google Gemini（推荐）" value="google-gemini" />
<el-option label="自定义 OpenAI 格式 API" value="openai-compatible" />
<el-option label="DeepL" value="deepl" />
<!-- 其他选项 -->
```

**文本润色服务**：
```vue
<el-option label="OpenAI GPT-4o" value="openai-gpt4o" />
<el-option label="自定义 OpenAI 格式" value="openai-compatible" />
<el-option label="Claude 3.5" value="claude-3.5" />
<el-option label="Google Gemini" value="google-gemini" />
```

**译文优化服务**：
```vue
<el-option label="OpenAI GPT-4o" value="openai-gpt4o" />
<el-option label="自定义 OpenAI 格式" value="openai-compatible" />
<el-option label="Claude 3.5" value="claude-3.5" />
<el-option label="Google Gemini" value="google-gemini" />
```

#### 1.2 优化自定义端点字段

为所有服务的 endpoint 字段添加：

1. **更清晰的占位符**：
   ```
   placeholder="例如: https://gemini-balance.xxx.com"
   ```

2. **详细的提示信息**（通过 tooltip）：
   ```vue
   <el-tooltip placement="top">
     <template #content>
       <div style="max-width: 300px">
         <p style="margin: 0 0 8px 0; font-weight: 600;">自定义端点用途：</p>
         <p style="margin: 0 0 8px 0;">• 使用第三方代理服务（如 gemini-balance、one-api 等）</p>
         <p style="margin: 0 0 8px 0;">• 使用企业内部的 API 网关</p>
         <p style="margin: 0 0 8px 0;">• 配置自建的 OpenAI 兼容服务</p>
         <p style="margin: 0; color: #909399; font-size: 12px;">留空则使用官方默认端点</p>
       </div>
     </template>
     <el-icon><QuestionFilled /></el-icon>
   </el-tooltip>
   ```

3. **条件显示**（针对 LLM 服务）：
   - 当选择 OpenAI 或自定义格式时才显示 endpoint 字段
   - 避免不必要的配置项干扰

#### 1.3 更新数据模型

**文件**: `client/src/api/types.ts`

添加缺失的 endpoint 字段：
```typescript
export interface GetSettingsResponse {
  // ... 其他字段
  polishing_endpoint?: string      // 新增
  optimization_endpoint?: string   // 新增
}

export interface UpdateSettingsRequest {
  // ... 其他字段
  polishing_endpoint?: string      // 新增
  optimization_endpoint?: string   // 新增
}
```

#### 1.4 更新表单初始化

**文件**: `client/src/views/SettingsView.vue` (script 部分)

```typescript
form.value = {
  // ... 其他字段
  polishing_endpoint: settings.value.polishing_endpoint || '',
  optimization_endpoint: settings.value.optimization_endpoint || '',
}
```

### 2. 用户指南

**文件**: `notes/client/CUSTOM_API_GUIDE.md`

创建完整的自定义 API 配置指南，包括：

1. **概述**：说明自定义 API 的用途和应用场景
2. **支持的服务类型**：逐一说明每种服务的配置方法
3. **常见代理服务示例**：
   - gemini-balance
   - one-api / new-api
   - 企业 API 网关
4. **OpenAI 格式 API 说明**：详细说明请求格式和认证方式
5. **配置验证**：如何测试配置是否正确
6. **安全建议**：API 密钥保护和最佳实践
7. **技术细节**：后端实现说明
8. **常见问题 FAQ**

---

## 实施内容

### 修改的文件

1. **client/src/views/SettingsView.vue**
   - 添加"自定义 OpenAI 格式 API"选项（3处）
   - 优化所有服务的 endpoint 字段显示（5处）
   - 添加详细的 tooltip 说明
   - 更新表单数据初始化逻辑

2. **client/src/api/types.ts**
   - 在 `GetSettingsResponse` 接口添加 2 个 endpoint 字段
   - 在 `UpdateSettingsRequest` 接口添加 2 个 endpoint 字段

### 新增的文件

1. **notes/client/CUSTOM_API_GUIDE.md**
   - 完整的用户配置指南
   - 200+ 行详细说明

2. **notes/client/CUSTOM_API_ENHANCEMENT.md**（本文档）
   - 功能增强说明
   - 技术实施细节

---

## 使用示例

### 示例 1: 使用 gemini-balance 代理服务

用户通过 gemini-balance 服务以 OpenAI 格式调用 Gemini：

**配置步骤**：
1. 进入"服务配置"页面
2. 在"翻译服务"部分：
   - **服务商**：选择"自定义 OpenAI 格式 API"
   - **API 密钥**：输入 `sk-balance-xxx`（代理服务提供的密钥）
   - **自定义端点**：输入 `https://gemini-balance.example.com`
3. 点击"保存配置"

**后端处理**：
- 系统会调用 OpenAI LLM Adapter
- 请求发送到：`https://gemini-balance.example.com/v1/chat/completions`
- 使用 Bearer Token 认证：`Authorization: Bearer sk-balance-xxx`
- 代理服务将请求转发给 Google Gemini

### 示例 2: 使用企业内部 OpenAI 网关

企业用户通过内部网关访问 OpenAI API：

**配置步骤**：
1. 在"文本润色"部分：
   - **启用文本润色**：打开
   - **服务商**：选择"OpenAI GPT-4o"
   - **API 密钥**：输入企业内部分配的密钥
   - **自定义端点**：输入 `https://ai-gateway.company.com`
2. 保存配置

**后端处理**：
- 请求发送到：`https://ai-gateway.company.com/v1/chat/completions`
- 企业网关验证密钥并转发到 OpenAI
- 支持审计、限流等企业级功能

### 示例 3: 使用官方 API（无自定义端点）

**配置步骤**：
1. **服务商**：选择"OpenAI GPT-4o"
2. **API 密钥**：输入 OpenAI 官方密钥
3. **自定义端点**：留空

**后端处理**：
- 使用默认端点：`https://api.openai.com/v1/chat/completions`
- 直接调用 OpenAI 官方 API

---

## 技术优势

### 1. 后端已就绪
- 无需修改后端代码
- 后端 OpenAI Adapter 已完美支持自定义 endpoint
- 配置管理器已支持 endpoint 字段的存储和解密

### 2. 前端增强简洁
- 只需添加选项和说明
- 不改变现有数据流和验证逻辑
- 向后兼容现有配置

### 3. 用户体验改善
- 清晰的选项和说明
- 详细的使用指南
- 降低配置门槛

### 4. 灵活性强
- 支持多种代理服务
- 支持企业网关
- 支持自建服务

---

## 兼容性

### 向后兼容性
- ✅ 现有配置不受影响
- ✅ 新增字段为可选字段
- ✅ 不改变现有 API 接口

### 代理服务兼容性
理论上兼容所有 OpenAI 格式的服务，包括但不限于：
- gemini-balance（Gemini → OpenAI 格式）
- one-api / new-api（多服务商统一接口）
- FastGPT（企业级 GPT 应用）
- LocalAI（本地部署）
- Ollama（本地大模型）
- 各种自建的兼容服务

---

## 测试建议

### 1. 功能测试

**测试用例 1: 使用官方 API**
- 配置：OpenAI GPT-4o，官方密钥，endpoint 留空
- 预期：正常调用 OpenAI 官方 API

**测试用例 2: 使用自定义端点**
- 配置：自定义 OpenAI 格式，代理密钥，自定义 endpoint
- 预期：请求发送到自定义端点

**测试用例 3: 配置保存和加载**
- 操作：保存配置后刷新页面
- 预期：配置正确加载并显示

### 2. UI 测试

- ✅ 选项显示正确
- ✅ Tooltip 提示清晰可读
- ✅ 占位符文本有帮助
- ✅ 条件显示逻辑正确（LLM endpoint 字段）

### 3. 错误处理测试

- ❌ 错误的端点地址（404）
- ❌ 无效的 API 密钥（401）
- ❌ 网络超时（timeout）

---

## 后续优化建议

### 短期（v1.1）
1. **模型名称配置**：允许用户自定义模型名称（如 `gemini-1.5-pro`）
2. **连接测试**：添加"测试连接"按钮验证配置
3. **错误提示优化**：更详细的错误信息和修复建议

### 中期（v1.2）
1. **配置模板**：提供常见代理服务的配置模板
2. **批量配置**：允许一键配置所有服务使用同一代理
3. **配置导入/导出**：支持配置的备份和迁移

### 长期（v2.0）
1. **多配置切换**：支持保存多套配置并快速切换
2. **负载均衡**：支持配置多个端点进行负载均衡
3. **API 使用监控**：显示 API 调用次数、成本等统计

---

## 文档更新

本次增强涉及的文档：

1. ✅ **CUSTOM_API_GUIDE.md**（新增）
   - 用户配置指南
   - 详细使用说明

2. ✅ **CUSTOM_API_ENHANCEMENT.md**（本文档，新增）
   - 技术实施说明
   - 开发者参考

3. 🔄 **Settings-Page-Design.md**（待更新）
   - 需要更新服务商选项列表
   - 需要添加自定义 API 的说明

---

## 总结

本次增强通过以下方式解决了用户需求：

1. **添加明确的自定义选项**：用户可以直接选择"自定义 OpenAI 格式 API"
2. **提供清晰的说明**：通过 tooltip 和占位符帮助用户理解配置
3. **创建详细指南**：200+ 行的配置指南涵盖各种使用场景
4. **保持简洁设计**：只在需要时显示相关配置项

**核心价值**：
- ✅ 降低配置门槛
- ✅ 支持更多使用场景
- ✅ 兼容各种代理服务
- ✅ 保持系统灵活性

用户现在可以轻松配置 `域名 + 密钥` 来使用任何 OpenAI 兼容的服务，包括 gemini-balance、one-api 等代理服务。

---

**文档结束**

