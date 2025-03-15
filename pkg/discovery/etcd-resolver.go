// etcdResolver.go

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/golang/glog"
	etcdV3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

const (
	tickerTime = 1 * time.Second //*etcdResolver 定期执行服务节点解析的频率
)

type etcdResolver struct {
	//*记录所有创建的解析器，同一个host只创建一个解析器
	mr map[string]resolver.Resolver
	//*读写锁
	mrMux sync.RWMutex

	//*etcd 客户端
	cli *etcdV3.Client
	//*etcd 地址
	etcdAddrs []string
	//*连接 etcd 超时时间
	dialTimeout time.Duration

	tnsMux sync.RWMutex
	//*需要解析的目标节点
	targetNodeSet map[string]*Node

	snsMux sync.RWMutex
	//*解析到的服务节点， host:addr:*Node
	serviceNodes map[string]map[string]*Node

	cancel context.CancelFunc

	once sync.Once
}

// *返回name对应的节点信息
func (e *etcdResolver) GetServiceNodes(name string) []*Node {
	e.snsMux.RLock()
	defer e.snsMux.RUnlock()
	nodes := make([]*Node, 0)
	for _, n := range e.serviceNodes[name] {
		nodes = append(nodes, n)

	}
	return nodes
}

// *将解析到的节点信息添加到ServiceNodes中
func (e *etcdResolver) setServiceNodes(name string, nodes ...*Node) {
	e.snsMux.Lock()
	defer e.snsMux.Unlock()
	ns := e.serviceNodes[name]
	if ns == nil {
		ns = make(map[string]*Node)
	}
	for i := range nodes {
		log.Infof("resolver node [%s:%s]", name, nodes[i].Addr)
		ns[nodes[i].Addr] = nodes[i]
	}
	e.serviceNodes[name] = ns
}

// *删除解析完成服务节点
func (e *etcdResolver) removeServiceNode(name, addr string) {
	fmt.Println("remove node------------ [%s:%s]", name, addr)
	e.snsMux.Lock()
	defer e.snsMux.Unlock()
	nodes := e.serviceNodes[name]
	if nodes == nil {
		return
	}
	delete(nodes, addr)
}

func (e *etcdResolver) removeServiceNodeWithPrefix(name string) {
	fmt.Println("remove node------------ [%s]", name)
	e.snsMux.Lock()
	defer e.snsMux.Unlock()
	delete(e.serviceNodes, name)
}

// *设置host对应的解析器
func (e *etcdResolver) SetManuResolver(host string, m resolver.Resolver) {
	e.mrMux.Lock()
	defer e.mrMux.Unlock()
	e.mr[host] = m
}

// *根据host获取解析器
func (e *etcdResolver) getManuResolver(host string) (resolver.Resolver, bool) {
	e.mrMux.RLock()
	defer e.mrMux.RUnlock()
	if m, ok := e.mr[host]; ok {
		return m, ok
	}
	return nil, false
}

// *设置需要解析的目标节点
func (e *etcdResolver) SetTargetNode(host string) {
	e.tnsMux.Lock()
	e.targetNodeSet[host] = &Node{Name: host}
	e.tnsMux.Unlock()

	//*开始解析时进行相关操作，只执行一次
	e.once.Do(func() {
		fmt.Println("start resolver")
		var ctx context.Context
		ctx, e.cancel = context.WithCancel(context.Background())
		e.start(ctx)
	})
}

// *获取需要解析目标节点
func (e *etcdResolver) getTargetNodes() []*Node {
	e.tnsMux.RLock()
	defer e.tnsMux.RUnlock()

	nodes := make([]*Node, 0)
	for _, n := range e.targetNodeSet {
		nodes = append(nodes, n)
	}
	return nodes
}

func (e *etcdResolver) DetTargetNodes(Name string) {
	e.tnsMux.RLock()
	defer e.tnsMux.RUnlock()
	delete(e.targetNodeSet, Name)

}

// *解析所有需要解析的节点
func (e *etcdResolver) resolverAll(ctx context.Context) {
	nodes := e.getTargetNodes()
	for _, node := range nodes {
		//*根据前缀获取节点信息
		cctx, cancel := context.WithTimeout(context.Background(), e.dialTimeout)
		//*从etcd获取到键值对
		rsp, err := e.cli.Get(cctx, node.buildPrefix(), etcdV3.WithPrefix())
		cancel()
		if err != nil {
			log.Errorf("get service node [%s] error:%s", node.Name, err.Error())
			continue
		}
		for j := range rsp.Kvs {
			n := &Node{}
			err = json.Unmarshal(rsp.Kvs[j].Value, n)
			if err != nil {
				log.Errorf("get service node [%s] error:%s", node.Name, err.Error())
				continue
			}
			//*将解析出的节点存储到ServiceNodes中
			e.setServiceNodes(node.Name, n)
		}
	}

	//*解析完服务节点后，更新到连接上
	e.mrMux.RLock()
	defer e.mrMux.RUnlock()
	for _, v := range e.mr {
		v.ResolveNow(resolver.ResolveNowOptions{})
	}
}

