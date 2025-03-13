package conf

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
	//*用于解析 TOML 配置文件
	"github.com/BurntSushi/toml"
	xtime "github.com/gyy0727/mygoim/pkg/time"
)

var (
	confPath  string  //*配置文件路径
	region    string  //*存储应用程序所在的区域
	zone      string  //*存储应用程序所在的可用区（Zone）
	deployEnv string  //*存储应用程序的部署环境
	host      string  //*存储应用程序所在的主机名
	addrs     string  //*存储应用程序的公共 IP 地址列表
	weight    int64   //*存储应用程序的负载均衡权重
	offline   bool    //*标识应用程序是否处于离线状态
	debug     bool    //*标识应用程序是否启用调试模式
	Conf      *Config //*指向 Config 结构体的指针，用于存储解析后的配置
)

func init() {
	var (
		defHost, _    = os.Hostname()                                 //*默认主机名
		defAddrs      = os.Getenv("ADDRS")                            //*默认的addr
		defWeight, _  = strconv.ParseInt(os.Getenv("WEIGHT"), 10, 32) //*默认的权重
		defOffline, _ = strconv.ParseBool(os.Getenv("OFFLINE"))       //*是否在线
		defDebug, _   = strconv.ParseBool(os.Getenv("DEBUG"))         //*是否开启调试
	)
	flag.StringVar(&confPath, "conf", "comet-example.toml", "default config path.")
	flag.StringVar(&region, "region", os.Getenv("REGION"), "avaliable region. or use REGION env variable, value: sh etc.")
	flag.StringVar(&zone, "zone", os.Getenv("ZONE"), "avaliable zone. or use ZONE env variable, value: sh001/sh002 etc.")
	flag.StringVar(&deployEnv, "deploy.env", os.Getenv("DEPLOY_ENV"), "deploy env. or use DEPLOY_ENV env variable, value: dev/fat1/uat/pre/prod etc.")
	flag.StringVar(&host, "host", defHost, "machine hostname. or use default machine hostname.")
	flag.StringVar(&addrs, "addrs", defAddrs, "server public ip addrs. or use ADDRS env variable, value: 127.0.0.1 etc.")
	flag.Int64Var(&weight, "weight", defWeight, "load balancing weight, or use WEIGHT env variable, value: 10 etc.")
	flag.BoolVar(&offline, "offline", defOffline, "server offline. or use OFFLINE env variable, value: true/false etc.")
	flag.BoolVar(&debug, "debug", defDebug, "server debug. or use DEBUG env variable, value: true/false etc.")
}

func Init() (err error) {
	Conf = Default() //*初始化为默认
	_, err = toml.DecodeFile(confPath, &Conf)
	return
}

func Default() *Config {
	return &Config{
		Debug: debug,
		Env:   &Env{Region: region, Zone: zone, DeployEnv: deployEnv, Host: host, Weight: weight, Addrs: strings.Split(addrs, ","), Offline: offline},
		Etcd: &EtcdConfig{
			Endpoints:   []string{"http://127.0.0.1:2379"}, //*默认 etcd 地址
			DialTimeout: 5 * time.Second,                   //*默认连接超时时间
			Username:    "",                                //*默认无用户名
			Password:    "",                                //*默认无密码
			Prefix:      "/myapp/dev",                      //*默认键前缀
		},
		RPCClient: &RPCClient{
			Dial:    xtime.Duration(time.Second),
			Timeout: xtime.Duration(time.Second),
		},
		RPCServer: &RPCServer{
			Network:           "tcp",
			Addr:              ":3109",
			Timeout:           xtime.Duration(time.Second),
			IdleTimeout:       xtime.Duration(time.Second * 60),
			MaxLifeTime:       xtime.Duration(time.Hour * 2),
			ForceCloseWait:    xtime.Duration(time.Second * 20),
			KeepAliveInterval: xtime.Duration(time.Second * 60),
			KeepAliveTimeout:  xtime.Duration(time.Second * 20),
		},
		TCP: &TCP{
			Bind:         []string{":3101"},
			Sndbuf:       4096,
			Rcvbuf:       4096,
			KeepAlive:    false,
			Reader:       32,
			ReadBuf:      1024,
			ReadBufSize:  8192,
			Writer:       32,
			WriteBuf:     1024,
			WriteBufSize: 8192,
		},
		Websocket: &Websocket{
			Bind: []string{":3102"},
		},
		Protocol: &Protocol{
			Timer:            32,
			TimerSize:        2048,
			CliProto:         5,
			SvrProto:         10,
			HandshakeTimeout: xtime.Duration(time.Second * 5),
		},
		Bucket: &Bucket{
			Size:          32,
			Channel:       1024,
			Room:          1024,
			RoutineAmount: 32,
			RoutineSize:   1024,
		},
	}
}

