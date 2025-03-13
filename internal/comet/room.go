package comet

import (
	"sync"

	"github.com/gyy0727/mygoim/api/protocol"
	"github.com/gyy0727/mygoim/internal/comet/errors"
	"go.uber.org/zap"
)

type Room struct {
	ID        string       //*房间名称
	rLock     sync.RWMutex //*读锁
	next      *Channel     //*指向房间中双向链表的头指针
	drop      bool         //*是否已被丢弃
	Online    int32        //*房间在线用户的数量
	AllOnline int32        //*历史在线用户数量
}

func NewRoom(id string) (r *Room) {
	logger.Info("NewRoom,新建房间", zap.String("id", id))
	r = new(Room)
	r.ID = id
	r.drop = false
	r.next = nil
	r.Online = 0
	return
}

// *用于将 Channel 添加到 Room 中
func (r *Room) Put(ch *Channel) (err error) {
	r.rLock.Lock()
	//*如果房间没有被丢弃,则将 Channel 插入到双向链表的头部
	if !r.drop {
		if r.next != nil {
			r.next.Prev = ch
		}
		ch.Next = r.next
		ch.Prev = nil
		r.next = ch
		//*增加在线用户的数量
		r.Online++
	} else {
		err = errors.ErrRoomDroped
	}
	r.rLock.Unlock()
	logger.Info("房间添加通道", zap.String("房间id", r.ID), zap.Int32("房间在线人数", r.Online))
	return
}

// *删除通道,返回房间是否还存在活跃用户
func (r *Room) Del(ch *Channel) bool {
	//*将传入的通道的前后项通过双向链表相连
	r.rLock.Lock()
	if ch.Next != nil {
		ch.Next.Prev = ch.Prev
	}
	if ch.Prev != nil {
		ch.Prev.Next = ch.Next
	} else {
		r.next = ch.Next
	}
	ch.Next = nil
	ch.Prev = nil
	//*更新在线人数和drop
	r.Online--
	r.drop = r.Online == 0
	r.rLock.Unlock()
	logger.Info("房间删除通道", zap.String("房间id", r.ID), zap.Int32("房间在线人数", r.Online))
	return r.drop
}

// *广播消息
func (r *Room) Push(p *protocol.Proto) {
	r.rLock.RLock()
	for ch := r.next; ch != nil; ch = ch.Next {
		_ = ch.Push(p)
	}
	logger.Info("房间广播消息", zap.String("房间id", r.ID), zap.Int32("房间在线人数", r.Online))
	r.rLock.RUnlock()
}

// *用于关闭房间中的所有 Channel
func (r *Room) Close() {
	r.rLock.RLock()
	for ch := r.next; ch != nil; ch = ch.Next {
		ch.Close()
	}
	logger.Info("房间关闭所有channel", zap.String("房间id", r.ID), zap.Int32("房间在线人数", r.Online))
	r.rLock.RUnlock()
}

// *用于获取房间的在线用户数
func (r *Room) OnlineNum() int32 {
	if r.AllOnline > 0 {
		return r.AllOnline
	}
	logger.Info("房间获取在线人数", zap.String("房间id", r.ID), zap.Int32("房间在线人数", r.Online))
	return r.Online
}
