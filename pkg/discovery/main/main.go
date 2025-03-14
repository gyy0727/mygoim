package main

import (
	"os"
	"os/signal"
	"syscall"

	discovery "github.com/gyy0727/mygoim/pkg/discovery" // 替换为你的项目路径
)

func main() {
	// 1. 初始化 etcd 注册器
	discovery.etcdRegisterInit()

	// 2. 创建服务节点
	node := &discovery.Node{
		Name: "my_service",     // 服务名称
		Addr: "127.0.0.1:8080", // 服务地址
	}

	// 3. 注册服务节点
	eRegister := discovery.eRegister
	eRegister.addServiceNode(node)

	// 4. 保持程序运行
	// 使用信号监听来优雅地关闭程序
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 5. 停止注册任务
	eRegister.stop()
}
