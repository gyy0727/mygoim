package bytes

import "sync"

type Buffer struct {
	buf  []byte
	next *Buffer //*内存池
}

// *返回内存池的缓冲区
func (b *Buffer) Bytes() []byte {
	return b.buf
}

type Pool struct {
	lock sync.Mutex //*同步
	free *Buffer    //*空闲的内存池
	max  int        //*内存池的最大大小
	num  int        //*表示池中缓冲区的数量
	size int        //*表示每个缓冲区的大小
}

func NewPool(num, size int) (p *Pool) {
	p = new(Pool)
	p.init(num, size)
	return
}

func (p *Pool) init(num, size int) {
	p.num = num
	p.size = size
	p.max = num * size
	p.grow()
}

// *扩充内存池
func (p *Pool) grow() {
	var (
		i   int
		b   *Buffer
		bs  []Buffer
		buf []byte
	)
	//*分配一个连续的缓冲区 
	buf = make([]byte, p.max)
	//*创建一个指定数量的缓冲区切片
	bs = make([]Buffer, p.num)
	p.free = &bs[0]
	b = p.free
	for i = 1; i < p.num; i++ {
		b.buf = buf[(i-1)*p.size : i*p.size]
		b.next = &bs[i]
		b = b.next
	}
	b.buf = buf[(i-1)*p.size : i*p.size]
	b.next = nil
}


//*链表的方式组成一个内存池
func (p *Pool) Get() (b *Buffer) {
	p.lock.Lock()
	if b = p.free; b == nil {
		p.grow()
		b = p.free
	}
	p.free = b.next
	p.lock.Unlock()
	return
}


func (p *Pool) Put(b *Buffer) {
	p.lock.Lock()
	b.next = p.free
	p.free = b
	p.lock.Unlock()
}
