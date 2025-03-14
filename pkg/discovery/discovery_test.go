package discovery

import (
	"testing"
	"time"
)

func TestDiscoverService(t *testing.T) {
	// 初始化 etcd 解析器
	etcdResolverInit()

	// 设置 etcd 地址
	eResolver.etcdAddrs = []string{"47.115.200.76:2379"}

	// 设置需要解析的服务名称
	serviceName := "service1"

	// 设置目标节点
	eResolver.setTargetNode(serviceName)

	// 等待解析完成
	time.Sleep(2 * time.Second)

	// 获取解析到的服务节点
	nodes := eResolver.getServiceNodes(serviceName)
	if len(nodes) == 0 {
		t.Fatalf("No service nodes found for %s", serviceName)
	}

	// 打印解析到的服务节点
	for _, node := range nodes {
		t.Logf("Discovered service node: %s", node.Addr)
	}

	// 停止解析器
	eResolver.stop()
}
