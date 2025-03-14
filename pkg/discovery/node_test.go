package discovery

import (
	"fmt"
	"testing"
)

func TestNodeMethods(t *testing.T) {
	// 创建一个 Node 实例
	node := Node{
		Name: "service.example.com",
		Addr: "127.0.0.1:8080",
	}

	// 测试 transName 方法
	transName := node.transName()
	fmt.Printf("Input Name: %s\n", node.Name)
	fmt.Printf("Output transName: %s\n", transName)

	// 测试 buildKey 方法
	key := node.buildKey()
	fmt.Printf("Input Name: %s, Addr: %s\n", node.Name, node.Addr)
	fmt.Printf("Output buildKey: %s\n", key)

	// 测试 buildPrefix 方法
	prefix := node.buildPrefix()
	fmt.Printf("Input Name: %s\n", node.Name)
	fmt.Printf("Output buildPrefix: %s\n", prefix)
}
