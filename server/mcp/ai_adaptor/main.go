package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"video-in-chinese/ai_adaptor/internal/adapters"
	"video-in-chinese/ai_adaptor/internal/config"
	pb "video-in-chinese/ai_adaptor/proto"
)

// server 实现 AIAdaptor gRPC 服务
type server struct {
	pb.UnimplementedAIAdaptorServer
	registry      *adapters.AdapterRegistry
	redisClient   *config.RedisClient
	cryptoManager *config.CryptoManager
}

// ASR 实现 ASR 服务
func (s *server) ASR(ctx context.Context, req *pb.ASRRequest) (*pb.ASRResponse, error) {
	// TODO: 实现 ASR 服务逻辑（Phase 5）
	return nil, status.Errorf(codes.Unimplemented, "ASR service not implemented yet")
}

// Polish 实现文本润色服务
func (s *server) Polish(ctx context.Context, req *pb.PolishRequest) (*pb.PolishResponse, error) {
	// TODO: 实现文本润色服务逻辑（Phase 5）
	return nil, status.Errorf(codes.Unimplemented, "Polish service not implemented yet")
}

// Translate 实现翻译服务
func (s *server) Translate(ctx context.Context, req *pb.TranslateRequest) (*pb.TranslateResponse, error) {
	// TODO: 实现翻译服务逻辑（Phase 5）
	return nil, status.Errorf(codes.Unimplemented, "Translate service not implemented yet")
}

// Optimize 实现译文优化服务
func (s *server) Optimize(ctx context.Context, req *pb.OptimizeRequest) (*pb.OptimizeResponse, error) {
	// TODO: 实现译文优化服务逻辑（Phase 5）
	return nil, status.Errorf(codes.Unimplemented, "Optimize service not implemented yet")
}

// CloneVoice 实现声音克隆服务
func (s *server) CloneVoice(ctx context.Context, req *pb.CloneVoiceRequest) (*pb.CloneVoiceResponse, error) {
	// TODO: 实现声音克隆服务逻辑（Phase 5）
	return nil, status.Errorf(codes.Unimplemented, "CloneVoice service not implemented yet")
}

// initializeAdapters 初始化并注册所有适配器
func initializeAdapters(registry *adapters.AdapterRegistry) {
	// TODO: Phase 4 - 注册 ASR 适配器
	// registry.RegisterASR("aliyun", &asr.AliyunASRAdapter{})
	// registry.RegisterASR("azure", &asr.AzureASRAdapter{})
	// registry.RegisterASR("google", &asr.GoogleASRAdapter{})

	// TODO: Phase 4 - 注册翻译适配器
	// registry.RegisterTranslation("deepl", &translation.DeepLAdapter{})
	// registry.RegisterTranslation("google", &translation.GoogleTranslateAdapter{})
	// registry.RegisterTranslation("azure", &translation.AzureTranslateAdapter{})

	// TODO: Phase 4 - 注册 LLM 适配器
	// registry.RegisterLLM("openai-gpt4o", &llm.OpenAIAdapter{})
	// registry.RegisterLLM("claude", &llm.ClaudeAdapter{})
	// registry.RegisterLLM("gemini", &llm.GeminiAdapter{})

	// TODO: Phase 4 - 注册声音克隆适配器
	// registry.RegisterVoiceCloning("aliyun_cosyvoice", &voice_cloning.AliyunCosyVoiceAdapter{})

	log.Println("Adapters initialized (placeholder - will be implemented in Phase 4)")
}

func main() {
	// 读取配置
	port := os.Getenv("AI_ADAPTOR_GRPC_PORT")
	if port == "" {
		port = "50053"
	}

	log.Printf("Starting AIAdaptor service on port %s...", port)

	// 初始化 Redis 客户端
	redisClient, err := config.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()
	log.Println("✓ Redis client initialized")

	// 初始化加密管理器
	cryptoManager, err := config.NewCryptoManager()
	if err != nil {
		log.Fatalf("Failed to create crypto manager: %v", err)
	}
	log.Println("✓ Crypto manager initialized")

	// 创建适配器注册表
	registry := adapters.NewAdapterRegistry()
	log.Println("✓ Adapter registry created")

	// 初始化并注册所有适配器
	initializeAdapters(registry)

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()
	pb.RegisterAIAdaptorServer(grpcServer, &server{
		registry:      registry,
		redisClient:   redisClient,
		cryptoManager: cryptoManager,
	})
	log.Println("✓ gRPC server created")

	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// 启动 gRPC 服务器（在 goroutine 中）
	go func() {
		log.Printf("✓ AIAdaptor service listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	log.Println("Shutting down AIAdaptor service...")
	grpcServer.GracefulStop()
	log.Println("✓ AIAdaptor service stopped")
}

