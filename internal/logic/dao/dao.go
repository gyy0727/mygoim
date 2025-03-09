package dao

import (
		"github.com/gomodule/redigo/redis"
	"github.com/gyy0727/mygoim/internal/logic/conf"
	kafka "gopkg.in/Shopify/sarama.v1"
)

type Dao struct {
	c           *conf.Config //*项目的配置对象，包含 Redis 和 Kafka 的配置信息
	kafkaPub    kafka.SyncProducer //*Kafka 的生产者对象，用于向 Kafka 发送消息
	redis       *redis.Pool //*Redis 连接池，用于与 Redis 交互
	redisExpire int32 //*Redis 数据的过期时间(秒)
}


