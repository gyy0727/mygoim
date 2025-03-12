package dao

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gyy0727/mygoim/internal/logic/conf"
	kafka "gopkg.in/Shopify/sarama.v1"
)

type Dao struct {
	c           *conf.Config       //*项目的配置对象，包含 Redis 和 Kafka 的配置信息
	kafkaPub    kafka.SyncProducer //*Kafka 的生产者对象，用于向 Kafka 发送消息
	redis       *redis.Pool        //*Redis 连接池，用于与 Redis 交互
	redisExpire int32              //*Redis 数据的过期时间(秒)
}

// *新建一个数据访问对象（Dao）的实例
func New(c *conf.Config) *Dao {
	d := &Dao{
		c:           c,
		kafkaPub:    newKafkaPub(c.Kafka),
		redis:       newRedis(c.Redis),
		redisExpire: int32(time.Duration(c.Redis.Expire) / time.Second),
	}
	return d
}

// *新建一个kafka客户端,生产者
func newKafkaPub(c *conf.Kafka) kafka.SyncProducer {
	kc := kafka.NewConfig()
	kc.Producer.RequiredAcks = kafka.WaitForAll //*这意味着生产者会等待所有副本（包括 Leader 和所有 Follower）都确认消息已写入后，才认为消息发送成功
	kc.Producer.Retry.Max = 10                  //*设置生产者的最大重试次数为 10
	kc.Producer.Return.Successes = true         //*这意味着生产者会返回成功发送的消息的元数据
	pub, err := kafka.NewSyncProducer(c.Brokers, kc)
	if err != nil {
		panic(err)
	}
	return pub
}

// *新建一个redis客户端
func newRedis(c *conf.Redis) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     c.Idle,                       //*最大空闲连接数
		MaxActive:   c.Active,                     //*最大活跃连接数
		IdleTimeout: time.Duration(c.IdleTimeout), //*空闲连接超时时间
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial(c.Network, c.Addr,
				redis.DialConnectTimeout(time.Duration(c.DialTimeout)), //*设置连接超时时间
				redis.DialReadTimeout(time.Duration(c.ReadTimeout)),    //*设置读取超时时间
				redis.DialWriteTimeout(time.Duration(c.WriteTimeout)),  //*设置写入超时时间
				redis.DialPassword(c.Auth),                             //*设置 Redis 认证密码
			)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
}

// *关闭redis连接
func (d *Dao) Close() error {
	return d.redis.Close()
}

// *检查redis的可用性
func (d *Dao) Ping(c context.Context) error {
	return d.pingRedis(c)
}
