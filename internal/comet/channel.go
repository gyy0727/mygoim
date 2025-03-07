package comet

import (
	"sync"

	"github.com/gyy0727/mygoim/internal/comet/errors"
	"github.com/gyy0727/mygoim/pkg/bufio"

	"github.com/gyy0727/mygoim/api/protocol"
)

type Channel struct {
	Room     *Room                //*关联的房间
	CliProto Ring                 //*客户端协议缓冲区（环形缓冲区）
	signal   chan *protocol.Proto //* 用于传递协议消息的信号通道
	Writer   bufio.Writer         //*用于写入数据的缓冲区
	Reader   bufio.Reader         //*用于读取数据的缓冲区
	Next     *Channel             //*双向链表中的下一个 Channel
	Prev     *Channel             //*双向链表中的上一个 Channel
	Mid      int64                //*用户 ID
	Key      string               //*客户端唯一标识
	IP       string               //*客户端 IP 地址
	watchOps map[int32]struct{}   //*监听的操作集合
	mutex    sync.RWMutex         //*读写锁，用于保护 watchOps 的并发访问
}

// *新建一个通道
func NewChannel(cli, svr int) *Channel {
	c := new(Channel)
	c.CliProto.Init(cli)
	c.signal = make(chan *protocol.Proto, svr)
	c.watchOps = make(map[int32]struct{})
	return c
}

// *用于将指定的操作添加到监听集合 (watchOps) 中
func (c *Channel) Watch(accepts ...int32) {
	c.mutex.Lock()
	for _, op := range accepts {
		c.watchOps[op] = struct{}{}
	}
	c.mutex.Unlock()
}

// *用于从监听集合 (watchOps) 中移除指定的操作
func (c *Channel) UnWatch(accepts ...int32) {
	c.mutex.Lock()
	for _, op := range accepts {
		delete(c.watchOps, op)
	}
	c.mutex.Unlock()
}

// *用于检查某个操作是否在监听集合 (watchOps) 中
func (c *Channel) NeedPush(op int32) bool {
	c.mutex.RLock()
	if _, ok := c.watchOps[op]; ok {
		c.mutex.RUnlock()
		return true
	}
	c.mutex.RUnlock()
	return false
}

//*用于将消息推送到 Channel 的信号通道 (signal)
func (c *Channel) Push(p *protocol.Proto) (err error) {
	select {
	case c.signal <- p:
	default:
		err = errors.ErrSignalFullMsgDropped
	}
	return
}

///*用于从信号通道 (signal) 中读取消息
func (c *Channel) Ready() *protocol.Proto {
	return <-c.signal
}


//*用于向信号通道 (signal) 发送 ProtoReady 信号，通知 Channel 有新的消息需要处理
func (c *Channel) Signal() {
	c.signal <- protocol.ProtoReady
}

//*用于向信号通道 (signal) 发送 ProtoFinish 信号，通知 Channel 关闭
func (c *Channel) Close() {
	c.signal <- protocol.ProtoFinish
}
