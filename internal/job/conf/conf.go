package conf

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	xtime "github.com/gyy0727/mygoim/pkg/time"
	"go.etcd.io/etcd/clientv3"
)

var (
	confPath  string
	region    string
	zone      string
	deployEnv string
	host      string
	Conf      *Config
)

func init() {
	var (
		defHost, _ = os.Hostname()
	)
	flag.StringVar(&confPath, "conf", "job-example.toml", "default config path")
	flag.StringVar(&region, "region", os.Getenv("REGION"), "avaliable region. or use REGION env variable, value: sh etc.")
	flag.StringVar(&zone, "zone", os.Getenv("ZONE"), "avaliable zone. or use ZONE env variable, value: sh001/sh002 etc.")
	flag.StringVar(&deployEnv, "deploy.env", os.Getenv("DEPLOY_ENV"), "deploy env. or use DEPLOY_ENV env variable, value: dev/fat1/uat/pre/prod etc.")
	flag.StringVar(&host, "host", defHost, "machine hostname. or use default machine hostname.")
}

func Init() (err error) {
	Conf = Default()
	_, err = toml.DecodeFile(confPath, &Conf)
	if err != nil {
		return err
	}

	// 初始化 etcd 客户端
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   Conf.Discovery.Endpoints,
		DialTimeout: Conf.Discovery.DialTimeout,
		Username:    Conf.Discovery.Username,
		Password:    Conf.Discovery.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %v", err)
	}
	defer etcdClient.Close()

	// 将 etcd 客户端保存到全局变量或结构体中，供后续使用
	// 例如：Conf.EtcdClient = etcdClient

	return nil
}

func Default() *Config {
	return &Config{
		Env: &Env{Region: region, Zone: zone, DeployEnv: deployEnv, Host: host},
		Discovery: &EtcdConfig{
			Endpoints:   []string{"127.0.0.1:2379"}, // 默认 etcd 地址
			DialTimeout: time.Second * 5,            // 默认超时时间
		},
		Comet: &Comet{RoutineChan: 1024, RoutineSize: 32},
		Room: &Room{
			Batch:  20,
			Signal: xtime.Duration(time.Second),
			Idle:   xtime.Duration(time.Minute * 15),
		},
	}
}

type Config struct {
	Env       *Env
	Kafka     *Kafka
	Discovery *EtcdConfig // 改为 etcd 配置
	Comet     *Comet
	Room      *Room
}

type EtcdConfig struct {
	Endpoints   []string      // etcd 集群地址
	DialTimeout time.Duration // 连接超时时间
	Username    string        // 用户名（如果有认证）
	Password    string        // 密码（如果有认证）
}

type Room struct {
	Batch  int
	Signal xtime.Duration
	Idle   xtime.Duration
}

type Comet struct {
	RoutineChan int
	RoutineSize int
}

type Kafka struct {
	Topic   string
	Group   string
	Brokers []string
}

type Env struct {
	Region    string
	Zone      string
	DeployEnv string
	Host      string
}