func (e *etcdResolver) start(ctx context.Context) {
	fmt.Println("开启执行start 协程")
	if len(e.etcdAddrs) == 0 {
		panic("discovery should call SetDiscoveryAddress or set env DISCOVERY_HOST")
	}

	var err error
	e.cli, err = etcdV3.New(etcdV3.Config{
		Endpoints:   e.etcdAddrs,
		DialTimeout: e.dialTimeout,
	})
	if err != nil {
		panic(err)
	}

	//*开始先全部解析
	e.resolverAll(ctx)

	monitoredNodes := make(map[string]bool)
	NodeCancel := make(map[string]func())
	tickerResolver := time.NewTicker(tickerTime)
	tickerWatch := time.NewTicker(tickerTime)

	//*定时解析
	go func() {
		fmt.Println("执行定时解析")
		for {
			select {
			case <-tickerResolver.C:
				fmt.Println("定时解析ticker")
				e.resolverAll(ctx)

			case <-ctx.Done():
				log.Infoln("resolver ticker exit")
				return
			}
		}
	}()

	//*每个节点watch变化
	go func() {
		fmt.Println("执行定时监听")
		for {
			select {
			case <-tickerWatch.C:
				nodes := e.getTargetNodes()
				fmt.Println("定时监听ticker")
				currentNodes := make(map[string]bool)
				//*检查新增节点
				for _, node := range nodes {

					currentNodes[node.Name] = true
					if !monitoredNodes[node.Name] {
						cctx, cancel := context.WithCancel(ctx)
						monitoredNodes[node.Name] = true
						NodeCancel[node.Name] = cancel
						fmt.Println("监听节点：", node.Name)
						go e.watchNode(cctx, node)
					}
				}
				if nodes == nil {
					currentNodes = nil
				}

				//*检查删除的节点
				for name := range monitoredNodes {
					if !currentNodes[name] {
						NodeCancel[name]()
						delete(NodeCancel, name)
						e.removeServiceNodeWithPrefix(name)
						fmt.Println("删除节点%s", name)
						delete(monitoredNodes, name)
					}
				}

			case <-ctx.Done():
				log.Infoln("resolver ticker exit")
				return
			}
		}
	}()

}

func (e *etcdResolver) watchNode(ctx context.Context, node *Node) {
	fmt.Println("watching node")
	cctx, cancel := context.WithCancel(ctx)
	defer cancel() //*确保在函数退出时取消上下文
	wc := e.cli.Watch(cctx, node.buildPrefix(), etcdV3.WithPrefix())
	for {
		select {
		case rsp := <-wc:
			for _, event := range rsp.Events {
				switch event.Type {
				case etcdV3.EventTypePut:
					n := &Node{}
					err := json.Unmarshal(event.Kv.Value, n)
					if err != nil {
						log.Errorf("unmarshal to node error:%s", err.Error())
						continue
					}
					e.setServiceNodes(node.Name, n)
				case etcdV3.EventTypeDelete:
					name, addr, err := node.SplitPath(event.Kv.String())
					if err != nil {
						panic("解析节点错误")

					}
					e.removeServiceNode(name, addr)
					rsp, err := e.cli.Get(cctx, node.buildPrefix(), etcdV3.WithPrefix())
					if err != nil {
						log.Errorf("get keys for prefix %s error:%s", node.buildPrefix(), err.Error())
						continue
					}
					if len(rsp.Kvs) == 0 {
						//*如果所有相关键都被删除，取消协程
						cancel()
						log.Infof("node watcher for %s exited due to all keys deletion", node.Name)
						return
					}
				}
			}
		case <-ctx.Done():
			//*TODO log
			return
		}
	}

}

// *关闭etcd服务发现器
func (e *etcdResolver) Stop() {
	log.Infoln("resolver stop")
	e.cancel()
}

var EResolver *etcdResolver

// *注册器初始化
func EtcdResolverInit() {
	envEtcdAddr := os.Getenv("DISCOVERY_HOST")
	log.Infof("DISCOVERY_HOST: %s", envEtcdAddr)
	EResolver = &etcdResolver{
		mr:            make(map[string]resolver.Resolver),
		dialTimeout:   time.Second * 3,
		targetNodeSet: make(map[string]*Node),
		serviceNodes:  make(map[string]map[string]*Node),
	}
	if len(envEtcdAddr) > 0 {
		EResolver.etcdAddrs = strings.Split(envEtcdAddr, ";")
	}
}
