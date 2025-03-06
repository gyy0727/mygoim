package comet

import (
	"log"
	"os"

	"github.com/gyy0727/mygoim/internal/comet/conf"
)

var whitelist *Whitelist

type Whitelist struct {
	log  *log.Logger
	list map[int64]struct{}
}

//*初始化,将config结构体的白名单数据加载到当前Whitelsit结构体 
func InitWhitelist(c *conf.Whitelist) (err error) {
	var (
		mid int64
		f   *os.File
	)

	if f, err = os.OpenFile(c.WhiteLog, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644); err == nil {
		whitelist = new(Whitelist)
		whitelist.log = log.New(f, "", log.LstdFlags)
		whitelist.list = make(map[int64]struct{})
		for _, mid = range c.Whitelist {
			whitelist.list[mid] = struct{}{}
		}
	}
	return
}

//*判断是否包含当前用户 
func (w *Whitelist) Contains(mid int64) (ok bool) {
	if mid > 0 {
		_, ok = w.list[mid]
	}
	return
}

//*用于将格式化日志输出到 Whitelist 的日志文件中
func (w *Whitelist) Printf(format string, v ...interface{}) {
	w.log.Printf(format, v...)
}
