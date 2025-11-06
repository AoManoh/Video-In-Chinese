# API 密钥配置清单

**文档版本**: v1.0  
**创建日期**: 2025-11-06  
**用途**: 集成测试 API 密钥配置清单

---

## 使用说明

1. 请在下方表格中填写您的 API 密钥信息
2. 填写完成后，将信息复制到 `scripts/setup_redis_config_template.ps1` 脚本中
3. 执行脚本将配置写入 Redis

---

## 必需的 API 密钥（P0）

### 1. 阿里云 ASR（语音识别）

**用途**: Processor 服务步骤 4 - ASR（语音识别）

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `asr_vendor` | `aliyun` | 固定值 |
| `aliyun_asr_app_key` | `_______________` | 请填写 |
| `aliyun_asr_access_key_id` | `_______________` | 请填写 |
| `aliyun_asr_access_key_secret` | `_______________` | 请填写 |
| `asr_language_code` | `zh-CN` | 中文: zh-CN, 英文: en-US |
| `asr_region` | `cn-shanghai` | 推荐: cn-shanghai |

**获取方式**：
- 登录阿里云控制台：https://www.aliyun.com/
- 进入"智能语音交互"产品页面
- 创建应用并获取 App Key
- 创建 AccessKey（RAM 用户）

---

### 2. Google 翻译

**用途**: Processor 服务步骤 7 - 翻译

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `translation_vendor` | `google` | 固定值 |
| `google_translation_api_key` | `_______________` | 请填写 |

**获取方式**：
- 登录 Google Cloud Console：https://console.cloud.google.com/
- 启用 Cloud Translation API
- 创建 API 密钥（Credentials → Create Credentials → API Key）

---

### 3. 阿里云 CosyVoice（声音克隆）

**用途**: Processor 服务步骤 9 - 声音克隆

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `voice_cloning_vendor` | `aliyun_cosyvoice` | 固定值 |
| `aliyun_cosyvoice_app_key` | `_______________` | 请填写 |
| `aliyun_cosyvoice_access_key_id` | `_______________` | 请填写 |
| `aliyun_cosyvoice_access_key_secret` | `_______________` | 请填写 |
| `voice_cloning_output_dir` | `./data/voices` | 固定值 |

**获取方式**：
- 登录阿里云控制台：https://www.aliyun.com/
- 进入"智能语音交互"产品页面
- 创建 CosyVoice 应用并获取 App Key
- 创建 AccessKey（RAM 用户）

---

## 可选的 API 密钥（P1）

### 4. 文本润色（可选）

**用途**: Processor 服务步骤 6 - 文本润色

#### 选项 A：使用 Gemini

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `polishing_vendor` | `gemini` | 固定值 |
| `gemini_api_key` | `_______________` | 请填写 |
| `polishing_model_name` | `gemini-1.5-flash` | 推荐: gemini-1.5-flash |

**获取方式**：
- 登录 Google AI Studio：https://aistudio.google.com/
- 创建 API 密钥

#### 选项 B：使用 OpenAI

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `polishing_vendor` | `openai` | 固定值 |
| `openai_api_key` | `_______________` | 请填写 |
| `polishing_model_name` | `gpt-4o` | 推荐: gpt-4o |

**获取方式**：
- 登录 OpenAI Platform：https://platform.openai.com/
- 创建 API 密钥

---

### 5. 译文优化（可选）

**用途**: Processor 服务步骤 8 - 译文优化

#### 选项 A：使用 Gemini

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `optimization_vendor` | `gemini` | 固定值 |
| `gemini_api_key` | `_______________` | 请填写（与文本润色共用） |
| `optimization_model_name` | `gemini-1.5-flash` | 推荐: gemini-1.5-flash |

#### 选项 B：使用 OpenAI

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `optimization_vendor` | `openai` | 固定值 |
| `openai_api_key` | `_______________` | 请填写（与文本润色共用） |
| `optimization_model_name` | `gpt-4o` | 推荐: gpt-4o |

---

## 测试数据清单

### 测试视频文件

请准备以下测试视频文件：

| 文件名 | 时长 | 格式 | 分辨率 | 语言 | 文件大小 | 存放路径 | 备注 |
|--------|------|------|--------|------|----------|----------|------|
| test_video_01.mp4 | ___ 秒 | MP4 | ___p | 中文 | ___ MB | `d:\Go-Project\video-In-Chinese\data\videos\test\` | ___ |
| test_video_02.mp4 | ___ 秒 | MP4 | ___p | 英文 | ___ MB | `d:\Go-Project\video-In-Chinese\data\videos\test\` | ___ |

**文件要求**：
- 时长：10-30 秒（推荐 15 秒）
- 格式：MP4, MOV, AVI, MKV（推荐 MP4）
- 分辨率：720p 或 1080p
- 音频：必须包含音频轨道
- 语言：中文或英文（推荐中文）
- 文件大小：<50 MB
- 内容：包含清晰的人声对话，避免背景噪音过大

---

## 配置步骤

### 步骤 1：填写 API 密钥

请在上方表格中填写您的 API 密钥信息。

### 步骤 2：复制到配置脚本

打开 `scripts/setup_redis_config_template.ps1` 脚本，将上方表格中的值复制到脚本顶部的变量中：

```powershell
# 阿里云 ASR（语音识别）- 必需
$ALIYUN_ASR_APP_KEY = "YOUR_ALIYUN_ASR_APP_KEY"
$ALIYUN_ASR_ACCESS_KEY_ID = "YOUR_ALIYUN_ASR_ACCESS_KEY_ID"
$ALIYUN_ASR_ACCESS_KEY_SECRET = "YOUR_ALIYUN_ASR_ACCESS_KEY_SECRET"
$ASR_LANGUAGE_CODE = "zh-CN"
$ASR_REGION = "cn-shanghai"

