package time

import (
	"sync"
	itime "time"

	"go.uber.org/zap"
)

const (
	timerFormat      = "2006-01-02 15:04:05"
	infiniteDuration = itime.Duration(1<<63 - 1)
)

type TimerData struct {
	Key    string     //*任务的唯一标识
	expire itime.Time //*任务的过期时间
	fn     func()     //*任务到期时执行的函数
	index  int        //*任务在堆中的索引
	next   *TimerData //*指向下一个 TimerData，用于空闲链表
}

// *计算当前时间到任务过期时间的剩余时间
func (td *TimerData) Delay() itime.Duration {
	return itime.Until(td.expire)
}

// *将 TimerData 的过期时间（td.expire）格式化为字符串
func (td *TimerData) ExpireString() string {
	return td.expire.Format(timerFormat)
}

type Timer struct {
	lock   sync.Mutex
	free   *TimerData   //*指向一个空闲的 TimerData 对象，用于复用已完成的定时任务，减少内存分配和垃圾回收的开销
	timers []*TimerData //*存储所有活跃的定时任务，通常是一个最小堆（Min-Heap），用于快速获取下一个即将到期的任务
	signal *itime.Timer //*用于触发下一个定时任务的执行。它是一个 time.Timer 对象，会在下一个任务到期时触发
	num    int          //*记录当前活跃的定时任务数量
}

// *新建一个定时器管理器
func NewTimer(num int) (t *Timer) {
	t = new(Timer)
	t.init(num)
	return t
}

func (t *Timer) Init(num int) {
	t.init(num)
}

// *初始化带n个定时器的timer
func (t *Timer) init(num int) {
	t.signal = itime.NewTimer(infiniteDuration)
	t.timers = make([]*TimerData, 0, num)
	t.num = num
	t.grow()
	go t.start()
}

// *用于扩展空闲的 TimerData 对象池
func (t *Timer) grow() {
	var (
		i   int
		td  *TimerData
		tds = make([]TimerData, t.num)
	)
	t.free = &(tds[0])
	td = t.free
	for i = 1; i < t.num; i++ {
		td.next = &(tds[i])
		td = td.next
	}
	td.next = nil
}

// *取出一个空闲的timerdata
func (t *Timer) get() (td *TimerData) {
	if td = t.free; td == nil {
		t.grow()
		td = t.free
	}
	t.free = td.next
	return
}

// *将空闲的timerdata对象存入链表实现复用
func (t *Timer) put(td *TimerData) {
	td.fn = nil
	td.next = t.free
	t.free = td
}

// *用于添加一个新的定时任务
func (t *Timer) Add(expire itime.Duration, fn func()) (td *TimerData) {
	t.lock.Lock()
	td = t.get()
	td.expire = itime.Now().Add(expire)
	td.fn = fn
	t.add(td)
	t.lock.Unlock()
	return
}

// *删除一个定时任务
func (t *Timer) Del(td *TimerData) {
	t.lock.Lock()
	t.del(td)
	t.put(td)
	t.lock.Unlock()
}

// *添加定时任务
func (t *Timer) add(td *TimerData) {
	var d itime.Duration
	//*定时任务的索引 
	td.index = len(t.timers)
	//*添加到定时任务列表 
	t.timers = append(t.timers, td)
	t.up(td.index)
	if td.index == 0 {
		//*新添加的任务排到了最前面 
		d = td.Delay()
		t.signal.Reset(d)
		if Debug {
			logger.Info("timer: add reset delay", zap.Int64("delay_ms", int64(d)/int64(itime.Millisecond)))

		}
	}
	if Debug {
		logger.Info("timer: push item",
			zap.String("key", td.Key),
			zap.String("expire", td.ExpireString()),
			zap.Int("index", td.index),
		)

	}
}

func (t *Timer) del(td *TimerData) {
	var (
		i    = td.index
		last = len(t.timers) - 1
	)
	if i < 0 || i > last || t.timers[i] != td {
		if Debug {
			logger.Info("timer del",
				zap.Int("i", i),
				zap.Int("last", last),
				zap.Any("td", td),
			)

		}
		return
	}
	if i != last {
		t.swap(i, last)
		t.down(i, last)
		t.up(i)
	}

	t.timers[last].index = -1
	t.timers = t.timers[:last]
	if Debug {
		logger.Info("timer: remove item",
			zap.String("key", td.Key),
			zap.String("expire", td.ExpireString()),
			zap.Int("index", td.index),
		)

	}
}
//*更新对应的定时任务的过期时间 
func (t *Timer) Set(td *TimerData, expire itime.Duration) {
	t.lock.Lock()
	t.del(td)
	td.expire = itime.Now().Add(expire)
	t.add(td)
	t.lock.Unlock()
}

//*开始监听定时任务 
func (t *Timer) start() {
	for {
		t.expire()
		//*NOTE 只有接收到超时信号才会继续执行expire
		<-t.signal.C
	}
}

func (t *Timer) expire() {
	var (
		fn func()
		td *TimerData
		d  itime.Duration
	)
	t.lock.Lock()
	for {
		if len(t.timers) == 0 {
			d = infiniteDuration
			if Debug {
				logger.Info("timer: no other instance")
			}
			break
		}
		td = t.timers[0]
		//*最接近超时的定时任务仍未超时 
		if d = td.Delay(); d > 0 {
			break
		}
		//*超时 
		fn = td.fn
		t.del(td)
		t.lock.Unlock()
		if fn == nil {
			logger.Warn("expire timer no fn")
		} else {
			if Debug {
				logger.Info("timer expired, call fn",
					zap.String("key", td.Key),
					zap.String("expire", td.ExpireString()),
					zap.Int("index", td.index),
				)

			}
			//*执行超时回调 
			fn()
		}
		t.lock.Lock()
	}
	t.signal.Reset(d)
	if Debug {
		logger.Info("timer: expire reset delay",
			zap.Int64("delay_ms", int64(d)/int64(itime.Millisecond)),
		)

	}
	t.lock.Unlock()
}


func (t *Timer) up(j int) {
	for {
		i := (j - 1) / 2
		if i >= j || !t.less(j, i) {
			break
		}
		t.swap(i, j)
		j = i
	}
}

func (t *Timer) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && !t.less(j1, j2) {
			j = j2
		}
		if !t.less(j, i) {
			break
		}
		t.swap(i, j)
		i = j
	}
}

func (t *Timer) less(i, j int) bool {
	return t.timers[i].expire.Before(t.timers[j].expire)
}

func (t *Timer) swap(i, j int) {
	t.timers[i], t.timers[j] = t.timers[j], t.timers[i]
	t.timers[i].index = i
	t.timers[j].index = j
}
