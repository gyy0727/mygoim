package comet

import (
	"github.com/gyy0727/mygoim/internal/comet/conf"
	"github.com/gyy0727/mygoim/pkg/bytes"
	"github.com/gyy0727/mygoim/pkg/time"
)

//*用于在 comet 服务器中实现连接的分片管理。Round 结构体的主要目的
//*通过轮询（round-robin）的方式分配资源（如读写缓冲区和定时器），以减少锁竞争并提高并发性能

type RoundOptions struct {
	Timer        int //*定时器的数量
	TimerSize    int //*每个定时器的大小
	Reader       int //*读缓冲区的数量
	ReadBuf      int //*每个读缓冲区的大小
	ReadBufSize  int //*读缓冲区的容量
	Writer       int //*写缓冲区的数量
	WriteBuf     int //*每个写缓冲区的大小
	WriteBufSize int //*写缓冲区的容量
}

type Round struct {
	readers []bytes.Pool //*读缓冲区池
	writers []bytes.Pool //*写缓冲区池
	timers  []time.Timer //*定时器池
	options RoundOptions //*配置选项
}

func NewRound(c *conf.Config) (r *Round) {
	var i int
	r = &Round{
		options: RoundOptions{
			Reader:       c.TCP.Reader,
			ReadBuf:      c.TCP.ReadBuf,
			ReadBufSize:  c.TCP.ReadBufSize,
			Writer:       c.TCP.Writer,
			WriteBuf:     c.TCP.WriteBuf,
			WriteBufSize: c.TCP.WriteBufSize,
			Timer:        c.Protocol.Timer,
			TimerSize:    c.Protocol.TimerSize,
		}}
	//*初始化读缓冲区
	r.readers = make([]bytes.Pool, r.options.Reader)
	for i = 0; i < r.options.Reader; i++ {
		r.readers[i].Init(r.options.ReadBuf, r.options.ReadBufSize)
	}
	//*初始化写缓冲区
	r.writers = make([]bytes.Pool, r.options.Writer)
	for i = 0; i < r.options.Writer; i++ {
		r.writers[i].Init(r.options.WriteBuf, r.options.WriteBufSize)
	}
	//*初始化定时器管理器
	r.timers = make([]time.Timer, r.options.Timer)
	for i = 0; i < r.options.Timer; i++ {
		r.timers[i].Init(r.options.TimerSize)
	}
	return
}

//*通过轮询的方式（rn%r.options.Timer）从定时器池中选择一个定时器
func (r *Round) Timer(rn int) *time.Timer {
	return &(r.timers[rn%r.options.Timer])
}

//*通过轮询的方式（rn%r.options.Reader）从读缓冲区池中选择一个读缓冲区
func (r *Round) Reader(rn int) *bytes.Pool {
	return &(r.readers[rn%r.options.Reader])
}

//*通过轮询的方式（rn%r.options.Writer）从写缓冲区池中选择一个写缓冲区
func (r *Round) Writer(rn int) *bytes.Pool {
	return &(r.writers[rn%r.options.Writer])
}
