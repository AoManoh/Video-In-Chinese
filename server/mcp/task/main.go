package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"video-in-chinese/task/internal/logic"
	"video-in-chinese/task/internal/svc"
	pb "video-in-chinese/task/proto"
)

// server 实现 TaskService gRPC 服务
type server struct {
	pb.UnimplementedTaskServiceServer
	svcCtx *svc.ServiceContext
}

// CreateTask 实现 CreateTask RPC 方法
func (s *server) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	// 创建业务逻辑实例
	l := logic.NewCreateTaskLogic(ctx, s.svcCtx)
	// 调用业务逻辑处理请求
	return l.CreateTask(req)
}

// GetTaskStatus 实现 GetTaskStatus RPC 方法
func (s *server) GetTaskStatus(ctx context.Context, req *pb.GetTaskStatusRequest) (*pb.GetTaskStatusResponse, error) {
	// 创建业务逻辑实例
	l := logic.NewGetTaskStatusLogic(ctx, s.svcCtx)
	// 调用业务逻辑处理请求
	return l.GetTaskStatus(req)
}

func main() {
	// 读取配置
	port := os.Getenv("TASK_GRPC_PORT")
	if port == "" {
		port = "50050"
	}

	log.Printf("Starting Task service on port %s...", port)

	// 初始化服务上下文
	svcCtx, err := svc.NewServiceContext()
	if err != nil {
		log.Fatalf("Failed to create service context: %v", err)
	}
	defer svcCtx.Close()
	log.Println("✓ Service context initialized")

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, &server{
		svcCtx: svcCtx,
	})
	log.Println("✓ gRPC server created")

	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// 启动 gRPC 服务器（在 goroutine 中）
	go func() {
		log.Printf("✓ Task service listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	log.Println("Shutting down Task service...")
	grpcServer.GracefulStop()
	log.Println("✓ Task service stopped")
}

