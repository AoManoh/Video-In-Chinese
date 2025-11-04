# 自定义 API 服务配置指南

**版本**: 1.0  
**创建日期**: 2025-11-04

---

## 概述

本系统支持用户自定义 API 服务配置，允许通过第三方代理服务或自建服务来调用 AI 模型。这对于以下场景特别有用：

- 🌐 使用第三方代理服务（如 gemini-balance、one-api、new-api 等）
- 🏢 使用企业内部的 API 网关
- 🔧 配置自建的 OpenAI 兼容服务
- 💰 通过负载均衡或轮询服务降低成本

---

## 支持的服务类型

### 1. 翻译服务（Translation）

#### 方式一：使用 OpenAI 格式代理
如果您通过代理服务使用 Gemini 或其他大模型进行翻译：

1. **服务商**：选择 `自定义 OpenAI 格式 API`
2. **API 密钥**：填入代理服务提供的密钥（如 `sk-xxx`）
3. **自定义端点**：填入代理服务地址（如 `https://gemini-balance.xxx.com`）

**示例配置**：
```
服务商: 自定义 OpenAI 格式 API
API 密钥: sk-proj-abc123xyz789
自定义端点: https://gemini-balance.example.com
```

#### 方式二：使用官方服务 + 自定义端点
如果您想使用官方 API 但通过自己的网关：

1. **服务商**：选择官方服务商（如 `Google Gemini`）
2. **API 密钥**：填入官方 API 密钥
3. **自定义端点**：填入您的网关地址

---

### 2. 文本润色（Polishing）

文本润色服务支持使用 OpenAI 格式的 API：

1. **启用文本润色**：打开开关
2. **服务商**：
   - 选择 `OpenAI GPT-4o` 使用官方 OpenAI API
   - 选择 `自定义 OpenAI 格式` 使用代理服务
3. **API 密钥**：填入相应的 API 密钥
4. **自定义端点**（仅当选择 OpenAI 相关服务时显示）：
   - 留空则使用官方端点 `https://api.openai.com`
   - 填入自定义地址以使用代理服务

**代理服务示例**：
```
服务商: 自定义 OpenAI 格式
API 密钥: sk-custom-key-123
自定义端点: https://api.your-proxy.com
```

**官方 API + 代理示例**：
```
服务商: OpenAI GPT-4o
API 密钥: sk-proj-openai-key
自定义端点: https://openai-gateway.your-company.com
```

---

### 3. 译文优化（Optimization）

配置方式与文本润色相同：

1. **启用译文优化**：打开开关
2. **服务商**：选择 `OpenAI GPT-4o` 或 `自定义 OpenAI 格式`
3. **API 密钥** 和 **自定义端点**：根据实际情况填写

---

### 4. ASR 语音识别（ASR）

ASR 服务也支持自定义端点：

1. **服务商**：选择服务商（如 `OpenAI Whisper`）
2. **API 密钥**：填入 API 密钥
3. **自定义端点**：
   - 留空使用官方端点
   - 填入自定义地址以使用代理服务或企业网关

---

### 5. 声音克隆（Voice Cloning）

声音克隆服务同样支持自定义端点：

1. **服务商**：选择服务商（如 `阿里云 CosyVoice`）
2. **API 密钥**：填入 API 密钥
3. **自定义端点**：根据需要配置

---

## 常见代理服务示例

### 1. gemini-balance

**用途**：通过 OpenAI 格式调用 Google Gemini  
**配置示例**：
```
服务商: 自定义 OpenAI 格式 API
API 密钥: sk-balance-xxx
自定义端点: https://gemini-balance.your-domain.com
```

### 2. one-api / new-api

**用途**：统一管理多个 AI 服务商的密钥  
**配置示例**：
```
服务商: 自定义 OpenAI 格式 API
API 密钥: sk-oneapi-xxx
自定义端点: https://api.your-oneapi.com
```

### 3. 企业 API 网关

**用途**：通过企业内部网关统一管理 API 调用  
**配置示例**：
```
服务商: OpenAI GPT-4o（或其他官方服务）
API 密钥: 企业内部分配的密钥
自定义端点: https://ai-gateway.your-company.com
```

---

## OpenAI 格式 API 说明

当选择"自定义 OpenAI 格式 API"时，系统会：

