package etcdservice

import (
	"context"
	"log"
	"net/url"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/resolver"
)

//*全局etcd变量
var cli *clientv3.Client

//*etcdResolver 解析struct
type etcdResolver struct {
	rawAddr string
	cc      resolver.ClientConn
}

//*新建一个自定义名称解析器
func NewResolver(etcdAddr string) resolver.Builder {
	return &etcdResolver{rawAddr: etcdAddr}
}

//*Build 构建etcd client
func (r *etcdResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	var err error
	if cli == nil {
		cli, err = clientv3.New(clientv3.Config{
			//*从 r.rawAddr 中解析出 etcd 的地址列表，使用 ; 分隔
			Endpoints:   strings.Split(r.rawAddr, ";"),
			//*设置连接超时时间为 15 秒
			DialTimeout: 15 * time.Second,
		})
		if err != nil {
			return nil, err
		}
	}

	r.cc = cc

	//*解析 target.URL 获取 scheme 和 endpoint
	parsedURL, err := url.Parse(target.URL.Path)
	if err != nil {
		return nil, err
	}

	//*监听路径
	watchKey := "/" + parsedURL.Scheme + "/" + parsedURL.Host + "/"
	go r.watch(watchKey)

	return r, nil
}

//*返回自定义名称解析器的名称
func (r etcdResolver) Scheme() string {
	return "etcd"
}

//*于立即触发解析逻辑
func (r etcdResolver) ResolveNow(rn resolver.ResolveNowOptions) {
	log.Println("ResolveNow")
}

//*关闭解析器
func (r etcdResolver) Close() {
	log.Println("Close")
}

//*watch 监听resolve列表变化
func (r *etcdResolver) watch(keyPrefix string) {
	var addrList []resolver.Address

	getResp, err := cli.Get(context.Background(), keyPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Println(err)
	} else {
		for i := range getResp.Kvs {
			addrList = append(addrList, resolver.Address{Addr: strings.TrimPrefix(string(getResp.Kvs[i].Key), keyPrefix)})
		}
	}

	//*新版本etcd去除了NewAddress方法 以UpdateState代替
	r.cc.UpdateState(resolver.State{Addresses: addrList})

	rch := cli.Watch(context.Background(), keyPrefix, clientv3.WithPrefix())
	for n := range rch {
		for _, ev := range n.Events {
			addr := strings.TrimPrefix(string(ev.Kv.Key), keyPrefix)
			switch ev.Type {
			case mvccpb.PUT:
				if !exist(addrList, addr) {
					addrList = append(addrList, resolver.Address{Addr: addr})
					r.cc.UpdateState(resolver.State{Addresses: addrList})
				}
			case mvccpb.DELETE:
				if s, ok := remove(addrList, addr); ok {
					addrList = s
					r.cc.UpdateState(resolver.State{Addresses: addrList})
				}
			}
			log.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}

//*判断addr是否存在resolver.address
func exist(l []resolver.Address, addr string) bool {
	for i := range l {
		if l[i].Addr == addr {
			return true
		}
	}
	return false
}

//*remove addr 从 resolver.Address 列表移除
func remove(s []resolver.Address, addr string) ([]resolver.Address, bool) {
	for i := range s {
		if s[i].Addr == addr {
			s[i] = s[len(s)-1]
			return s[:len(s)-1], true
		}
	}
	return nil, false
}
