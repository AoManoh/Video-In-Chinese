# AIAdaptor 服务实现文档（第三层）

**文档版本**: 1.0  
**最后更新**: 2025-10-30  
**关联第二层文档**: `notes/AIAdaptor-design.md`

---

## 1. 项目结构

```
server/mcp/ai-adaptor/
├── main.go                         # gRPC 服务入口
├── go.mod
├── go.sum
├── internal/
│   ├── config/
│   │   └── config.go               # 配置加载
│   ├── logic/
│   │   └── ai_adaptor_logic.go     # 主业务逻辑
│   ├── adapters/
│   │   ├── interface.go            # 适配器接口定义
│   │   ├── aliyun_cosyvoice.go     # 阿里云 CosyVoice 适配器
│   │   ├── aliyun_asr.go           # 阿里云 ASR 适配器
│   │   ├── deepl_translation.go    # DeepL 翻译适配器
│   │   └── openai_llm.go           # OpenAI LLM 适配器
│   ├── voice_manager/
│   │   └── voice_manager.go        # 音色管理（缓存、注册、轮询）
│   └── storage/
│       └── redis.go                # Redis 操作
├── proto/
│   └── aiadaptor.proto             # gRPC 接口定义
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

	"ai-adaptor/internal/config"
	"ai-adaptor/internal/logic"
	pb "ai-adaptor/proto"
)

type server struct {
	pb.UnimplementedAIAdaptorServer
	logic *logic.AIAdaptorLogic
}

func (s *server) ASR(ctx context.Context, req *pb.ASRRequest) (*pb.ASRResponse, error) {
	return s.logic.ASR(ctx, req)
}

func (s *server) Polish(ctx context.Context, req *pb.PolishRequest) (*pb.PolishResponse, error) {
	return s.logic.Polish(ctx, req)
}

func (s *server) Translate(ctx context.Context, req *pb.TranslateRequest) (*pb.TranslateResponse, error) {
	return s.logic.Translate(ctx, req)
}

func (s *server) Optimize(ctx context.Context, req *pb.OptimizeRequest) (*pb.OptimizeResponse, error) {
	return s.logic.Optimize(ctx, req)
}

func (s *server) CloneVoice(ctx context.Context, req *pb.CloneVoiceRequest) (*pb.CloneVoiceResponse, error) {
	return s.logic.CloneVoice(ctx, req)
}

func main() {
	port := flag.Int("port", 50053, "The server port")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化业务逻辑
	aiAdaptorLogic, err := logic.NewAIAdaptorLogic(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize AI adaptor logic: %v", err)
	}

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAIAdaptorServer(s, &server{logic: aiAdaptorLogic})
	reflection.Register(s)

	log.Printf("AIAdaptor service listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
```

---

### 2.2 internal/adapters/interface.go（适配器接口定义）

```go
package adapters

import (
	"context"
	pb "ai-adaptor/proto"
)

// ASRAdapter ASR 适配器接口
type ASRAdapter interface {
	ASR(ctx context.Context, audioPath string) (*pb.ASRResponse, error)
}

// TranslationAdapter 翻译适配器接口
type TranslationAdapter interface {
	Translate(ctx context.Context, text, sourceLang, targetLang, videoType string) (string, error)
}

// LLMAdapter LLM 适配器接口
type LLMAdapter interface {
	Polish(ctx context.Context, text, videoType, customPrompt string) (string, error)
	Optimize(ctx context.Context, text string) (string, error)
}

// VoiceCloningAdapter 声音克隆适配器接口
type VoiceCloningAdapter interface {
	CloneVoice(ctx context.Context, speakerID, text, referenceAudio string) (string, error)
}
```

---

### 2.3 internal/adapters/aliyun_cosyvoice.go（阿里云 CosyVoice 适配器）

