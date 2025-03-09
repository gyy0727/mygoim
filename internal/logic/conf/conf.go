package conf

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/bilibili/discovery/naming"
	xtime "github.com/gyy0727/mygoim/pkg/time"

	"github.com/BurntSushi/toml"
)

var (
	confPath  string  //*配置文件的路径
	region    string  //*当前服务的区域（如 sh 表示上海）
	zone      string  //* 当前服务的可用区（如 sh001 表示上海 001 区）
	deployEnv string  //*部署环境（如 dev 表示开发环境，prod 表示生产环境）
	host      string  //*当前主机的主机名
	weight    int64   //* 负载均衡权重
	Conf      *Config //*全局的配置对象，类型为 *Config
)

func init() {
	var (
		defHost, _   = os.Hostname()
		defWeight, _ = strconv.ParseInt(os.Getenv("WEIGHT"), 10, 32)
	)
	flag.StringVar(&confPath, "conf", "logic-example.toml", "default config path")
	flag.StringVar(&region, "region", os.Getenv("REGION"), "avaliable region. or use REGION env variable, value: sh etc.")
	flag.StringVar(&zone, "zone", os.Getenv("ZONE"), "avaliable zone. or use ZONE env variable, value: sh001/sh002 etc.")
	flag.StringVar(&deployEnv, "deploy.env", os.Getenv("DEPLOY_ENV"), "deploy env. or use DEPLOY_ENV env variable, value: dev/fat1/uat/pre/prod etc.")
	flag.StringVar(&host, "host", defHost, "machine hostname. or use default machine hostname.")
	flag.Int64Var(&weight, "weight", defWeight, "load balancing weight, or use WEIGHT env variable, value: 10 etc.")
}

// *解析配置文件
func Init() (err error) {
	Conf = Default()
	_, err = toml.DecodeFile(confPath, &Conf)
	return
}

// *Default 函数返回一个默认的 Config 对象
func Default() *Config {
	return &Config{
		Env:       &Env{Region: region, Zone: zone, DeployEnv: deployEnv, Host: host, Weight: weight},
		Discovery: &naming.Config{Region: region, Zone: zone, Env: deployEnv, Host: host},
		HTTPServer: &HTTPServer{
			Network:      "tcp",
			Addr:         "3111",
			ReadTimeout:  xtime.Duration(time.Second),
			WriteTimeout: xtime.Duration(time.Second),
		},
		RPCClient: &RPCClient{Dial: xtime.Duration(time.Second), Timeout: xtime.Duration(time.Second)},
		RPCServer: &RPCServer{
			Network:           "tcp",
			Addr:              "3119",
			Timeout:           xtime.Duration(time.Second),
			IdleTimeout:       xtime.Duration(time.Second * 60),
			MaxLifeTime:       xtime.Duration(time.Hour * 2),
			ForceCloseWait:    xtime.Duration(time.Second * 20),
			KeepAliveInterval: xtime.Duration(time.Second * 60),
			KeepAliveTimeout:  xtime.Duration(time.Second * 20),
		},
		Backoff: &Backoff{MaxDelay: 300, BaseDelay: 3, Factor: 1.8, Jitter: 1.3},
	}
}

type Config struct {
	Env        *Env                //*环境相关的配置
	Discovery  *naming.Config      //*服务发现相关的配置
	RPCClient  *RPCClient          //*RPC 客户端配置
	RPCServer  *RPCServer          //*RPC 服务端配置
	HTTPServer *HTTPServer         //*HTTP 服务端配置
	Kafka      *Kafka              //*Kafka 相关的配置
	Redis      *Redis              //*Redis 相关的配置
	Node       *Node               //*节点相关的配置
	Backoff    *Backoff            //*重试策略相关的配置
	Regions    map[string][]string //*区域映射配
}

// *用于存储环境相关的配置
type Env struct {
	Region    string //*区域
	Zone      string //*可用区
	DeployEnv string //*部署环境
	Host      string //*主机名
	Weight    int64  //*负载均衡权重
}

// *结构体用于存储节点相关的配置
type Node struct {
	DefaultDomain string         //*默认域名
	HostDomain    string         //*主机域名
	TCPPort       int            //*tcp端口
	WSPort        int            //*websocket端口
	WSSPort       int            //*安全websocket端口(包含tls验证)
	HeartbeatMax  int            //*心跳超时最大次数
	Heartbeat     xtime.Duration //*心跳超时间隔
	RegionWeight  float64        //*区域权重
}

// *重试策略的配置
type Backoff struct {
	MaxDelay  int32   //*最大延迟
	BaseDelay int32   //*基础延迟
	Factor    float32 //*因子
	Jitter    float32 //*抖动
}

// *redis相关配置
type Redis struct {
	Network      string         //*网络类型
	Addr         string         //*服务器地址
	Auth         string         //*认证消息
	Active       int            //*活跃连接数
	Idle         int            //*空闲连接数
	DialTimeout  xtime.Duration //*连接超时
	ReadTimeout  xtime.Duration //*读取超时
	WriteTimeout xtime.Duration //*写入超时
	IdleTimeout  xtime.Duration //*空闲超时
	Expire       xtime.Duration //*过期时间
}

// *kafka相关配置
type Kafka struct {
	Topic   string
	Brokers []string
}

// *RPC客户端配置
type RPCClient struct {
	Dial    xtime.Duration //*连接超时
	Timeout xtime.Duration //*请求超时
}

// *rpc服务端配置
type RPCServer struct {
	Network           string         //*网络类型
	Addr              string         //*服务器地址
	Timeout           xtime.Duration //*超时时间
	IdleTimeout       xtime.Duration //*最大空闲时间
	MaxLifeTime       xtime.Duration //*最大生命周期
	ForceCloseWait    xtime.Duration //*强制关闭等待时间
	KeepAliveInterval xtime.Duration //*保活间隔
	KeepAliveTimeout  xtime.Duration //*保活超时
}

// *http服务器配置
type HTTPServer struct {
	Network      string         //*网络类型
	Addr         string         //*服务器地址
	ReadTimeout  xtime.Duration //*读超时
	WriteTimeout xtime.Duration //*写超时
}
