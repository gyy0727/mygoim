package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	discovery "github.com/gyy0727/mygoim/pkg/discovery" // 替换为你的项目路径
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	// 1. 初始化 etcd 注册器
	discovery.EtcdResolverInit()

	// 2. 创建服务节点
	node := &discovery.Node{
		Name: "my_service",     // 服务名称
		Addr: "127.0.0.1:8080", // 服务地址
	}

	// 3. 注册服务节点
	eRegister := discovery.ERegister
	eRegister.AddServiceNode(node)

	// 4. 查询 etcd 中的服务节点
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"47.115.200.76:2379"}, // etcd 地址
	})
	if err != nil {
		fmt.Println("Failed to connect to etcd:", err)
		return
	}
	defer cli.Close()

	resp, err := cli.Get(context.Background(), "/my_service", clientv3.WithPrefix())
	if err != nil {
		fmt.Println("Failed to query etcd:", err)
		return
	}

	for _, kv := range resp.Kvs {
		fmt.Printf("Key: %s, Value: %s\n", kv.Key, kv.Value)
	}

	// 5. 保持程序运行
	// 使用信号监听来优雅地关闭程序
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 6. 停止注册任务
	eRegister.Stop()
}
