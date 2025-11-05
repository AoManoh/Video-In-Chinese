package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config defines the configuration for the Processor service.
//
// This configuration includes Redis connection, gRPC client configurations,
// task queue settings, and concurrency control.
type Config struct {
	service.ServiceConf

	// Redis configuration
	Redis struct {
		Host string
		Type string
		Pass string
	}

	// gRPC client configurations (GoZero RpcClientConf format)
	AIAdaptorRpcConf      zrpc.RpcClientConf
	AudioSeparatorRpcConf zrpc.RpcClientConf

	// Local storage path for video files
	LocalStoragePath string

	// Task queue configuration
	TaskQueueKey            string
	TaskPullIntervalSeconds int

	// Concurrency control
	MaxConcurrency int

	// ffmpeg path (optional, defaults to system PATH)
	FfmpegPath string `json:",optional"`
}