1. 使用 OpenAI Chat Completions API 格式发送请求
2. 请求路径为：`{自定义端点}/v1/chat/completions`
3. 认证方式：`Authorization: Bearer {API密钥}`
4. 请求体格式：
```json
{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "系统提示"},
    {"role": "user", "content": "用户输入"}
  ],
  "temperature": 0.7,
  "max_tokens": 2048,
  "top_p": 0.9
}
```

这意味着任何兼容 OpenAI API 格式的服务都可以使用，包括：
- 官方 OpenAI API
- gemini-balance（Gemini 转 OpenAI 格式）
- one-api / new-api（多服务商统一接口）
- LocalAI、Ollama 等本地部署方案
- 各种自建的兼容服务

---

## 配置验证

### 自动格式验证（Phase 1）

系统会在保存配置时自动验证以下内容：

#### 1. API 密钥格式验证

不同服务商的 API 密钥有不同的格式要求：

| 服务商 | 格式要求 | 示例 |
|--------|---------|------|
| **OpenAI** | 以 `sk-` 开头，至少 20 个字符 | `sk-proj-abc123...` |
| **Google Gemini** | 39 个字符的字母数字组合 | `AIzaSyAbc123...` |
| **阿里云** | 16-30 个字符 | `LTAI5t...` |
| **Azure** | 32 个字符的十六进制字符串 | `abc123def456...` |
| **DeepL** | 以 `:fx` 结尾的 UUID 格式 | `12345678-abcd-...:fx` |
| **Claude** | 以 `sk-ant-` 开头 | `sk-ant-api03-...` |
| **自定义服务** | 通常以 `sk-` 开头，至少 10 个字符 | `sk-custom-...` |

**验证时机**：
- ✅ 保存配置时自动验证
- ✅ 格式错误时会弹出详细提示
- ✅ 包含建议的获取方式链接

**示例错误提示**：
```
❌ ASR API 密钥格式错误：
OpenAI API 密钥格式：sk-xxx，至少 20 个字符（当前长度：15）

💡 获取方式：访问 https://platform.openai.com/api-keys 创建 API 密钥
```

#### 2. 自定义端点格式验证

系统会验证自定义端点的 URL 格式：

**验证规则**：
- ✅ 必须以 `http://` 或 `https://` 开头
- ✅ 必须包含有效的域名
- ⚠️ 使用 `http://`（非 localhost）时会警告

**安全建议**：
```
⚠️ 检测到您使用了 HTTP 协议的自定义端点。
为保护 API 密钥安全，强烈建议使用 HTTPS 协议。
是否继续保存？
```

#### 3. 必填字段检查

系统要求至少配置以下三个核心服务：
- ✅ ASR 服务商和 API 密钥
- ✅ 翻译服务商和 API 密钥
- ✅ 声音克隆服务商和 API 密钥

**验证结果**：
```
发现以下配置问题：
❌ 翻译 API 密钥未配置
⚠️ 文本润色 API 密钥格式错误：应以 sk- 开头

💡 提示：配置错误可能导致任务处理失败，请仔细检查后重新保存
```

---

### 任务失败时的智能提示（Phase 1）

当任务因配置问题失败时，系统会：

#### 1. 识别配置相关错误

系统会自动识别以下错误类型：
- `401/403 错误` → API 密钥无效或权限不足
- `429 错误` → API 配额不足
- 包含"API 密钥"关键词的错误
- 包含"配置"关键词的错误

#### 2. 提供解决建议

**示例 1：API 密钥无效**
```
❌ 任务失败
任务 abc12345 处理失败：ASR 失败：API 密钥无效 (HTTP 401)

🔧 API 密钥无效或已过期，请在配置页面检查并更新密钥

💡 建议前往"服务配置"页面检查 API 密钥设置
```

**示例 2：API 配额不足**
```
❌ 任务失败
任务 abc12345 处理失败：翻译失败：API 配额不足 (HTTP 429)

🔧 API 配额不足或请求频率过高，请检查账户配额或稍后重试

💡 建议前往"服务配置"页面检查 API 密钥设置
```

**示例 3：配置解密失败**
```
❌ 任务失败
任务 abc12345 处理失败：配置解密失败

🔧 配置数据可能损坏，请重新保存配置

💡 建议前往"服务配置"页面检查 API 密钥设置
```

#### 3. 自动引导修复

系统会在失败通知 1 秒后，自动弹出提示：
```
⚠️ 建议前往"服务配置"页面检查 API 密钥设置
```

用户点击后可直接跳转到配置页面。

---

### 手动测试方法

