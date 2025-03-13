package comet

import (
	"github.com/gyy0727/mygoim/api/protocol"
	"github.com/gyy0727/mygoim/internal/comet/conf"
	"github.com/gyy0727/mygoim/internal/comet/errors"
	"go.uber.org/zap"
)

type Ring struct {
	rp   uint64           //*读指针
	num  uint64           //*缓冲区的大小,必须是2的幂次方
	mask uint64           //*掩码,用于快速计算索引,num-1
	wp   uint64           //*写指针
	data []protocol.Proto //*储存protocol.Proto的消息
}

// *初始化一个环形缓冲区
func NewRing(num int) *Ring {
	r := new(Ring)
	r.init(uint64(num))
	return r
}

func (r *Ring) Init(num int) {
	r.init(uint64(num))
}

// *根据num初始化缓冲区
func (r *Ring) init(num uint64) {
	//*确保num是2的幂次方
	if num&(num-1) != 0 {
		for num&(num-1) != 0 {
			num &= num - 1
		}
		num <<= 1
	}
	//*初始化切片
	r.data = make([]protocol.Proto, num)
	r.num = num
	r.mask = r.num - 1
}

// *取出消息
func (r *Ring) Get() (proto *protocol.Proto, err error) {
	//*检查缓冲区是否为空
	if r.rp == r.wp {
		return nil, errors.ErrRingEmpty
	}
	proto = &r.data[r.rp&r.mask]
	return
}

// *后移缓冲区的读指针
func (r *Ring) GetAdv() {
	r.rp++
	if conf.Conf.Debug {
		logger.Info("ring rp and idx",
			zap.Uint64("rp", r.rp),         //*使用 zap.Uint64 记录 uint64 类型的 rp 值
			zap.Uint64("idx", r.rp&r.mask), //*使用 zap.Uint64 记录 uint64 类型的 idx 值
		)

	}
}

// *用于向环形缓冲区（Ring）中写入一个 protocol.Proto 消息
func (r *Ring) Set() (proto *protocol.Proto, err error) {
	//*已满
	if r.wp-r.rp >= r.num {
		return nil, errors.ErrRingFull
	}
	proto = &r.data[r.wp&r.mask]
	return
}

// *Set 函数的主要功能是向环形缓冲区中写入一个 protocol.Proto 消息
func (r *Ring) SetAdv() {
	r.wp++
	if conf.Conf.Debug {
		logger.Info("ring wp and idx",
			zap.Uint64("wp", r.wp),         //*使用 zap.Uint64 记录 uint64 类型的 wp 值
			zap.Uint64("idx", r.wp&r.mask), //*使用 zap.Uint64 记录 uint64 类型的 idx 值
		)

	}
}


//*重置读写指针为初始0
func (r *Ring) Reset() {
	r.rp = 0
	r.wp = 0
}