# Google 翻译 - 必需
$GOOGLE_TRANSLATION_API_KEY = "YOUR_GOOGLE_TRANSLATION_API_KEY"

# 阿里云 CosyVoice（声音克隆）- 必需
$ALIYUN_COSYVOICE_APP_KEY = "YOUR_ALIYUN_COSYVOICE_APP_KEY"
$ALIYUN_COSYVOICE_ACCESS_KEY_ID = "YOUR_ALIYUN_COSYVOICE_ACCESS_KEY_ID"
$ALIYUN_COSYVOICE_ACCESS_KEY_SECRET = "YOUR_ALIYUN_COSYVOICE_ACCESS_KEY_SECRET"
$VOICE_CLONING_OUTPUT_DIR = "./data/voices"

# 文本润色（可选）- 使用 Gemini
$POLISHING_VENDOR = "gemini"
$GEMINI_API_KEY = "YOUR_GEMINI_API_KEY"
$POLISHING_MODEL_NAME = "gemini-1.5-flash"

# 译文优化（可选）- 使用 Gemini
$OPTIMIZATION_VENDOR = "gemini"
$OPTIMIZATION_MODEL_NAME = "gemini-1.5-flash"
```

### 步骤 3：执行配置脚本

```powershell
cd d:\Go-Project\video-In-Chinese
.\scripts\setup_redis_config_template.ps1
```

### 步骤 4：验证配置

```powershell
# 查看所有配置
docker exec redis-test redis-cli HGETALL app:settings

# 查看特定配置
docker exec redis-test redis-cli HGET app:settings asr_vendor
docker exec redis-test redis-cli HGET app:settings translation_vendor
docker exec redis-test redis-cli HGET app:settings voice_cloning_vendor
```

---

## 配置验证清单

请在完成配置后勾选以下清单：

- [ ] 阿里云 ASR API 密钥已配置
- [ ] Google 翻译 API 密钥已配置
- [ ] 阿里云 CosyVoice API 密钥已配置
- [ ] 文本润色 API 密钥已配置（可选）
- [ ] 译文优化 API 密钥已配置（可选）
- [ ] 测试视频文件已准备
- [ ] 配置脚本已执行
- [ ] Redis 配置已验证

---

## 常见问题

### Q1: 如何获取阿里云 AccessKey？

A: 
1. 登录阿里云控制台
2. 点击右上角头像 → AccessKey 管理
3. 创建 AccessKey（推荐使用 RAM 用户）
4. 保存 AccessKey ID 和 AccessKey Secret

### Q2: 如何获取 Google API 密钥？

A:
1. 登录 Google Cloud Console
2. 创建项目或选择现有项目
3. 启用 Cloud Translation API
4. 创建 API 密钥（Credentials → Create Credentials → API Key）
5. 限制 API 密钥的使用范围（推荐）

### Q3: 如何测试 API 密钥是否有效？

A:
- 阿里云 ASR：使用阿里云 SDK 测试
- Google 翻译：使用 curl 测试
  ```bash
  curl "https://translation.googleapis.com/language/translate/v2?key=YOUR_API_KEY&q=hello&target=zh-CN"
  ```
- Gemini：使用 curl 测试
  ```bash
  curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=YOUR_API_KEY" -H "Content-Type: application/json" -d '{"contents":[{"parts":[{"text":"Hello"}]}]}'
  ```

### Q4: 如果 API 密钥配置错误怎么办？

A:
1. 重新编辑 `scripts/setup_redis_config_template.ps1` 脚本
2. 修改错误的 API 密钥
3. 重新执行脚本（会覆盖旧配置）
4. 验证配置是否正确

---

**文档结束**

---

# 配置补充

1. 谷歌服务（OpenAI格式）

```yaml
ApiKey: "sk-aomanoh"
BaseURL: "https://balance.aomanoh.com/v1"
Model: "gemini-2.5-pro"
# 其余可调用多模态大模型：请参考 API 文档说明 `https://ai.google.dev/gemini-api/docs/models?hl=zh-cn`
# 注意，你提到的 1.5 模型已经被移除，请严格参考官方文档说明！
```

2. 阿里云服务（都可以选择使用）：

- paraformer-v2 模型实时语音识别API参考:[实时语音识别（Paraformer）-大模型服务平台百炼-阿里云](https://help.aliyun.com/zh/model-studio/paraformer-real-time-speech-recognition-api-reference/?spm=a2c4g.11186623.0.0.642d53d5TPmCap)

- fun-asr-realtime模型：[实时语音识别-通义千问-大模型服务平台百炼(Model Studio)-阿里云帮助中心](https://help.aliyun.com/zh/model-studio/qwen-real-time-speech-recognition?spm=a2c4g.11186623.help-menu-2400256.d_0_3_1.642d53d5TPmCap)

```
ApiKey: "sk-c36a30284fa44101a6e1f556e07c9574"
Model: # 请参考官方文档
```

3. 视频位置

- `notes\server\test\video\每日英语——励志英语片段 2025-11-06 02-09-13.mp4`
