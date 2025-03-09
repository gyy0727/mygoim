package http

import (
	"fmt"
	"net/http/httputil"
	"runtime"
	"time"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
)

// *NOTE 理解gin框架的洋葱模型
func loggerHandler(c *gin.Context) {
	//*start : 当前时间
	start := time.Now()
	//*获得请求路径
	path := c.Request.URL.Path
	//*请求的查询参数
	raw := c.Request.URL.RawQuery
	//*请求的方法
	method := c.Request.Method
	//*移交控制权给下一个中间件
	c.Next()

	//*执行结束的时间点
	end := time.Now()
	//*计算操作耗时
	latency := end.Sub(start)
	//*获取HTTP响应的状态码
	statusCode := c.Writer.Status()
	//*从上下文中获取自定义的错误码（contextErrCode）
	ecode := c.GetInt(contextErrCode)
	//*获取客户端ip
	clientIP := c.ClientIP()
	if raw != "" {
		path = path + "?" + raw
	}
	log.Infof("METHOD:%s | PATH:%s | CODE:%d | IP:%s | TIME:%d | ECODE:%d", method, path, statusCode, clientIP, latency/time.Millisecond, ecode)
}

// *用于捕获和处理 panic 的中间件函数
func recoverHandler(c *gin.Context) {
	defer func() {
		//*使用 recover() 捕获 panic，err 是 panic 的值
		if err := recover(); err != nil {
			//*定义一个常量 size，值为 64KB，用于分配堆栈跟踪的缓冲区大小
			const size = 64 << 10
			buf := make([]byte, size)
			//*获取当前的堆栈跟踪信息，并将其存储在 buf 中
			buf = buf[:runtime.Stack(buf, false)]
			//*获取当前 HTTP 请求的信息，并将其存储在 httprequest
			httprequest, _ := httputil.DumpRequest(c.Request, false)
			pnc := fmt.Sprintf("[Recovery] %s panic recovered:\n%s\n%s\n%s", time.Now().Format("2006-01-02 15:04:05"), string(httprequest), err, buf)
			fmt.Print(pnc)
			log.Error(pnc)
			//*中止请求并返回 500 状态码
			c.AbortWithStatus(500)
		}
	}()
	c.Next()
}
