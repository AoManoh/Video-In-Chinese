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

	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters"
	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters/asr"
	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters/llm"
	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters/translation"
	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters/voice_cloning"
	"video-in-chinese/server/mcp/ai_adaptor/internal/config"
	"video-in-chinese/server/mcp/ai_adaptor/internal/logic"
	"video-in-chinese/server/mcp/ai_adaptor/internal/voice_cache"
	pb "video-in-chinese/server/mcp/ai_adaptor/proto"
)

// server 实现 AIAdaptor gRPC 服务
type server struct {
	pb.UnimplementedAIAdaptorServer
	registry      *adapters.AdapterRegistry
	configManager *config.ConfigManager
	redisClient   *config.RedisClient
	cryptoManager *config.CryptoManager
}

// ASR 实现 ASR 服务
func (s *server) ASR(ctx context.Context, req *pb.ASRRequest) (*pb.ASRResponse, error) {
	// 创建 ASR 服务逻辑实例
	asrLogic := logic.NewASRLogic(s.registry, s.configManager)
	// 调用服务逻辑处理请求
	return asrLogic.ProcessASR(ctx, req)
}

// Polish 实现文本润色服务
func (s *server) Polish(ctx context.Context, req *pb.PolishRequest) (*pb.PolishResponse, error) {
	// 创建文本润色服务逻辑实例
	polishLogic := logic.NewPolishLogic(s.registry, s.configManager)
	// 调用服务逻辑处理请求
	return polishLogic.ProcessPolish(ctx, req)
}

// Translate 实现翻译服务
func (s *server) Translate(ctx context.Context, req *pb.TranslateRequest) (*pb.TranslateResponse, error) {
	// 创建翻译服务逻辑实例
	translateLogic := logic.NewTranslateLogic(s.registry, s.configManager)
	// 调用服务逻辑处理请求
	return translateLogic.ProcessTranslate(ctx, req)
}

// Optimize 实现译文优化服务
func (s *server) Optimize(ctx context.Context, req *pb.OptimizeRequest) (*pb.OptimizeResponse, error) {
	// 创建译文优化服务逻辑实例
	optimizeLogic := logic.NewOptimizeLogic(s.registry, s.configManager)
	// 调用服务逻辑处理请求
	return optimizeLogic.ProcessOptimize(ctx, req)
}

// CloneVoice 实现声音克隆服务
func (s *server) CloneVoice(ctx context.Context, req *pb.CloneVoiceRequest) (*pb.CloneVoiceResponse, error) {
	// 创建声音克隆服务逻辑实例
	cloneVoiceLogic := logic.NewCloneVoiceLogic(s.registry, s.configManager)
	// 调用服务逻辑处理请求
	return cloneVoiceLogic.ProcessCloneVoice(ctx, req)
}

// initializeAdapters 初始化并注册所有适配器
// 参数:
//   - registry: 适配器注册表
//   - voiceManager: 音色缓存管理器（用于声音克隆适配器）
func initializeAdapters(registry *adapters.AdapterRegistry, voiceManager *voice_cache.VoiceManager) {
	log.Println("[initializeAdapters] Registering all adapters...")

	// 注册 ASR 适配器
	registry.RegisterASR("aliyun", asr.NewAliyunASRAdapter())
	log.Println("✓ Registered ASR adapter: aliyun")

	registry.RegisterASR("azure", asr.NewAzureASRAdapter())
	log.Println("✓ Registered ASR adapter: azure")

	registry.RegisterASR("google", asr.NewGoogleASRAdapter())
	log.Println("✓ Registered ASR adapter: google")

	// 注册翻译适配器
	registry.RegisterTranslation("google", translation.NewGoogleTranslationAdapter())
	log.Println("✓ Registered translation adapter: google")

	registry.RegisterTranslation("openai-compatible", translation.NewOpenAICompatibleTranslationAdapter())
	log.Println("✓ Registered translation adapter: openai-compatible")

	// 注册 LLM 适配器
	registry.RegisterLLM("gemini", llm.NewGeminiLLMAdapter())
	log.Println("✓ Registered LLM adapter: gemini")

	registry.RegisterLLM("openai", llm.NewOpenAILLMAdapter())
	log.Println("✓ Registered LLM adapter: openai")

	registry.RegisterLLM("openai-compatible", llm.NewOpenAILLMAdapter())
	log.Println("✓ Registered LLM adapter: openai-compatible")

	// 注册声音克隆适配器
	registry.RegisterVoiceCloning("aliyun-cosyvoice", voice_cloning.NewAliyunCosyVoiceAdapter(voiceManager))
	log.Println("✓ Registered voice cloning adapter: aliyun-cosyvoice")

	log.Println("[initializeAdapters] All adapters registered successfully")
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

	// 创建配置管理器
	configManager := config.NewConfigManager(redisClient, cryptoManager)
	log.Println("✓ Config manager created")

	// 创建音色缓存管理器
	voiceManager := voice_cache.NewVoiceManager(redisClient)
	log.Println("✓ Voice manager created")

	// 创建适配器注册表
	registry := adapters.NewAdapterRegistry()
	log.Println("✓ Adapter registry created")

	// 初始化并注册所有适配器
	initializeAdapters(registry, voiceManager)

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()
	pb.RegisterAIAdaptorServer(grpcServer, &server{
		registry:      registry,
		configManager: configManager,
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
