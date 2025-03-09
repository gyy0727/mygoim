package main

import (
	"context"
	"fmt"
	"net"

	pb "github.com/gyy0727/mygoim/test/grpc/api" // 导入生成的 Protobuf 包
	"google.golang.org/grpc"
)

// 定义服务端结构体
type server struct {
	pb.UnimplementedAddtwoServer // 嵌入默认实现
}

// 实现 add 方法
func (s *server) Add(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	// 计算 x + y
	result := req.X + req.Y
	// 返回响应
	return &pb.Response{Z: result}, nil
}

func main() {
	// 监听本地 50051 端口
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("failed to listen: %v\n", err)
		return
	}

	// 创建 gRPC 服务器
	s := grpc.NewServer()
	// 注册服务
	pb.RegisterAddtwoServer(s, &server{})

	// 启动服务
	fmt.Println("Server listening on :50051")
	if err := s.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v\n", err)
		return
	}
}