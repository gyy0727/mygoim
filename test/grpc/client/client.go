package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/gyy0727/mygoim/test/grpc/api" // 导入生成的 Protobuf 包
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 连接到服务端
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// 创建 gRPC 客户端
	client := pb.NewAddtwoClient(conn)

	// 创建请求
	req := &pb.Request{
		X: 10,
		Y: 20,
	}

	// 调用服务
	resp, err := client.Add(context.Background(), req)
	if err != nil {
		log.Fatalf("failed to call Add: %v", err)
	}

	// 打印结果
	fmt.Printf("Response: z = %d\n", resp.Z)
}
