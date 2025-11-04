package svc

import (
	"log"

	"video-in-chinese/task/internal/storage"
)

// ServiceContext 服务上下文，管理所有依赖
// 采用依赖注入模式，便于测试和维护
type ServiceContext struct {
	RedisClient *storage.RedisClient
	FileStorage *storage.FileStorage
}

// NewServiceContext 创建服务上下文实例
// 初始化所有依赖：Redis 客户端、文件存储
func NewServiceContext() (*ServiceContext, error) {
	log.Println("[ServiceContext] Initializing service context...")

	// 初始化 Redis 客户端
	redisClient, err := storage.NewRedisClient()
	if err != nil {
		return nil, err
	}

	// 初始化文件存储
	fileStorage, err := storage.NewFileStorage()
	if err != nil {
		return nil, err
	}

	log.Println("[ServiceContext] ✓ Service context initialized")

	return &ServiceContext{
		RedisClient: redisClient,
		FileStorage: fileStorage,
	}, nil
}

// Close 关闭所有资源
func (svc *ServiceContext) Close() error {
	log.Println("[ServiceContext] Closing service context...")
	
	if err := svc.RedisClient.Close(); err != nil {
		log.Printf("[ServiceContext] Failed to close Redis client: %v", err)
		return err
	}

	log.Println("[ServiceContext] ✓ Service context closed")
	return nil
}