```go
package adapters

import (
	"context"
	"fmt"
	"time"

	"ai-adaptor/internal/voice_manager"
)

type AliyunCosyVoiceAdapter struct {
	apiKey       string
	endpoint     string
	voiceManager *voice_manager.VoiceManager
}

func NewAliyunCosyVoiceAdapter(apiKey, endpoint string, vm *voice_manager.VoiceManager) *AliyunCosyVoiceAdapter {
	return &AliyunCosyVoiceAdapter{
		apiKey:       apiKey,
		endpoint:     endpoint,
		voiceManager: vm,
	}
}

func (a *AliyunCosyVoiceAdapter) CloneVoice(ctx context.Context, speakerID, text, referenceAudio string) (string, error) {
	// 1. 检查缓存
	voiceID, err := a.voiceManager.GetVoiceID(ctx, speakerID)
	if err != nil || voiceID == "" {
		// 2. 音色未缓存，需要注册
		voiceID, err = a.registerVoice(ctx, speakerID, referenceAudio)
		if err != nil {
			return "", fmt.Errorf("failed to register voice: %w", err)
		}
	}

	// 3. 使用 voice_id 合成音频
	audioPath, err := a.synthesizeAudio(ctx, voiceID, text)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize audio: %w", err)
	}

	return audioPath, nil
}

func (a *AliyunCosyVoiceAdapter) registerVoice(ctx context.Context, speakerID, referenceAudio string) (string, error) {
	// 1. 上传参考音频到临时 OSS
	publicURL, err := a.uploadToOSS(ctx, referenceAudio)
	if err != nil {
		return "", fmt.Errorf("failed to upload to OSS: %w", err)
	}

	// 2. 调用阿里云 API 创建音色
	voiceID, err := a.createVoice(ctx, publicURL)
	if err != nil {
		return "", fmt.Errorf("failed to create voice: %w", err)
	}

	// 3. 轮询音色状态，直到 OK（最多等待 60 秒）
	for i := 0; i < 60; i++ {
		status, err := a.queryVoiceStatus(ctx, voiceID)
		if err != nil {
			return "", fmt.Errorf("failed to query voice status: %w", err)
		}
		if status == "OK" {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// 4. 缓存 voice_id
	if err := a.voiceManager.CacheVoiceID(ctx, speakerID, voiceID, referenceAudio); err != nil {
		return "", fmt.Errorf("failed to cache voice ID: %w", err)
	}

	return voiceID, nil
}

func (a *AliyunCosyVoiceAdapter) uploadToOSS(ctx context.Context, referenceAudio string) (string, error) {
	// TODO: 实现上传到阿里云 OSS
	return "https://oss.example.com/temp/" + referenceAudio, nil
}

func (a *AliyunCosyVoiceAdapter) createVoice(ctx context.Context, publicURL string) (string, error) {
	// TODO: 调用阿里云 CosyVoice API 创建音色
	return "voice_id_12345", nil
}

func (a *AliyunCosyVoiceAdapter) queryVoiceStatus(ctx context.Context, voiceID string) (string, error) {
	// TODO: 调用阿里云 CosyVoice API 查询音色状态
	return "OK", nil
}

func (a *AliyunCosyVoiceAdapter) synthesizeAudio(ctx context.Context, voiceID, text string) (string, error) {
	// TODO: 调用阿里云 CosyVoice API 合成音频
	return "/tmp/synthesized_audio.wav", nil
}
```

---

### 2.4 internal/voice_manager/voice_manager.go（音色管理）

```go
package voice_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type VoiceInfo struct {
	VoiceID        string    `json:"voice_id"`
	CreatedAt      time.Time `json:"created_at"`
	ReferenceAudio string    `json:"reference_audio"`
}

type VoiceManager struct {
	redis *redis.Client
}

func NewVoiceManager(redisClient *redis.Client) *VoiceManager {
	return &VoiceManager{redis: redisClient}
}

// GetVoiceID 从缓存中获取 voice_id
func (vm *VoiceManager) GetVoiceID(ctx context.Context, speakerID string) (string, error) {
	key := fmt.Sprintf("voice:%s", speakerID)
	data, err := vm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // 未缓存
	}
	if err != nil {
		return "", err
	}

	var voiceInfo VoiceInfo
	if err := json.Unmarshal([]byte(data), &voiceInfo); err != nil {
		return "", err
	}

	return voiceInfo.VoiceID, nil
}

// CacheVoiceID 缓存 voice_id
func (vm *VoiceManager) CacheVoiceID(ctx context.Context, speakerID, voiceID, referenceAudio string) error {
	key := fmt.Sprintf("voice:%s", speakerID)
	voiceInfo := VoiceInfo{
		VoiceID:        voiceID,
		CreatedAt:      time.Now(),
		ReferenceAudio: referenceAudio,
	}

	data, err := json.Marshal(voiceInfo)
	if err != nil {
		return err
	}

	// 缓存 7 天
	return vm.redis.Set(ctx, key, data, 7*24*time.Hour).Err()
}
```

---

## 3. 后续实现任务

### 3.1 待实现的功能

1. **internal/adapters/aliyun_asr.go**: 阿里云 ASR 适配器实现
2. **internal/adapters/deepl_translation.go**: DeepL 翻译适配器实现
3. **internal/adapters/openai_llm.go**: OpenAI LLM 适配器实现
4. **internal/logic/ai_adaptor_logic.go**: 主业务逻辑（适配器选择、配置读取）
5. **internal/storage/redis.go**: Redis 操作实现
6. **阿里云 API 集成**: 完成 OSS 上传、CosyVoice API 调用
7. **单元测试**: 各个适配器的单元测试

### 3.2 依赖库

```go
// go.mod
module ai-adaptor

go 1.21

require (
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/aliyun/aliyun-oss-go-sdk v2.2.9+incompatible
)
```

---

## 4. Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ai-adaptor main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/ai-adaptor .

EXPOSE 50053
CMD ["./ai-adaptor"]
```

---

