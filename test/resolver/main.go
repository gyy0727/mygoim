package main

import (
	"fmt"
	"runtime"
	"time"

	discovery "github.com/gyy0727/mygoim/pkg/discovery"
)

func main() {

	discovery.EtcdResolverInit()

	discovery.EResolver.SetTargetNode("goim/logic")
	go func() {
		for {
			time.Sleep(1 * time.Second)
			node := discovery.EResolver.GetServiceNodes("goim/logic")
			if len(node) == 0 {
				fmt.Println("node is nil")
				fmt.Println("Number of goroutines:", runtime.NumGoroutine())
				continue
			}
			fmt.Println("Number of goroutines:", runtime.NumGoroutine())
			fmt.Println("--------node:%s", node[0].Addr)
			time.Sleep(3 * time.Second)
		}
	}()
	time.Sleep(5 * time.Second)
	// fmt.Println("删除要解析的目标节点")
	// discovery.EResolver.DetTargetNodes("test")
	time.Sleep(10 * time.Second)
	// discovery.EResolver.Stop()
	time.Sleep(10 * time.Second)
	fmt.Println("Number of goroutines--stop:", runtime.NumGoroutine())
	// time.Sleep(5 * time.Minute)
	// sigChan := make(chan os.Signal, 1)

	// // 通知 signal 包捕获 SIGINT 信号（Ctrl+C）
	// signal.Notify(sigChan, syscall.SIGINT)
	// <-sigChan
}
