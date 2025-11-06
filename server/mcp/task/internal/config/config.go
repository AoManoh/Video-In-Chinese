package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	RpcServerConf    zrpc.RpcServerConf `json:"RpcServerConf" yaml:"RpcServerConf"`
	TaskRedis        redis.RedisConf    `json:"TaskRedis" yaml:"TaskRedis"`
	LocalStoragePath string             `json:"LocalStoragePath" yaml:"LocalStoragePath"`
}
