package etcdservice

import (
	"context"
	"log"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
)

const Schema = "grpcEtcd" // 统一 Schema


// Register 注册地址到 ETCD 组件中，使用 ; 分割
func Register(etcdAddr, name string, addr string, ttl int64) error {
	var err error

	// 如果全局客户端未初始化，则创建
	if cli == nil {
		cli, err = clientv3.New(clientv3.Config{
			Endpoints:   strings.Split(etcdAddr, ";"),
			DialTimeout: 15 * time.Second,
		})
		if err != nil {
			log.Printf("connect to etcd err: %s", err)
			return err
		}
	}

	// 创建租约并注册服务
	err = withAlive(name, addr, ttl)
	if err != nil {
		log.Printf("register service err: %s", err)
		return err
	}

	// 定时续约
	ticker := time.NewTicker(time.Second * time.Duration(ttl))
	go func() {
		for range ticker.C {
			getResp, err := cli.Get(context.Background(), "/"+Schema+"/"+name+"/"+addr)
			if err != nil {
				log.Printf("get from etcd err: %s", err)
				continue
			}

			// 如果键不存在，则重新注册
			if getResp.Count == 0 {
				err = withAlive(name, addr, ttl)
				if err != nil {
					log.Printf("re-register service err: %s", err)
				}
			}
		}
	}()

	return nil
}

// withAlive 创建租约并注册服务
func withAlive(name string, addr string, ttl int64) error {
	// 创建租约
	leaseResp, err := cli.Grant(context.Background(), ttl)
	if err != nil {
		return err
	}

	// 注册服务
	key := "/" + Schema + "/" + name + "/" + addr
	_, err = cli.Put(context.Background(), key, addr, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		log.Printf("put to etcd err: %s", err)
		return err
	}

	// 保持租约
	ch, err := cli.KeepAlive(context.Background(), leaseResp.ID)
	if err != nil {
		log.Printf("keep alive err: %s", err)
		return err
	}

	// 清空 keep alive 返回的 channel
	go func() {
		for range ch {
			// 忽略 keep alive 响应
		}
	}()

	return nil
}

// UnRegister 从 ETCD 中注销服务
func UnRegister(name string, addr string) {
	if cli != nil {
		key := "/" + Schema + "/" + name + "/" + addr
		_, err := cli.Delete(context.Background(), key)
		if err != nil {
			log.Printf("delete from etcd err: %s", err)
		} else {
			log.Printf("unregister service: %s", key)
		}
	}
}
