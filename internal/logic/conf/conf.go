package conf

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	xtime "github.com/gyy0727/mygoim/pkg/time"
)

var (
	confPath  string  //*配置文件的路径
	deployEnv string  //*部署环境（如 dev 表示开发环境，prod 表示生产环境）
	host      string  //*当前主机的主机名
	weight    int64   //*负载均衡权重
	Conf      *Config //*全局的配置对象，类型为 *Config
)

func init() {
	var (
		defHost, _   = os.Hostname()
		defWeight, _ = strconv.ParseInt(os.Getenv("WEIGHT"), 10, 32)
	)
	flag.StringVar(&confPath, "conf", "logic.toml", "default config path")
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
		Env: &Env{DeployEnv: deployEnv, Host: host, Weight: weight},
		Discovery: &EtcdConfig{
			Endpoints:   []string{"http://127.0.0.1:2379"}, // 默认的 etcd 集群地址
			DialTimeout: 5,                                 // 默认连接超时时间为 5 秒
			Username:    "",                                // 默认无用户名
			Password:    "",                                // 默认无密码
		},
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
	Env        *Env        //*环境相关的配置
	Discovery  *EtcdConfig //*服务发现相关的配置
	RPCClient  *RPCClient  //*RPC 客户端配置
	RPCServer  *RPCServer  //*RPC 服务端配置
	HTTPServer *HTTPServer //*HTTP 服务端配置
	Kafka      *Kafka      //*Kafka 相关的配置
	Redis      *Redis      //*Redis 相关的配置
	// Node       *Node               //*节点相关的配置
	Backoff *Backoff            //*重试策略相关的配置
	Regions map[string][]string //*区域映射配
}

type EtcdConfig struct {
	Endpoints   []string //*etcd 集群的端点列表
	DialTimeout int      //*连接 etcd 的超时时间（单位：秒）
	Username    string   //*etcd 用户名
	Password    string   //*etcd 密码
}

// *用于存储环境相关的配置
type Env struct {
	DeployEnv string //*部署环境
	Host      string //*主机名
	Weight    int64  //*负载均衡权重
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
	Topic   string   //*表示Kafka消息的主题名称
	Brokers []string //*存储Kafka集群的Broker地址列表
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
