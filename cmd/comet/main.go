package main

import (
	"context"       //* 用于上下文控制
	"encoding/json" //* 用于 JSON 编码服务实例信息
	"flag"          //* 命令行参数解析
	"fmt"           //* 格式化输出
	"math/rand"     //* 随机数生成
	"net"           //* 网络相关操作，如提取端口
	"os"            //* 操作系统相关（信号、退出）
	"os/signal"     //* 处理系统信号
	"runtime"       //* 控制 Go 运行时
	"strconv"       //* 数字与字符串转换
	"strings"       //* 字符串处理
	"syscall"       //* 系统调用常量
	"time"          //* 时间相关操作

	"github.com/Terry-Mao/goim/pkg/ip"              //* 获取内网 IP
	log "github.com/golang/glog"                    //* 日志记录
	"github.com/gyy0727/mygoim/internal/comet"      //* comet 服务器相关
	"github.com/gyy0727/mygoim/internal/comet/conf" //* 配置管理
	"github.com/gyy0727/mygoim/internal/comet/grpc" //* gRPC 服务

	//* 数据模型定义
	clientv3 "go.etcd.io/etcd/client/v3" //* etcd 客户端
)

const (
	ver   = "2.0.0"      //* 程序版本号
	appid = "goim.comet" //* 应用 ID
)

func main() {
	flag.Parse()
	if err := conf.Init(); err != nil {
		panic(err)
	}
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	println(conf.Conf.Debug)
	log.Infof("goim-comet [version: %s env: %+v] start", ver, conf.Conf.Env)

	//* 初始化 comet 服务器
	srv := comet.NewServer(conf.Conf)
	if err := comet.InitWhitelist(conf.Conf.Whitelist); err != nil {
		panic(err)
	}
	if err := comet.InitTCP(srv, conf.Conf.TCP.Bind, runtime.NumCPU()); err != nil {
		panic(err)
	}
	if err := comet.InitWebsocket(srv, conf.Conf.Websocket.Bind, runtime.NumCPU()); err != nil {
		panic(err)
	}
	if conf.Conf.Websocket.TLSOpen {
		if err := comet.InitWebsocketWithTLS(srv, conf.Conf.Websocket.TLSBind, conf.Conf.Websocket.CertFile, conf.Conf.Websocket.PrivateFile, runtime.NumCPU()); err != nil {
			panic(err)
		}
	}

	//* 初始化 gRPC 服务
	rpcSrv := grpc.New(conf.Conf.RPCServer, srv)

	//* 使用 etcd 进行服务注册
	etcdEndpoints := []string{"127.0.0.1:2379"} //* etcd 端点，根据实际情况修改，建议将其写入配置文件中
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Errorf("连接 etcd 失败: %v", err)
		panic(err)
	}
	//* 注册服务，返回一个用于注销的 cancel 函数
	cancel := registerEtcd(etcdCli, srv)

	//* 处理系统信号，实现优雅退出
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Infof("goim-comet 收到信号 %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			if cancel != nil {
				cancel() //* 注销服务：撤销 etcd 租约
			}
			rpcSrv.GracefulStop()
			srv.Close()
			log.Infof("goim-comet [version: %s] exit", ver)
			log.Flush()
			return
		case syscall.SIGHUP:
			//* 收到 SIGHUP 信号，可在此处添加配置重载代码
		default:
			return
		}
	}
}

// registerEtcd 使用 etcd 进行服务注册，并定期更新服务元数据
func registerEtcd(cli *clientv3.Client, srv *comet.Server) context.CancelFunc {
	env := conf.Conf.Env
	addr := ip.InternalIP()
	_, port, _ := net.SplitHostPort(conf.Conf.RPCServer.Addr)

	//* 构造服务实例信息，使用 JSON 格式保存
	instance := map[string]string{
		"region":   env.Region,                        //* 所在区域
		"zone":     env.Zone,                          //* 可用区
		"env":      env.DeployEnv,                     //* 部署环境
		"hostname": env.Host,                          //* 主机名
		"appid":    appid,                             //* 应用 ID
		"addr":     "grpc://" + addr + ":" + port,     //* 服务地址
		"weight":   strconv.FormatInt(env.Weight, 10), //* 服务权重
		"offline":  strconv.FormatBool(env.Offline),   //* 是否下线
		"addrs":    strings.Join(env.Addrs, ","),      //* 其他地址
	}

	//* etcd 键的命名规则，此处使用 /services/{appid}/{hostname}，可根据实际需求调整
	key := fmt.Sprintf("/services/%s/%s", appid, env.Host)
	val, err := json.Marshal(instance)
	if err != nil {
		log.Errorf("服务实例 JSON 编码失败: %v", err)
		panic(err)
	}

	//* 创建 etcd 租约，TTL 设置为 10 秒
	leaseResp, err := cli.Grant(context.Background(), 10)
	if err != nil {
		log.Errorf("创建 etcd 租约失败: %v", err)
		panic(err)
	}

	//* 将服务实例信息注册到 etcd，附带租约
	_, err = cli.Put(context.Background(), key, string(val), clientv3.WithLease(leaseResp.ID))
	if err != nil {
		log.Errorf("在 etcd 中注册服务失败: %v", err)
		panic(err)
	}

	//* 启动租约续约，保持服务注册状态
	kaCh, err := cli.KeepAlive(context.Background(), leaseResp.ID)
	if err != nil {
		log.Errorf("启动 etcd 租约续约失败: %v", err)
		panic(err)
	}

	//* 使用一个上下文用于控制注销时退出续约和更新
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				//* 注销服务时撤销租约
				cli.Revoke(context.Background(), leaseResp.ID)
				return
			case ka, ok := <-kaCh:
				if !ok {
					log.Errorf("etcd 续约通道已关闭")
					return
				}
				log.Infof("收到 etcd 租约续约响应: %v", ka)
			case <-time.After(10 * time.Second):
				//* 定时更新动态元数据，如连接数和 IP 数量
				var conns int
				ips := make(map[string]struct{})
				for _, bucket := range srv.Buckets() {
					for ip := range bucket.IPCount() {
						ips[ip] = struct{}{}
					}
					conns += bucket.ChannelCount()
				}
				instance["connCount"] = fmt.Sprintf("%d", conns)
				instance["ipCount"] = fmt.Sprintf("%d", len(ips))
				newVal, err := json.Marshal(instance)
				if err != nil {
					log.Errorf("服务实例 JSON 编码失败: %v", err)
					continue
				}
				_, err = cli.Put(context.Background(), key, string(newVal), clientv3.WithLease(leaseResp.ID))
				if err != nil {
					log.Errorf("更新 etcd 中服务元数据失败: %v", err)
				}
			}
		}
	}()
	return cancel
}
