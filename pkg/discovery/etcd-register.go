// etcdRegister.go

package discovery

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/golang/glog"
	etcdV3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

type etcdRegister struct {
	//*注册节点set
	nodeSet map[string]*Node
	//*etcd句柄
	cli *etcdV3.Client
	//*etcd服务地址
	etcdAddrs []string
	//*连接超时时间
	dialTimeout time.Duration
	//*etcd租约id
	etcdLeaseId etcdV3.LeaseID
	//*注册节点过期时间
	ttl int64
	//*取消函数，用去结束注册任务
	cancel context.CancelFunc
	once   sync.Once
}

// *新增注册的服务节点
func (e *etcdRegister) addServiceNode(node *Node) {
	e.nodeSet[node.buildKey()] = node
	//*新增注册节点的时候，开始执行注册任务
	e.once.Do(
		func() {
			var ctx context.Context
			ctx, e.cancel = context.WithCancel(context.Background())
			e.start(ctx)
		})
}

// *开始注册任务
func (e *etcdRegister) start(ctx context.Context) {
	if len(e.etcdAddrs) == 0 {
		panic("discovery should call SetDiscoveryAddress or set env DISCOVERY_HOST")
	}

	//*连接etcd
	var err error
	//*新建etcd客户端
	e.cli, err = etcdV3.New(etcdV3.Config{
		Endpoints:   e.etcdAddrs,
		DialTimeout: e.dialTimeout,
	})

	if err != nil {
		panic(err)
	}

	//*创建租约
	cctx, cancel := context.WithTimeout(ctx, e.dialTimeout)
	rsp, err := e.cli.Grant(cctx, e.ttl)
	if err != nil {
		panic(err)
	}
	cancel()
	//*租约的id
	e.etcdLeaseId = rsp.ID

	//*KeepAlive 会定期向 etcd 发送保活请求，以确保租约不会过期
	//*每次保活成功时，kc 通道会接收到一个 *LeaseKeepAliveResponse 对象（kaRsp）
	kc, err := e.cli.KeepAlive(ctx, rsp.ID)
	if err != nil {
		log.Errorf("etcd keepalive error:%s", err.Error())
	}

	go func() {
		defer log.Infoln("register goroutine exited") 
		for {
			select {
			case kaRsp, ok := <-kc:
				if !ok {
					log.Errorln("etcd keepalive channel closed")
					return 
				}
				if kaRsp != nil {
					e.register(ctx)
				}
			case <-ctx.Done():
				log.Infoln("register exit")
				return
			}
		}
	}()
}

//*注册节点
func (e *etcdRegister) register(ctx context.Context) {
	//*遍历所有的服务节点进行注册
	for _, n := range e.nodeSet {
		value, err := json.Marshal(n)
		if err != nil {
			log.Errorf("json marshal node:%s error:%s", n.Name, err.Error())
			continue
		}
		//*使用租约id注册
		cctx, cancel := context.WithTimeout(ctx, e.dialTimeout)
		_, err = e.cli.Put(cctx, n.buildKey(), string(value), etcdV3.WithLease(e.etcdLeaseId))
		cancel()

		if err != nil {
			log.Errorf("put %s:%s to etcd with lease id %d error:%s", n.buildKey(), string(value), e.etcdLeaseId, err.Error())
			continue
		}

	}
}

//*停止注册任务
func (e *etcdRegister) stop() {
	log.Infoln("register stop")
	//*退出注册任务
	e.cancel()

	//*清理注册信息
	for _, n := range e.nodeSet {
		value, err := json.Marshal(n)
		if err != nil {
			log.Errorf("json marshal node:%s error:%s", n.Name, err.Error())
			continue
		}
		cctx, cancel := context.WithTimeout(context.Background(), e.dialTimeout)
		_, _ = e.cli.Delete(cctx, n.buildKey())
		cancel()
		log.Infof("delete %s:%s from etcd", n.buildKey(), string(value))
	}
}

var eResolver *etcdResolver

//*注册器初始化
func etcdRegisterInit() {
	envEtcdAddr := os.Getenv("DISCOVERY_HOST")
	eResolver = &etcdResolver{
		mr:            make(map[string]resolver.Resolver),
		dialTimeout:   time.Second * 3,
		targetNodeSet: make(map[string]*Node),
		serviceNodes:  make(map[string]map[string]*Node),
	}
	if len(envEtcdAddr) > 0 {
		eResolver.etcdAddrs = strings.Split(envEtcdAddr, ";")
	}
}
