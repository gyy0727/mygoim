package main

import (
	_ "context"
	_ "encoding/json"
	"fmt"
	_ "fmt"
	_ "os"
	_ "os/signal"
	"runtime"
	_ "strings"
	_ "sync"
	_ "syscall"
	"time"
	_ "time"

	"github.com/gyy0727/mygoim/pkg/discovery"
	_ "github.com/gyy0727/mygoim/pkg/discovery"

	_ "github.com/golang/glog"
	_ "go.etcd.io/etcd/client/v3"
	_ "google.golang.org/grpc/resolver"
)

func main() {
	discovery.EtcdResolverInit()
	discovery.EResolver.SetTargetNode("test")
	// 定义一个简单的循环函数
	loopFunction := func() {
		for i := 0; i < 5; i++ {
			// 输出当前的协程数量
			fmt.Printf("Loop iteration %d, Number of goroutines: %d\n", i, runtime.NumGoroutine())
			time.Sleep(500 * time.Millisecond) // 模拟一些工作
		}
	}

	// 启动循环函数
	loopFunction()
	buf := make([]byte, 1<<20) // 分配足够大的缓冲区
	stackSize := runtime.Stack(buf, true)
	fmt.Printf("All goroutines stack:\n%s\n", buf[:stackSize])
	// 输出最终的协程数量
	fmt.Printf("Final number of goroutines: %d\n", runtime.NumGoroutine())
}
