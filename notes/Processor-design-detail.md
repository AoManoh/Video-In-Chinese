# Processor 服务实现文档（第三层）

**文档版本**: 1.0
**最后更新**: 2025-10-30
**关联第二层文档**: `notes/Processor-design.md`

---

## 1. 项目结构

```
server/mcp/processor/
├── main.go                         # gRPC 服务入口
├── go.mod
├── go.sum
├── internal/
│   ├── config/
│   │   └── config.go               # 配置加载
│   ├── logic/
│   │   └── processor_logic.go      # 主流程编排逻辑
│   ├── composer/                    # 音频合成包
│   │   ├── composer.go              # 核心接口定义
│   │   ├── concatenate.go           # 音频拼接
│   │   ├── align.go                 # 时长对齐
│   │   ├── merge.go                 # 音频合并
│   │   └── composer_test.go         # 单元测试
│   ├── mediautil/                   # 媒体工具包
│   │   ├── extract.go               # 提取音频
│   │   ├── merge.go                 # 合并音视频
│   │   └── mediautil_test.go        # 单元测试
│   └── storage/
│       └── redis.go                 # Redis 操作
├── proto/
│   └── processor.proto              # gRPC 接口定义
└── Dockerfile
```

---

## 2. 核心代码实现

### 2.1 main.go（gRPC 服务入口）

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"processor/internal/config"
	"processor/internal/logic"
	pb "processor/proto"
)

type server struct {
	pb.UnimplementedProcessorServer
	logic *logic.ProcessorLogic
}

func (s *server) ProcessVideo(ctx context.Context, req *pb.ProcessVideoRequest) (*pb.ProcessVideoResponse, error) {
	err := s.logic.ProcessVideo(ctx, req.TaskId, req.OriginalFileKey)
	if err != nil {
		return &pb.ProcessVideoResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}
	return &pb.ProcessVideoResponse{
		Success:      true,
		ErrorMessage: "",
	}, nil
}

