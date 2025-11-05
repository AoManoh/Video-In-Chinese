package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"video-in-chinese/server/mcp/processor/internal/config"
	"video-in-chinese/server/mcp/processor/internal/logic"
	"video-in-chinese/server/mcp/processor/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/processor.yaml", "the config file")

func main() {
	flag.Parse()

	// Load configuration
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Initialize service context
	svcCtx := svc.NewServiceContext(c)

	logx.Infof("Starting Processor service...")
	logx.Infof("Config: %+v", c)

	// Create context for graceful shutdown
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create worker semaphore for concurrency control
	workerSem := make(chan struct{}, c.MaxConcurrency)

	// Start task pull loop in separate goroutine
	go logic.StartTaskPullLoop(mainCtx, svcCtx, workerSem)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logx.Info("Shutting down Processor service...")
	cancel()

	logx.Info("Processor service stopped")
}
