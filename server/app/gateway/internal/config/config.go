// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	// Redis configuration
	Redis redis.RedisConf

	// Task service gRPC configuration
	TaskRpcConf zrpc.RpcClientConf

	// File storage configuration
	TempStoragePath  string
	LocalStoragePath string

	// Upload configuration
	MaxUploadSizeMB    int
	SupportedMimeTypes []string

	// API Key encryption configuration
	ApiKeyEncryptionSecret string

	// HTTP configuration
	HttpTimeoutSeconds       int
	MaxConcurrentConnections int
}