func main() {
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化业务逻辑
	processorLogic, err := logic.NewProcessorLogic(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize processor logic: %v", err)
	}

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProcessorServer(s, &server{logic: processorLogic})
	reflection.Register(s)

	log.Printf("Processor service listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
```

---

### 2.2 internal/config/config.go（配置加载）

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Processor 配置
	MaxConcurrency int
	StoragePath    string

	// 依赖服务地址
	AIAdaptorAddr       string
	AudioSeparatorAddr  string

	// Redis 配置
	RedisHost string
	RedisPort int

	// 应用配置
	AudioSeparationEnabled bool
	PolishingEnabled       bool
	OptimizationEnabled    bool
}

func LoadConfig() (*Config, error) {
	maxConcurrency, _ := strconv.Atoi(getEnv("PROCESSOR_MAX_CONCURRENCY", "1"))
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))

	return &Config{
		MaxConcurrency:     maxConcurrency,
		StoragePath:        getEnv("LOCAL_STORAGE_PATH", "./data/videos"),
		AIAdaptorAddr:      getEnv("AI_ADAPTOR_GRPC_ADDR", "ai-adaptor:50053"),
		AudioSeparatorAddr: getEnv("AUDIO_SEPARATOR_GRPC_ADDR", "audio-separator:50052"),
		RedisHost:          getEnv("REDIS_HOST", "redis"),
		RedisPort:          redisPort,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

---

### 2.3 internal/logic/processor_logic.go（主流程编排）

```go
package logic

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"processor/internal/composer"
	"processor/internal/config"
	"processor/internal/mediautil"
	"processor/internal/storage"
)

type ProcessorLogic struct {
	cfg              *config.Config
	redis            *storage.RedisClient
	composer         composer.Composer
	workerSemaphore  chan struct{}
}

func NewProcessorLogic(cfg *config.Config) (*ProcessorLogic, error) {
	// 初始化 Redis 客户端
	redisClient, err := storage.NewRedisClient(cfg.RedisHost, cfg.RedisPort)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// 初始化 Composer
	composerImpl := composer.NewComposer()

	// 初始化并发控制信号量
	workerSemaphore := make(chan struct{}, cfg.MaxConcurrency)

	return &ProcessorLogic{
		cfg:             cfg,
		redis:           redisClient,
		composer:        composerImpl,
		workerSemaphore: workerSemaphore,
	}, nil
}

func (p *ProcessorLogic) ProcessVideo(ctx context.Context, taskID, originalFileKey string) error {
	// 1-2. 并发控制
	select {
	case p.workerSemaphore <- struct{}{}:
		defer func() { <-p.workerSemaphore }()
	default:
		return fmt.Errorf("resource exhausted: max concurrency reached")
	}

	// 3. 状态更新
	if err := p.redis.UpdateTaskStatus(ctx, taskID, "PROCESSING", "", ""); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// 异常处理：确保失败时更新状态
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("panic: %v", r)
			p.redis.UpdateTaskStatus(ctx, taskID, "FAILED", "", errMsg)
			log.Printf("Task %s failed with panic: %v", taskID, r)
		}
	}()

	// 4-5. 文件准备 + 音频提取
	videoPath := filepath.Join(p.cfg.StoragePath, taskID, "original.mp4")
	audioPath := filepath.Join(p.cfg.StoragePath, taskID, "audio.wav")
	if err := mediautil.ExtractAudio(videoPath, audioPath); err != nil {
		p.redis.UpdateTaskStatus(ctx, taskID, "FAILED", "", err.Error())
		return fmt.Errorf("failed to extract audio: %w", err)
	}

	// 6. 音频分离（可选）
	var vocalsPath, bgmPath string
	if p.cfg.AudioSeparationEnabled {
		// TODO: 调用 audio-separator 服务
		vocalsPath = filepath.Join(p.cfg.StoragePath, taskID, "vocals.wav")
		bgmPath = filepath.Join(p.cfg.StoragePath, taskID, "bgm.wav")
	} else {
		vocalsPath = audioPath
		bgmPath = ""
	}

	// 7-11. AI 处理（调用 ai-adaptor）
	// TODO: 实现 AI 服务调用
	// asrResult := p.callAIAdaptor.ASR(vocalsPath)
	// polishedText := p.callAIAdaptor.Polish(asrResult.Text)
	// translatedText := p.callAIAdaptor.Translate(polishedText)
	// optimizedText := p.callAIAdaptor.Optimize(translatedText)
	// clonedAudios := p.callAIAdaptor.CloneVoice(asrResult.Speakers, optimizedText)

	// 12-14. 音频合成（调用内部 composer 包）
	clonedAudios := []composer.AudioSegment{} // TODO: 从 AI 服务获取
	originalDuration := 10.0                   // TODO: 从视频获取
	finalAudioPath := filepath.Join(p.cfg.StoragePath, taskID, "final_audio.wav")

	composeReq := composer.ComposeRequest{
		ClonedAudios:     clonedAudios,
		BackgroundAudio:  bgmPath,
		OriginalDuration: originalDuration,
		OutputPath:       finalAudioPath,
	}
	composeResp, err := p.composer.Compose(ctx, composeReq)
	if err != nil {
		p.redis.UpdateTaskStatus(ctx, taskID, "FAILED", "", err.Error())
		return fmt.Errorf("failed to compose audio: %w", err)
	}

	// 15-17. 视频合成 + 保存
	resultVideoPath := filepath.Join(p.cfg.StoragePath, taskID, "result.mp4")
	if err := mediautil.MergeAudioVideo(videoPath, composeResp.FinalAudioPath, resultVideoPath); err != nil {
		p.redis.UpdateTaskStatus(ctx, taskID, "FAILED", "", err.Error())
		return fmt.Errorf("failed to merge audio and video: %w", err)
	}

	// 更新状态为 COMPLETED
	if err := p.redis.UpdateTaskStatus(ctx, taskID, "COMPLETED", resultVideoPath, ""); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Printf("Task %s completed successfully", taskID)
	return nil
}
```

---

### 2.4 internal/composer/composer.go（音频合成接口）

```go
package composer

import (
	"context"
	"time"
)

