package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/golang/glog"
	"github.com/gomodule/redigo/redis"
	"github.com/gyy0727/mygoim/internal/logic/model"
	"github.com/zhenjl/cityhash"
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

// *添加用户与服务器的映射关系
// *mid:用户 ID
// *key:用户的唯一标识符
// *server:服务器消息
func (d *Dao) AddMapping(c context.Context, mid int64, key, server string) (err error) {
	//*获取连接
	conn := d.redis.Get()
	defer conn.Close()
	//*初始化命令计数器，用于记录需要执行的 Redis 命令数量
	//*默认值为 2，表示至少需要执行两条命令
	var n = 2
	//*如果 mid 大于 0，添加 mid 到 server 的映射,并添加过期时间
	if mid > 0 {
		if err = conn.Send("HSET", keyMidServer(mid), key, server); err != nil {
			log.Errorf("conn.Send(HSET %d,%s,%s) error(%v)", mid, server, key, err)
			return
		}
		if err = conn.Send("EXPIRE", keyMidServer(mid), d.redisExpire); err != nil {
			log.Errorf("conn.Send(EXPIRE %d,%s,%s) error(%v)", mid, key, server, err)
			return
		}
		n += 2
	}
	if err = conn.Send("SET", keyKeyServer(key), server); err != nil {
		log.Errorf("conn.Send(HSET %d,%s,%s) error(%v)", mid, server, key, err)
		return
	}
	if err = conn.Send("EXPIRE", keyKeyServer(key), d.redisExpire); err != nil {
		log.Errorf("conn.Send(EXPIRE %d,%s,%s) error(%v)", mid, key, server, err)
		return
	}
	if err = conn.Flush(); err != nil {
		log.Errorf("conn.Flush() error(%v)", err)
		return
	}
	//* 接收 Redis 管道（Pipeline）中所有命令的执行结果，并检查是否有错误
	for i := 0; i < n; i++ {
		if _, err = conn.Receive(); err != nil {
			log.Errorf("conn.Receive() error(%v)", err)
			return
		}
	}
	return
}

// *为 Redis 中的某些键设置过期时间
func (d *Dao) ExpireMapping(c context.Context, mid int64, key string) (has bool, err error) {
	conn := d.redis.Get()
	defer conn.Close()
	var n = 1
	if mid > 0 {
		if err = conn.Send("EXPIRE", keyMidServer(mid), d.redisExpire); err != nil {
			log.Errorf("conn.Send(EXPIRE %d,%s) error(%v)", mid, key, err)
			return
		}
		n++
	}
	if err = conn.Send("EXPIRE", keyKeyServer(key), d.redisExpire); err != nil {
		log.Errorf("conn.Send(EXPIRE %d,%s) error(%v)", mid, key, err)
		return
	}
	if err = conn.Flush(); err != nil {
		log.Errorf("conn.Flush() error(%v)", err)
		return
	}
	for i := 0; i < n; i++ {
		if has, err = redis.Bool(conn.Receive()); err != nil {
			log.Errorf("conn.Receive() error(%v)", err)
			return
		}
	}
	return
}

// *该函数用于删除 Redis 中的某些键或哈希字段
func (d *Dao) DelMapping(c context.Context, mid int64, key, server string) (has bool, err error) {
	//*获取redis连接
	conn := d.redis.Get()
	defer conn.Close()
	//*要执行的redis命令的数量
	n := 1
	//*删除映射
	if mid > 0 {
		if err = conn.Send("HDEL", keyMidServer(mid), key); err != nil {
			log.Errorf("conn.Send(HDEL %d,%s,%s) error(%v)", mid, key, server, err)
			return
		}
		n++
	}
	//*删除映射
	if err = conn.Send("DEL", keyKeyServer(key)); err != nil {
		log.Errorf("conn.Send(HDEL %d,%s,%s) error(%v)", mid, key, server, err)
		return
	}
	if err = conn.Flush(); err != nil {
		log.Errorf("conn.Flush() error(%v)", err)
		return
	}
	for i := 0; i < n; i++ {
		if has, err = redis.Bool(conn.Receive()); err != nil {
			log.Errorf("conn.Receive() error(%v)", err)
			return
		}
	}
	return
}

// *从 Redis 中批量获取与指定 keys 对应的值
func (d *Dao) ServersByKeys(c context.Context, keys []string) (res []string, err error) {
	conn := d.redis.Get()
	defer conn.Close()
	var args []interface{}
	for _, key := range keys {
		args = append(args, keyKeyServer(key))
	}
	if res, err = redis.Strings(conn.Do("MGET", args...)); err != nil {
		log.Errorf("conn.Do(MGET %v) error(%v)", args, err)
	}
	return
}

