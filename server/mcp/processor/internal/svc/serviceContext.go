package svc

import (
	"fmt"

	"video-in-chinese/server/mcp/processor/internal/config"
	"video-in-chinese/server/mcp/processor/internal/mediautil"
	"video-in-chinese/server/mcp/processor/internal/storage"
	"video-in-chinese/server/mcp/processor/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext holds all dependencies for the Processor service.
//
// This context is created once at service startup and shared across
// all request handlers. It manages Redis client, gRPC clients, and
// file path management.
type ServiceContext struct {
	Config config.Config

	// Storage layer
	RedisClient *storage.RedisClient
	PathManager *storage.PathManager

	// gRPC clients
	AIAdaptorClient      pb.AIAdaptorClient
	AudioSeparatorClient pb.AudioSeparatorClient
}

// NewServiceContext creates a new ServiceContext instance.
//
// This function initializes all dependencies including:
//   - Redis client for task queue and status management
//   - gRPC clients for AIAdaptor and AudioSeparator services
//   - Path manager for file path generation
//
// Parameters:
//   - c: configuration loaded from YAML file
//
// Returns:
//   - *ServiceContext: initialized service context
func NewServiceContext(c config.Config) *ServiceContext {
	// Initialize Redis client
	rdb := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})

	// Ping Redis to verify connection
	ok := rdb.Ping()
	if !ok {
		logx.Errorf("[ServiceContext] Failed to connect to Redis at %s", c.Redis.Host)
		panic(fmt.Sprintf("failed to connect to Redis at %s", c.Redis.Host))
	}
	logx.Infof("[ServiceContext] Connected to Redis at %s", c.Redis.Host)

	// Initialize storage layer
	redisClient := storage.NewRedisClient(rdb)
	pathManager := storage.NewPathManager(c.LocalStoragePath)

	// Initialize ffmpeg binary path if provided
	mediautil.SetFFmpegBinary(c.FfmpegPath)

	// Initialize AIAdaptor gRPC client (using GoZero zrpc)
	aiAdaptorRpcClient := zrpc.MustNewClient(c.AIAdaptorRpcConf)
	aiAdaptorClient := pb.NewAIAdaptorClient(aiAdaptorRpcClient.Conn())
	logx.Infof("[ServiceContext] Connected to AIAdaptor at %v", c.AIAdaptorRpcConf.Endpoints)

	var audioSeparatorClient pb.AudioSeparatorClient
	if len(c.AudioSeparatorRpcConf.Endpoints) > 0 || len(c.AudioSeparatorRpcConf.Target) > 0 || len(c.AudioSeparatorRpcConf.Etcd.Hosts) > 0 {
		audioSeparatorRpcClient, err := zrpc.NewClient(c.AudioSeparatorRpcConf)
		if err != nil {
			logx.Infof("[ServiceContext] AudioSeparator unavailable, disabling separation: %v", err)
		} else {
			audioSeparatorClient = pb.NewAudioSeparatorClient(audioSeparatorRpcClient.Conn())
			logx.Infof("[ServiceContext] Connected to AudioSeparator at %v", c.AudioSeparatorRpcConf.Endpoints)
		}
	} else {
		logx.Infof("[ServiceContext] AudioSeparator RPC not configured; skipping client initialization")
	}

	return &ServiceContext{
		Config:               c,
		RedisClient:          redisClient,
		PathManager:          pathManager,
		AIAdaptorClient:      aiAdaptorClient,
		AudioSeparatorClient: audioSeparatorClient,
	}
}
