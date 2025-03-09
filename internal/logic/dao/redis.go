package dao

import (
	"context"
	"fmt"
)

const (
	_prefixMidServer    = "mid_%d" //*存储用户 ID（mid）与键（key）和服务器的映射关系
	_prefixKeyServer    = "key_%s" //*存储键（key）与服务器的映射关系
	_prefixServerOnline = "ol_%s"  //*存储服务器的在线状态信息
)

// *用于生成 Redis 的键名
func keyMidServer(mid int64) string {
	return fmt.Sprintf(_prefixMidServer, mid)
}

// *用于生成 Redis 的键名
func keyKeyServer(key string) string {
	return fmt.Sprintf(_prefixKeyServer, key)
}

// *用于生成 Redis 的键名
func keyServerOnline(key string) string {
	return fmt.Sprintf(_prefixServerOnline, key)
}

// *通过发送 PING 命令检查 Redis 连接是否正常
func (d *Dao) pingRedis(c context.Context) (err error) {
	conn := d.redis.Get()
	_, err = conn.Do("SET", "PING", "PONG")
	conn.Close()
	return
}