// *Comet配置
type Config struct {
	Debug     bool        // *是否开启调试模式
	Env       *Env        // *环境相关的配置
	Etcd      *EtcdConfig // *Etcd服务发现配置
	TCP       *TCP        // *TCP连接配置
	Websocket *Websocket  // *WebSocket连接配置
	Protocol  *Protocol   // *协议相关的配置
	Bucket    *Bucket     // *连接桶的配置
	RPCClient *RPCClient  // *RPC客户端配置
	RPCServer *RPCServer  // *RPC服务端配置
	Whitelist *Whitelist  // *白名单配置
}

// *Etcd服务发现配置
type EtcdConfig struct {
	Endpoints   []string      // *Etcd集群地址列表
	DialTimeout time.Duration // *连接超时时间
	Username    string        // *用户名（可选，用于认证）
	Password    string        // *密码（可选，用于认证）
	Prefix      string        // *键前缀（可选，用于命名空间隔离）
}

// *环境相关的配置
type Env struct {
	Region    string   // *区域（如 "asia-east"）
	Zone      string   // *可用区（如 "zone1"）
	DeployEnv string   // *部署环境（如 "prod" 或 "dev"）
	Host      string   // *主机名（如服务器的hostname）
	Weight    int64    // *负载权重（用于负载均衡）
	Offline   bool     // *是否标记为离线状态
	Addrs     []string // *服务器地址列表（如IP:端口）
}

// *RPC客户端配置
type RPCClient struct {
	Dial    xtime.Duration // *连接超时时间
	Timeout xtime.Duration // *请求超时时间
}

// *RPC服务端配置
type RPCServer struct {
	Network           string         // *网络协议（如 "tcp"）
	Addr              string         // *监听地址（如 "0.0.0.0:8080"）
	Timeout           xtime.Duration // *请求处理超时时间
	IdleTimeout       xtime.Duration // *空闲连接超时时间
	MaxLifeTime       xtime.Duration // *连接的最大生命周期
	ForceCloseWait    xtime.Duration // *强制关闭前的等待时间
	KeepAliveInterval xtime.Duration // *心跳检测间隔时间
	KeepAliveTimeout  xtime.Duration // *心跳超时时间
}

// *TCP连接配置
type TCP struct {
	Bind         []string // *绑定地址列表（如 ["0.0.0.0:8080"]）
	Sndbuf       int      // *发送缓冲区大小（单位：字节）
	Rcvbuf       int      // *接收缓冲区大小（单位：字节）
	KeepAlive    bool     // *是否启用TCP KeepAlive
	Reader       int      // *读取goroutine数量
	ReadBuf      int      // *读取缓冲区大小（单位：字节）
	ReadBufSize  int      // *读取缓冲区容量（单位：字节）
	Writer       int      // *写入goroutine数量
	WriteBuf     int      // *写入缓冲区大小（单位：字节）
	WriteBufSize int      // *写入缓冲区容量（单位：字节）
}

// *WebSocket连接配置
type Websocket struct {
	Bind        []string // *绑定地址列表（如 ["0.0.0.0:8080"]）
	TLSOpen     bool     // *是否启用TLS
	TLSBind     []string // *TLS绑定地址列表（如 ["0.0.0.0:443"]）
	CertFile    string   // *TLS证书文件路径
	PrivateFile string   // *TLS私钥文件路径
}

// *协议相关的配置
type Protocol struct {
	Timer            int            // *定时器数量（用于协议处理）
	TimerSize        int            // *定时器容量（每个定时器的缓冲大小）
	SvrProto         int            // *服务器协议版本
	CliProto         int            // *客户端协议版本
	HandshakeTimeout xtime.Duration // *握手超时时间
}

// *连接桶的配置
type Bucket struct {
	Size          int    // *桶的最大容量，表示可以存储的总连接数
	Channel       int    // *用户连接映射的初始容量（chs map 的预分配大小）
	Room          int    // *房间映射的初始容量（rooms map 的预分配大小）
	RoutineAmount uint64 // *用于广播消息的goroutine数量
	RoutineSize   int    // *每个广播通道的缓冲容量（routines中每个chan的缓冲大小）
}

// *白名单配置
type Whitelist struct {
	Whitelist []int64 // *白名单用户ID列表
	WhiteLog  string  // *白名单日志文件路径
}
