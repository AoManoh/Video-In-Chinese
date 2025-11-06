package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	// Redis configuration for task storage
	Redis struct {
		Host string
		Type string
		Pass string
	}

	LocalStoragePath string
}
