package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	discovery "github.com/gyy0727/mygoim/pkg/discovery"
)

func main() {
	
	// 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic: %v\n", r)
		}
	}()

	// 确认程序启动
	fmt.Println("Program started")

	// 初始化 etcd 注册
	discovery.EtcdRegisterInit()

	// 打印环境变量
	envEtcdAddr := os.Getenv("DISCOVERY_HOST")
	fmt.Printf("DISCOVERY_HOST: %s\n", envEtcdAddr)

	// 检查 ERegister 是否已初始化
	if discovery.ERegister == nil {
		fmt.Println("ERegister is nil. Initializing...")
	}

	// 添加服务节点
	node := discovery.Node{
		Name: "test",
		Addr: "IP_ADDRESS:8080",
	}
	fmt.Printf("Node: %+v\n", node) // 打印节点信息

	discovery.ERegister.AddServiceNode(&node)

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Waiting for termination signal...")
	sig := <-sigChan
	fmt.Printf("Received signal: %v. Exiting...\n", sig)
}
