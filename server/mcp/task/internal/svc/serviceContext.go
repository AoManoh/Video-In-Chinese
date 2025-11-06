package svc

import (
	"log"

	"video-in-chinese/server/mcp/task/internal/config"
	"video-in-chinese/server/mcp/task/internal/storage"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *storage.RedisClient
	FileStorage *storage.FileStorage
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 Redis 客户端
	redisClient, err := storage.NewRedisClient(storage.RedisConfig{
		Host: c.TaskRedis.Host,
		Type: c.TaskRedis.Type,
		Pass: c.TaskRedis.Pass,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	// 初始化文件存储
	fileStorage, err := storage.NewFileStorage(c.LocalStoragePath)
	if err != nil {
		log.Fatalf("Failed to initialize file storage: %v", err)
	}

	return &ServiceContext{
		Config:      c,
		RedisClient: redisClient,
		FileStorage: fileStorage,
	}
}