虽然 Phase 1 不提供"测试连接"按钮，但您可以通过以下方式验证配置：

#### 方法 1：上传小视频测试

1. 准备一个 10-30 秒的小视频文件
2. 保存配置后，上传该视频
3. 等待任务处理（通常 1-3 分钟）
4. 如果配置正确，任务会成功完成
5. 如果配置错误，系统会显示详细的错误信息和解决建议

**建议测试视频**：
- 时长：10-30 秒
- 格式：MP4
- 大小：< 50MB
- 内容：包含清晰的英文语音

#### 方法 2：查看官方文档验证密钥

在保存前，可以访问服务商官网验证密钥格式：

- **OpenAI**: https://platform.openai.com/api-keys
- **Google Gemini**: https://makersuite.google.com/app/apikey
- **阿里云**: https://ram.console.aliyun.com/manage/ak
- **Azure**: https://portal.azure.com/
- **DeepL**: https://www.deepl.com/pro-api

---

### 配置保存后的验证

配置完成后，建议：

1. **保存配置**：点击"保存配置"按钮
   - 系统会自动验证格式
   - 如有问题会立即提示

2. **测试功能**：上传一个小视频进行测试
   - 这是最准确的验证方式
   - 会调用所有配置的服务

3. **查看错误**：如果失败，查看错误信息
   - 系统会自动识别配置问题
   - 提供详细的解决建议

常见错误：
- `401/403 错误`：API 密钥无效或过期 → 🔧 检查并更新密钥
- `404 错误`：自定义端点地址错误 → 🔧 检查端点 URL 是否正确
- `429 错误`：API 配额不足或请求频率过高 → 🔧 检查账户配额
- `5xx 错误`：代理服务或上游 API 服务错误 → 🔧 检查服务状态

---

## 安全建议

1. **保护 API 密钥**：
   - 不要在公共场合分享您的 API 密钥
   - 定期轮换 API 密钥
   - 使用具有最小权限的密钥

2. **使用 HTTPS**：
   - 确保自定义端点使用 HTTPS 协议
   - 验证代理服务的 SSL 证书有效性

3. **企业部署**：
   - 优先使用企业内部网关
   - 配置访问控制和审计日志
   - 定期检查 API 使用情况

---

## 技术细节

### 后端支持

后端的 OpenAI LLM Adapter 实现了完整的自定义端点支持：

```go
// 如果提供了自定义端点，使用自定义端点
apiEndpoint := endpoint
if apiEndpoint == "" {
    // 使用默认端点（OpenAI 官方 API）
    apiEndpoint = "https://api.openai.com"
}

// 移除末尾的斜杠并拼接完整路径
apiEndpoint = strings.TrimSuffix(apiEndpoint, "/")
fullEndpoint := apiEndpoint + "/v1/chat/completions"
```

### 认证方式

不同服务使用不同的认证方式：

- **OpenAI 格式**：`Authorization: Bearer {API密钥}`
- **Google Gemini**：URL 参数或 `X-Goog-Api-Key` 头
- **Google Translation**：`X-Goog-Api-Key` 头
- **阿里云服务**：使用 SDK 的标准认证

当使用"自定义 OpenAI 格式"时，统一使用 Bearer Token 认证。

---

## 常见问题（FAQ）

### Q1: 我的代理服务需要特殊的模型名称怎么办？

A: 目前系统使用固定的模型名称（如 `gpt-4o`）。如果您的代理服务支持模型映射，请在代理服务端配置模型别名。未来版本可能会支持自定义模型名称。

### Q2: 自定义端点是否支持 HTTP 协议？

A: 技术上支持，但强烈建议使用 HTTPS 以保护 API 密钥安全。

### Q3: 可以同时配置多个服务使用不同的代理吗？

A: 可以。每个服务（ASR、翻译、润色、优化、声音克隆）都可以独立配置自定义端点。

### Q4: 配置后是否会影响正在处理的任务？

A: 配置修改会立即生效。建议在没有正在处理的任务时修改配置。

### Q5: 如何测试自定义端点是否配置正确？

A: 保存配置后，上传一个小视频进行测试。系统会在处理过程中调用相应的 API，通过结果判断配置是否正确。

---

## 相关文档

- [配置管理页面设计](./2nd/Settings-Page-Design.md)
- [API 接口类型定义](./2nd/API-Types.md)
- [后端 AI Adaptor 设计](../../server/2nd/AIAdaptor-design.md)

---

**文档结束**

