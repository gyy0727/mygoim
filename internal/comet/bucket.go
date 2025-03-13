package comet

import (
	"sync"
	"sync/atomic"
	pb "github.com/gyy0727/mygoim/api/comet"
	"github.com/gyy0727/mygoim/api/protocol"
	"github.com/gyy0727/mygoim/internal/comet/conf"
)

type Bucket struct {
	c           *conf.Bucket                //*配置项
	cLock       sync.RWMutex                //*读写锁
	chs         map[string]*Channel         //*存储用户连接（Channel），键为用户标识（sub key)
	rooms       map[string]*Room            //*存储房间（Room），键为房间 ID
	routines    []chan *pb.BroadcastRoomReq //*用于广播消息的 goroutine 通道列表
	routinesNum uint64                      //*广播请求的计数器
	ipCnts      map[string]int32            //*记录每个IP地址的连接数
}

func NewBucket(c *conf.Bucket) (b *Bucket) {
	b = new(Bucket)
	b.chs = make(map[string]*Channel, c.Channel)
	b.ipCnts = make(map[string]int32)
	b.c = c
	b.rooms = make(map[string]*Room, c.Room)
	b.routines = make([]chan *pb.BroadcastRoomReq, c.RoutineAmount)
	//*在循环中为每个协程创建一个缓冲通道，通道元素是指向 pb.BroadcastRoomReq 的指针
	for i := uint64(0); i < c.RoutineAmount; i++ {
		c := make(chan *pb.BroadcastRoomReq, c.RoutineSize)
		b.routines[i] = c
		go b.roomproc(c)
	}
	return
}

// *bucket包含的channel数量
func (b *Bucket) ChannelCount() int {
	return len(b.chs)
}

// *返回房间数量,包含所有已注册房间（无论是否在线）
func (b *Bucket) RoomCount() int {
	return len(b.rooms)
}

// *返回包含在线用户的房间ID及其对应在线数,过滤离线房间
func (b *Bucket) RoomsCount() (res map[string]int32) {
	var (
		roomID string
		room   *Room
	)
	b.cLock.RLock()
	res = make(map[string]int32)
	for roomID, room = range b.rooms {
		if room.Online > 0 {
			res[roomID] = room.Online
		}
	}
	b.cLock.RUnlock()
	return
}

// *将channel迁移到新房间 
func (b *Bucket) ChangeRoom(nrid string, ch *Channel) (err error) {
	var (
		nroom *Room
		ok    bool
		oroom = ch.Room
	)
	//*删除房间
	if nrid == "" {
		//*将通道从原来的房间移除 
		if oroom != nil && oroom.Del(ch) {
			//*房间无在线人数 
			b.DelRoom(oroom)
		}
		ch.Room = nil
		return
	}
	b.cLock.Lock()
	//*检查nrid对应的房间是否存在  
	if nroom, ok = b.rooms[nrid]; !ok {
		nroom = NewRoom(nrid)
		b.rooms[nrid] = nroom
	}
	b.cLock.Unlock()
	if oroom != nil && oroom.Del(ch) {
		b.DelRoom(oroom)
	}
	//*添加到新房间 
	if err = nroom.Put(ch); err != nil {
		return
	}
	ch.Room = nroom
	return
}

//*添加连接 
func (b *Bucket) Put(rid string, ch *Channel) (err error) {
	var (
		room *Room
		ok   bool
	)
	b.cLock.Lock()
	//*关闭旧通道（如果存在相同Key的通道）
	if dch := b.chs[ch.Key]; dch != nil {
		dch.Close()
	}
	//*将新通道存入Bucket的chs映射
	b.chs[ch.Key] = ch
	if rid != "" {
		//*rid对应的房间不存在 
		if room, ok = b.rooms[rid]; !ok {
			//*创建房间 
			room = NewRoom(rid)
			b.rooms[rid] = room
		}
		//*关联通道 
		ch.Room = room
	}
	//*添加ip对应的链接计数 
	b.ipCnts[ch.IP]++
	b.cLock.Unlock()
	if room != nil {
		//*将通道添加到房间中 
		err = room.Put(ch)
	}
	return
}

// *删除key对应的通道
func (b *Bucket) Del(dch *Channel) {
	room := dch.Room
	b.cLock.Lock()
	//*如果存在该key
	if ch, ok := b.chs[dch.Key]; ok {
		if ch == dch {
			//*删除通道
			delete(b.chs, ch.Key)
		}
		//*移除客户端ip 的链接数量
		if b.ipCnts[ch.IP] > 1 {
			b.ipCnts[ch.IP]--
		} else {
			//*该ip地址不存在连接,移除ip
			delete(b.ipCnts, ch.IP)
		}
	}
	b.cLock.Unlock()
	if room != nil && room.Del(dch) {
		//*房间已经没有活跃用户了,移除房间 
		b.DelRoom(room)
	}
}

// *返回key对应的通道
func (b *Bucket) Channel(key string) (ch *Channel) {
	b.cLock.RLock()
	ch = b.chs[key]
	b.cLock.RUnlock()
	return
}

// *广播消息
func (b *Bucket) Broadcast(p *protocol.Proto, op int32) {
	var ch *Channel
	b.cLock.RLock()
	for _, ch = range b.chs {
		if !ch.NeedPush(op) {
			continue
		}
		_ = ch.Push(p)
	}
	b.cLock.RUnlock()
}

// *返回roomid对应的room
func (b *Bucket) Room(rid string) (room *Room) {
	b.cLock.RLock()
	room = b.rooms[rid]
	b.cLock.RUnlock()
	return
}

// *删除roomid对应的room
func (b *Bucket) DelRoom(room *Room) {
	b.cLock.Lock()
	delete(b.rooms, room.ID)
	b.cLock.Unlock()
	room.Close()
}

// *实现房间级广播请求的并行化分发
func (b *Bucket) BroadcastRoom(arg *pb.BroadcastRoomReq) {
	num := atomic.AddUint64(&b.routinesNum, 1) % b.c.RoutineAmount
	b.routines[num] <- arg
}

// *获取所有包含在线用户的房间ID集合
func (b *Bucket) Rooms() (res map[string]struct{}) {
	var (
		roomID string
		room   *Room
	)
	res = make(map[string]struct{})
	b.cLock.RLock()
	for roomID, room = range b.rooms {
		if room.Online > 0 {
			res[roomID] = struct{}{}
		}
	}
	b.cLock.RUnlock()
	return
}

// *获取当前存在连接计数的所有IP地址集合
func (b *Bucket) IPCount() (res map[string]struct{}) {
	var (
		ip string
	)
	b.cLock.RLock()
	res = make(map[string]struct{}, len(b.ipCnts))
	for ip = range b.ipCnts {
		res[ip] = struct{}{}
	}
	b.cLock.RUnlock()
	return
}

//*实现房间人数统计更新
//*roomCountMap通常从外部存储（如Redis集群）获取全局统计
func (b *Bucket) UpRoomsCount(roomCountMap map[string]int32) {
	var (
		roomID string
		room   *Room
	)
	b.cLock.RLock()
	for roomID, room = range b.rooms {
		room.AllOnline = roomCountMap[roomID]
	}
	b.cLock.RUnlock()
}

// *
func (b *Bucket) roomproc(c chan *pb.BroadcastRoomReq) {
	for {
		arg := <-c
		if room := b.Room(arg.RoomID); room != nil {
			room.Push(arg.Proto)
		}
	}
}