// *从 Redis 中批量获取与指定 mids 对应的哈希数据
func (d *Dao) KeysByMids(c context.Context, mids []int64) (ress map[string]string, olMids []int64, err error) {
	conn := d.redis.Get()
	defer conn.Close()
	ress = make(map[string]string)
	for _, mid := range mids {
		if err = conn.Send("HGETALL", keyMidServer(mid)); err != nil {
			log.Errorf("conn.Do(HGETALL %d) error(%v)", mid, err)
			return
		}
	}
	if err = conn.Flush(); err != nil {
		log.Errorf("conn.Flush() error(%v)", err)
		return
	}
	for idx := 0; idx < len(mids); idx++ {
		var (
			res map[string]string
		)
		if res, err = redis.StringMap(conn.Receive()); err != nil {
			log.Errorf("conn.Receive() error(%v)", err)
			return
		}
		if len(res) > 0 {
			olMids = append(olMids, mids[idx])
		}
		for k, v := range res {
			ress[k] = v
		}
	}
	return
}

//*将服务器的在线信息存储到 Redis 中
func (d *Dao) AddServerOnline(c context.Context, server string, online *model.Online) (err error) {
	roomsMap := map[uint32]map[string]int32{}
	for room, count := range online.RoomCount {
		rMap := roomsMap[cityhash.CityHash32([]byte(room), uint32(len(room)))%64]
		if rMap == nil {
			rMap = make(map[string]int32)
			roomsMap[cityhash.CityHash32([]byte(room), uint32(len(room)))%64] = rMap
		}
		rMap[room] = count
	}
	key := keyServerOnline(server)
	for hashKey, value := range roomsMap {
		err = d.addServerOnline(c, key, strconv.FormatInt(int64(hashKey), 10), &model.Online{RoomCount: value, Server: online.Server, Updated: online.Updated})
		if err != nil {
			return
		}
	}
	return
}

//*将单个哈希桶的在线信息存储到 Redis 中
func (d *Dao) addServerOnline(c context.Context, key string, hashKey string, online *model.Online) (err error) {
	conn := d.redis.Get()
	defer conn.Close()
	b, _ := json.Marshal(online)
	if err = conn.Send("HSET", key, hashKey, b); err != nil {
		log.Errorf("conn.Send(SET %s,%s) error(%v)", key, hashKey, err)
		return
	}
	if err = conn.Send("EXPIRE", key, d.redisExpire); err != nil {
		log.Errorf("conn.Send(EXPIRE %s) error(%v)", key, err)
		return
	}
	if err = conn.Flush(); err != nil {
		log.Errorf("conn.Flush() error(%v)", err)
		return
	}
	for i := 0; i < 2; i++ {
		if _, err = conn.Receive(); err != nil {
			log.Errorf("conn.Receive() error(%v)", err)
			return
		}
	}
	return
}

//*从 Redis 中获取服务器的在线信息
func (d *Dao) ServerOnline(c context.Context, server string) (online *model.Online, err error) {
	online = &model.Online{RoomCount: map[string]int32{}}
	key := keyServerOnline(server)
	for i := 0; i < 64; i++ {
		ol, err := d.serverOnline(c, key, strconv.FormatInt(int64(i), 10))
		if err == nil && ol != nil {
			online.Server = ol.Server
			if ol.Updated > online.Updated {
				online.Updated = ol.Updated
			}
			for room, count := range ol.RoomCount {
				online.RoomCount[room] = count
			}
		}
	}
	return
}

//*从 Redis 中获取单个哈希桶的在线信息
func (d *Dao) serverOnline(c context.Context, key string, hashKey string) (online *model.Online, err error) {
	conn := d.redis.Get()
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("HGET", key, hashKey))
	if err != nil {
		if err != redis.ErrNil {
			log.Errorf("conn.Do(HGET %s %s) error(%v)", key, hashKey, err)
		}
		return
	}
	online = new(model.Online)
	if err = json.Unmarshal(b, online); err != nil {
		log.Errorf("serverOnline json.Unmarshal(%s) error(%v)", b, err)
		return
	}
	return
}

//*从 Redis 中删除服务器的在线信息
func (d *Dao) DelServerOnline(c context.Context, server string) (err error) {
	conn := d.redis.Get()
	defer conn.Close()
	key := keyServerOnline(server)
	if _, err = conn.Do("DEL", key); err != nil {
		log.Errorf("conn.Do(DEL %s) error(%v)", key, err)
	}
	return
}
