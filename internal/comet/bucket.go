package comet

import (
	"sync"

	pb "github.com/gyy0727/mygoim/api/comet"
	"github.com/gyy0727/mygoim/internal/comet/conf"
)

type Bucket struct {
	c           *conf.Bucket                //*配置项
	cLock       sync.RWMutex                //*读写锁
	chs         map[string]*Channel         //*存储用户连接（Channel），键为用户标识（sub key)
	rooms       map[string]*Room            //*存储房间（Room），键为房间 ID
	routines    []chan *pb.BroadcastRoomReq //*用于广播消息的 goroutine 通道列表
	routinesNum uint64                      //*协程的数量
	ipCnts      map[string]int32            //*记录每个 IP 地址的连接数
}


func NewBucket(c *conf.Bucket) (b *Bucket) {
	b = new(Bucket)
	b.chs = make(map[string]*Channel, c.Channel)
	b.ipCnts = make(map[string]int32)
	b.c = c
	b.rooms = make(map[string]*Room, c.Room)
	b.routines = make([]chan *pb.BroadcastRoomReq, c.RoutineAmount)
	for i := uint64(0); i < c.RoutineAmount; i++ {
		c := make(chan *pb.BroadcastRoomReq, c.RoutineSize)
		b.routines[i] = c
		go b.roomproc(c)
	}
	return
}