// AudioSegment 音频片段
type AudioSegment struct {
	SpeakerID string  // 说话人 ID
	AudioPath string  // 音频文件路径
	StartTime float64 // 开始时间（秒）
	EndTime   float64 // 结束时间（秒）
}

// ComposeRequest 音频合成请求
type ComposeRequest struct {
	ClonedAudios     []AudioSegment // 克隆后的音频片段
	BackgroundAudio  string         // 背景音路径（可选）
	OriginalDuration float64        // 原始视频时长（秒）
	OutputPath       string         // 输出路径
}

// ComposeResponse 音频合成响应
type ComposeResponse struct {
	FinalAudioPath   string // 最终音频路径
	ProcessingTimeMs int64  // 处理耗时（毫秒）
}

// Composer 音频合成接口
type Composer interface {
	Compose(ctx context.Context, req ComposeRequest) (ComposeResponse, error)
}

// composerImpl 音频合成实现
type composerImpl struct{}

func NewComposer() Composer {
	return &composerImpl{}
}

func (c *composerImpl) Compose(ctx context.Context, req ComposeRequest) (ComposeResponse, error) {
	startTime := time.Now()

	// 1. 音频拼接
	concatenatedPath, err := concatenateAudios(req.ClonedAudios, req.OutputPath+".concat.wav")
	if err != nil {
		return ComposeResponse{}, err
	}

	// 2. 时长对齐
	alignedPath, err := alignDuration(concatenatedPath, req.OriginalDuration, req.OutputPath+".aligned.wav")
	if err != nil {
		return ComposeResponse{}, err
	}

	// 3. 音频合并（人声 + 背景音）
	var finalPath string
	if req.BackgroundAudio != "" {
		finalPath, err = mergeAudios(alignedPath, req.BackgroundAudio, req.OutputPath)
		if err != nil {
			return ComposeResponse{}, err
		}
	} else {
		finalPath = alignedPath
	}

	processingTime := time.Since(startTime).Milliseconds()
	return ComposeResponse{
		FinalAudioPath:   finalPath,
		ProcessingTimeMs: processingTime,
	}, nil
}
```

---

## 3. 后续实现任务

### 3.1 待实现的功能

1. **internal/composer/concatenate.go**: 音频拼接实现（使用 ffmpeg）
2. **internal/composer/align.go**: 时长对齐实现（静音填充 + 速度调整）
3. **internal/composer/merge.go**: 音频合并实现（人声 + 背景音）
4. **internal/mediautil/extract.go**: 音频提取实现（使用 ffmpeg）
5. **internal/mediautil/merge.go**: 音视频合并实现（使用 ffmpeg）
6. **internal/storage/redis.go**: Redis 操作实现
7. **AI 服务调用**: 集成 ai-adaptor gRPC 客户端
8. **单元测试**: composer 和 mediautil 的单元测试

### 3.2 依赖库

```go
// go.mod
module processor

go 1.21

require (
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	github.com/go-redis/redis/v8 v8.11.5
)
```

---

## 4. Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

# 安装 ffmpeg
RUN apk add --no-cache ffmpeg

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o processor main.go

FROM alpine:latest
RUN apk add --no-cache ffmpeg
WORKDIR /root/
COPY --from=builder /app/processor .

EXPOSE 50051
CMD ["./processor"]
```

---

## 5. 单元测试示例

### 5.1 internal/composer/composer_test.go

```go
package composer

import (
	"context"
	"testing"
)

func TestCompose(t *testing.T) {
	composer := NewComposer()

	req := ComposeRequest{
		ClonedAudios: []AudioSegment{
			{SpeakerID: "speaker1", AudioPath: "test1.wav", StartTime: 0, EndTime: 5},
			{SpeakerID: "speaker2", AudioPath: "test2.wav", StartTime: 5, EndTime: 10},
		},
		BackgroundAudio:  "",
		OriginalDuration: 10.0,
		OutputPath:       "/tmp/test_output.wav",
	}

	resp, err := composer.Compose(context.Background(), req)
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	if resp.FinalAudioPath == "" {
		t.Error("Expected non-empty final audio path")
	}
}
```

---
