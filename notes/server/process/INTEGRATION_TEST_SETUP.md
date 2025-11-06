# Processor 服务集成测试环境搭建指南

**文档版本**: v1.0  
**创建日期**: 2025-11-06  
**适用服务**: Processor 服务  
**预计完成时间**: 35-45 分钟

---

## 目录

1. [前置条件检查](#1-前置条件检查)
2. [ffmpeg 安装（Windows 11）](#2-ffmpeg-安装windows-11)
3. [Redis 服务启动](#3-redis-服务启动)
4. [测试数据准备](#4-测试数据准备)
5. [API 密钥配置](#5-api-密钥配置)
6. [环境验证](#6-环境验证)
7. [常见问题](#7-常见问题)

---

## 1. 前置条件检查

### 1.1 必须满足的条件（P0）

- [ ] Docker Desktop 已启动 ✅（已确认）
- [ ] Go 1.21+ 已安装 ✅（已确认，当前版本 1.25rc2）
- [ ] ffmpeg 已安装 ⚠️（待安装）
- [ ] Redis 服务已启动 ⚠️（待启动）

### 1.2 建议满足的条件（P1）

- [ ] 测试视频文件已准备 ⚠️（待准备）
- [ ] API 密钥已配置 ⚠️（待配置）

---

## 2. ffmpeg 安装（Windows 11）

### 2.1 下载 ffmpeg

1. 访问 ffmpeg 官方下载页面：
   - 官方网站：https://ffmpeg.org/download.html
   - Windows 构建版本（推荐）：https://www.gyan.dev/ffmpeg/builds/

2. 下载最新的 **ffmpeg-release-essentials.zip**：
   - 文件大小：约 80-100 MB
   - 包含：ffmpeg.exe, ffprobe.exe, ffplay.exe

### 2.2 解压和安装

1. 解压下载的 zip 文件到指定目录：
   ```
   推荐路径：C:\ffmpeg
   ```

2. 解压后的目录结构：
   ```
   C:\ffmpeg\
   ├── bin\
   │   ├── ffmpeg.exe
   │   ├── ffprobe.exe
   │   └── ffplay.exe
   ├── doc\
   └── presets\
   ```

### 2.3 添加到系统 PATH

1. 打开"系统环境变量"设置：
   - 按 `Win + X`，选择"系统"
   - 点击"高级系统设置"
   - 点击"环境变量"

2. 编辑 PATH 环境变量：
   - 在"系统变量"中找到 `Path`
   - 点击"编辑"
   - 点击"新建"
   - 添加：`C:\ffmpeg\bin`
   - 点击"确定"保存

3. 重启 PowerShell 或命令提示符

### 2.4 验证安装

打开新的 PowerShell 窗口，执行：

```powershell
ffmpeg -version
```

**期望输出**：
```
ffmpeg version N-XXXXX-gXXXXXXXXXX-essentials_build-www.gyan.dev
Copyright (c) 2000-2024 the FFmpeg developers
built with gcc X.X.X (GCC)
configuration: --enable-gpl --enable-version3 ...
```

如果看到版本信息，说明安装成功 ✅

### 2.5 验证 ffprobe

```powershell
ffprobe -version
```

**期望输出**：
```
ffprobe version N-XXXXX-gXXXXXXXXXX-essentials_build-www.gyan.dev
```

---

## 3. Redis 服务启动

### 3.1 使用 Docker 启动 Redis

在项目根目录执行：

```powershell
cd d:\Go-Project\video-In-Chinese

# 启动 Redis 容器
docker run -d `
  --name redis-test `
  -p 6379:6379 `
  --restart unless-stopped `
  redis:latest
```

**参数说明**：
- `-d`: 后台运行
- `--name redis-test`: 容器名称
- `-p 6379:6379`: 端口映射（主机:容器）
- `--restart unless-stopped`: 自动重启策略
- `redis:latest`: 使用最新版本的 Redis 镜像

### 3.2 验证 Redis 服务

```powershell
# 方法 1：使用 docker exec
docker exec -it redis-test redis-cli ping

# 期望输出：PONG
```

```powershell
# 方法 2：使用 redis-cli（如果已安装）
redis-cli ping

# 期望输出：PONG
```

### 3.3 查看 Redis 日志

```powershell
docker logs redis-test
```

### 3.4 停止和重启 Redis

```powershell
# 停止 Redis
docker stop redis-test

# 启动 Redis
docker start redis-test

# 重启 Redis
docker restart redis-test

# 删除 Redis 容器（慎用）
docker rm -f redis-test
```

---

## 4. 测试数据准备

### 4.1 测试视频文件要求

**文件规格**：
- **时长**: 10-30 秒（推荐 15 秒）
- **格式**: MP4, MOV, AVI, MKV（推荐 MP4）
- **分辨率**: 720p 或 1080p
- **音频**: 必须包含音频轨道
- **语言**: 中文或英文（推荐中文）
- **文件大小**: <50 MB

**内容要求**：
- 包含清晰的人声对话
- 避免背景噪音过大
- 避免多人同时说话
- 推荐使用新闻、访谈、教学视频片段

### 4.2 测试视频文件存放位置

请将测试视频文件放置在以下目录：

```
d:\Go-Project\video-In-Chinese\data\videos\test\
```

**文件命名建议**：
```
test_video_01.mp4
test_video_02.mp4
```

### 4.3 创建测试目录

```powershell
# 创建测试目录
New-Item -ItemType Directory -Path "d:\Go-Project\video-In-Chinese\data\videos\test" -Force
```

### 4.4 测试视频文件清单

请在下方填写您准备的测试视频文件信息：

| 文件名 | 时长 | 格式 | 分辨率 | 语言 | 文件大小 | 备注 |
|--------|------|------|--------|------|----------|------|
| test_video_01.mp4 | ___ 秒 | MP4 | ___p | 中文 | ___ MB | ___ |
| test_video_02.mp4 | ___ 秒 | MP4 | ___p | 英文 | ___ MB | ___ |

---

## 5. API 密钥配置

### 5.1 需要配置的 API 密钥

根据 Processor 服务的 18 步处理流程，需要配置以下 API 密钥：

#### 5.1.1 必须配置（P0）

1. **阿里云 ASR（语音识别）**
   - 用途：步骤 4 - ASR（语音识别）
   - 必需字段：
     - `asr_vendor`: `aliyun`
     - `aliyun_asr_app_key`: `_______________`（请填写）
     - `aliyun_asr_access_key_id`: `_______________`（请填写）
     - `aliyun_asr_access_key_secret`: `_______________`（请填写）
     - `asr_language_code`: `zh-CN`（中文）或 `en-US`（英文）
     - `asr_region`: `cn-shanghai`（推荐）

2. **Google 翻译（文本翻译）**
   - 用途：步骤 7 - 翻译
   - 必需字段：
     - `translation_vendor`: `google`
     - `google_translation_api_key`: `_______________`（请填写）

3. **阿里云 CosyVoice（声音克隆）**
   - 用途：步骤 9 - 声音克隆
   - 必需字段：
     - `voice_cloning_vendor`: `aliyun_cosyvoice`
     - `aliyun_cosyvoice_app_key`: `_______________`（请填写）
     - `aliyun_cosyvoice_access_key_id`: `_______________`（请填写）
     - `aliyun_cosyvoice_access_key_secret`: `_______________`（请填写）
     - `voice_cloning_output_dir`: `./data/voices`

#### 5.1.2 可选配置（P1）

4. **文本润色（可选）**
   - 用途：步骤 6 - 文本润色
   - 必需字段：
     - `polishing_vendor`: `gemini` 或 `openai`
     - `gemini_api_key`: `_______________`（如果使用 Gemini）
     - `openai_api_key`: `_______________`（如果使用 OpenAI）
     - `polishing_model_name`: `gemini-1.5-flash` 或 `gpt-4o`

5. **译文优化（可选）**
   - 用途：步骤 8 - 译文优化
   - 必需字段：
     - `optimization_vendor`: `gemini` 或 `openai`
     - `gemini_api_key`: `_______________`（如果使用 Gemini）
     - `openai_api_key`: `_______________`（如果使用 OpenAI）
     - `optimization_model_name`: `gemini-1.5-flash` 或 `gpt-4o`

### 5.2 配置方式：使用 Redis

#### 5.2.1 配置脚本模板

创建配置脚本 `scripts/setup_redis_config.ps1`：

```powershell
# Redis 配置脚本
# 用途：将 API 密钥配置写入 Redis

# 连接 Redis
$redisContainer = "redis-test"

# 配置 ASR（阿里云）
docker exec -it $redisContainer redis-cli HSET app:settings `
  asr_vendor "aliyun" `
  aliyun_asr_app_key "YOUR_ALIYUN_ASR_APP_KEY" `
  aliyun_asr_access_key_id "YOUR_ALIYUN_ASR_ACCESS_KEY_ID" `
  aliyun_asr_access_key_secret "YOUR_ALIYUN_ASR_ACCESS_KEY_SECRET" `
  asr_language_code "zh-CN" `
  asr_region "cn-shanghai"

# 配置翻译（Google）
docker exec -it $redisContainer redis-cli HSET app:settings `
  translation_vendor "google" `
  google_translation_api_key "YOUR_GOOGLE_TRANSLATION_API_KEY"

# 配置声音克隆（阿里云 CosyVoice）
docker exec -it $redisContainer redis-cli HSET app:settings `
  voice_cloning_vendor "aliyun_cosyvoice" `
  aliyun_cosyvoice_app_key "YOUR_ALIYUN_COSYVOICE_APP_KEY" `
  aliyun_cosyvoice_access_key_id "YOUR_ALIYUN_COSYVOICE_ACCESS_KEY_ID" `
  aliyun_cosyvoice_access_key_secret "YOUR_ALIYUN_COSYVOICE_ACCESS_KEY_SECRET" `
  voice_cloning_output_dir "./data/voices"

# 配置文本润色（可选，使用 Gemini）
docker exec -it $redisContainer redis-cli HSET app:settings `
  polishing_vendor "gemini" `
  gemini_api_key "YOUR_GEMINI_API_KEY" `
  polishing_model_name "gemini-1.5-flash"

# 配置译文优化（可选，使用 Gemini）
docker exec -it $redisContainer redis-cli HSET app:settings `
  optimization_vendor "gemini" `
  optimization_model_name "gemini-1.5-flash"

Write-Host "Redis configuration completed successfully!" -ForegroundColor Green
```

#### 5.2.2 执行配置脚本

1. 将上述脚本保存为 `scripts/setup_redis_config.ps1`
2. 替换所有 `YOUR_*` 占位符为实际的 API 密钥
3. 执行脚本：

```powershell
cd d:\Go-Project\video-In-Chinese
.\scripts\setup_redis_config.ps1
```

#### 5.2.3 验证配置

```powershell
# 查看所有配置
docker exec -it redis-test redis-cli HGETALL app:settings

# 查看特定配置
docker exec -it redis-test redis-cli HGET app:settings asr_vendor
```

---

## 6. 环境验证

### 6.1 验证清单

执行以下命令验证环境是否就绪：

```powershell
# 1. 验证 ffmpeg
ffmpeg -version
# 期望：显示版本信息

# 2. 验证 ffprobe
ffprobe -version
# 期望：显示版本信息

# 3. 验证 Redis
docker exec -it redis-test redis-cli ping
# 期望：PONG

# 4. 验证 Redis 配置
docker exec -it redis-test redis-cli HGETALL app:settings
# 期望：显示所有配置项

# 5. 验证测试数据目录
Test-Path "d:\Go-Project\video-In-Chinese\data\videos\test"
# 期望：True

# 6. 验证测试视频文件
Get-ChildItem "d:\Go-Project\video-In-Chinese\data\videos\test" -Filter *.mp4
# 期望：显示测试视频文件列表
```

### 6.2 环境验证脚本

创建验证脚本 `scripts/verify_integration_test_env.ps1`：

```powershell
# 环境验证脚本
Write-Host "=== Processor 服务集成测试环境验证 ===" -ForegroundColor Cyan

# 1. 验证 ffmpeg
Write-Host "`n[1/6] 验证 ffmpeg..." -ForegroundColor Yellow
try {
    $ffmpegVersion = ffmpeg -version 2>&1 | Select-Object -First 1
    Write-Host "  ✅ ffmpeg 已安装: $ffmpegVersion" -ForegroundColor Green
} catch {
    Write-Host "  ❌ ffmpeg 未安装或未添加到 PATH" -ForegroundColor Red
    exit 1
}

# 2. 验证 ffprobe
Write-Host "`n[2/6] 验证 ffprobe..." -ForegroundColor Yellow
try {
    $ffprobeVersion = ffprobe -version 2>&1 | Select-Object -First 1
    Write-Host "  ✅ ffprobe 已安装: $ffprobeVersion" -ForegroundColor Green
} catch {
    Write-Host "  ❌ ffprobe 未安装或未添加到 PATH" -ForegroundColor Red
    exit 1
}

# 3. 验证 Redis
Write-Host "`n[3/6] 验证 Redis..." -ForegroundColor Yellow
try {
    $redisPing = docker exec redis-test redis-cli ping 2>&1
    if ($redisPing -eq "PONG") {
        Write-Host "  ✅ Redis 服务正常运行" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Redis 服务未响应" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "  ❌ Redis 容器未启动" -ForegroundColor Red
    exit 1
}

# 4. 验证 Redis 配置
Write-Host "`n[4/6] 验证 Redis 配置..." -ForegroundColor Yellow
try {
    $redisConfig = docker exec redis-test redis-cli HGETALL app:settings 2>&1
    if ($redisConfig.Count -gt 0) {
        Write-Host "  ✅ Redis 配置已设置（$($redisConfig.Count / 2) 个配置项）" -ForegroundColor Green
    } else {
        Write-Host "  ⚠️  Redis 配置为空，请执行配置脚本" -ForegroundColor Yellow
    }
} catch {
    Write-Host "  ❌ 无法读取 Redis 配置" -ForegroundColor Red
}

# 5. 验证测试数据目录
Write-Host "`n[5/6] 验证测试数据目录..." -ForegroundColor Yellow
$testDataDir = "d:\Go-Project\video-In-Chinese\data\videos\test"
if (Test-Path $testDataDir) {
    Write-Host "  ✅ 测试数据目录存在: $testDataDir" -ForegroundColor Green
} else {
    Write-Host "  ⚠️  测试数据目录不存在，正在创建..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $testDataDir -Force | Out-Null
    Write-Host "  ✅ 测试数据目录已创建" -ForegroundColor Green
}

# 6. 验证测试视频文件
Write-Host "`n[6/6] 验证测试视频文件..." -ForegroundColor Yellow
$testVideos = Get-ChildItem $testDataDir -Filter *.mp4 -ErrorAction SilentlyContinue
if ($testVideos.Count -gt 0) {
    Write-Host "  ✅ 找到 $($testVideos.Count) 个测试视频文件" -ForegroundColor Green
    foreach ($video in $testVideos) {
        Write-Host "    - $($video.Name) ($([math]::Round($video.Length / 1MB, 2)) MB)" -ForegroundColor Gray
    }
} else {
    Write-Host "  ⚠️  未找到测试视频文件，请准备测试数据" -ForegroundColor Yellow
}

Write-Host "`n=== 环境验证完成 ===" -ForegroundColor Cyan
```

执行验证脚本：

```powershell
cd d:\Go-Project\video-In-Chinese
.\scripts\verify_integration_test_env.ps1
```

---

## 7. 常见问题

### 7.1 ffmpeg 相关问题

**Q1: ffmpeg 命令找不到？**

A: 请确认：
1. ffmpeg 已正确解压到 `C:\ffmpeg\bin`
2. 已将 `C:\ffmpeg\bin` 添加到系统 PATH
3. 已重启 PowerShell 或命令提示符

**Q2: ffmpeg 版本过旧？**

A: 请下载最新版本的 ffmpeg（推荐 4.0 或更高版本）

### 7.2 Redis 相关问题

**Q1: Redis 容器启动失败？**

A: 请检查：
1. Docker Desktop 是否已启动
2. 端口 6379 是否被占用：`netstat -ano | findstr :6379`
3. 查看 Docker 日志：`docker logs redis-test`

**Q2: Redis 配置丢失？**

A: Redis 容器重启后配置会丢失（未启用持久化），请重新执行配置脚本。

**解决方案**：启用 Redis 持久化：
```powershell
docker run -d `
  --name redis-test `
  -p 6379:6379 `
  -v redis-data:/data `
  --restart unless-stopped `
  redis:latest redis-server --appendonly yes
```

### 7.3 测试数据相关问题

**Q1: 测试视频文件格式不支持？**

A: 请确认：
1. 文件格式为 MP4, MOV, AVI, MKV
2. 文件包含音频轨道
3. 使用 ffprobe 检查文件信息：
   ```powershell
   ffprobe -i test_video_01.mp4
   ```

**Q2: 测试视频文件过大？**

A: 请使用 ffmpeg 压缩视频：
```powershell
ffmpeg -i input.mp4 -vcodec h264 -acodec aac -b:v 1M -b:a 128k output.mp4
```

---

## 附录：完整的环境搭建脚本

创建一键搭建脚本 `scripts/setup_integration_test_env.ps1`：

```powershell
# 一键搭建集成测试环境
Write-Host "=== Processor 服务集成测试环境一键搭建 ===" -ForegroundColor Cyan

# 1. 创建测试数据目录
Write-Host "`n[1/3] 创建测试数据目录..." -ForegroundColor Yellow
$testDataDir = "d:\Go-Project\video-In-Chinese\data\videos\test"
New-Item -ItemType Directory -Path $testDataDir -Force | Out-Null
Write-Host "  ✅ 测试数据目录已创建: $testDataDir" -ForegroundColor Green

# 2. 启动 Redis 容器
Write-Host "`n[2/3] 启动 Redis 容器..." -ForegroundColor Yellow
try {
    docker run -d `
      --name redis-test `
      -p 6379:6379 `
      -v redis-data:/data `
      --restart unless-stopped `
      redis:latest redis-server --appendonly yes
    Write-Host "  ✅ Redis 容器已启动" -ForegroundColor Green
} catch {
    Write-Host "  ⚠️  Redis 容器可能已存在，尝试启动..." -ForegroundColor Yellow
    docker start redis-test
}

# 3. 等待 Redis 就绪
Write-Host "`n[3/3] 等待 Redis 就绪..." -ForegroundColor Yellow
Start-Sleep -Seconds 3
$redisPing = docker exec redis-test redis-cli ping 2>&1
if ($redisPing -eq "PONG") {
    Write-Host "  ✅ Redis 服务已就绪" -ForegroundColor Green
} else {
    Write-Host "  ❌ Redis 服务未就绪" -ForegroundColor Red
    exit 1
}

Write-Host "`n=== 环境搭建完成 ===" -ForegroundColor Cyan
Write-Host "`n下一步：" -ForegroundColor Yellow
Write-Host "  1. 安装 ffmpeg（参考文档第 2 节）" -ForegroundColor Gray
Write-Host "  2. 准备测试视频文件（参考文档第 4 节）" -ForegroundColor Gray
Write-Host "  3. 配置 API 密钥（参考文档第 5 节）" -ForegroundColor Gray
Write-Host "  4. 执行环境验证脚本：.\scripts\verify_integration_test_env.ps1" -ForegroundColor Gray
```

---

**文档结束**

