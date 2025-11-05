// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"video-in-chinese/server/app/gateway/internal/config"
	"video-in-chinese/server/mcp/task-gozero/proto"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	// Redis client for configuration management
	RedisClient *redis.Redis

	// Task service gRPC client
	TaskRpcClient proto.TaskServiceClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Initialize Redis client
	redisClient := redis.MustNewRedis(c.Redis)

	// Initialize Task service gRPC client
	taskRpcConn := zrpc.MustNewClient(c.TaskRpcConf)

	return &ServiceContext{
		Config:        c,
		RedisClient:   redisClient,
		TaskRpcClient: proto.NewTaskServiceClient(taskRpcConn.Conn()),
	}
}
