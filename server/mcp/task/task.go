package main

import (
	"flag"
	"fmt"

	"video-in-chinese/server/mcp/task/internal/config"
	"video-in-chinese/server/mcp/task/internal/server"
	"video-in-chinese/server/mcp/task/internal/svc"
	"video-in-chinese/server/mcp/task/proto"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/task.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		proto.RegisterTaskServiceServer(grpcServer, server.NewTaskServiceServer(ctx))

		if c.RpcServerConf.Mode == service.DevMode || c.RpcServerConf.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.RpcServerConf.ListenOn)
	s.Start()
}
