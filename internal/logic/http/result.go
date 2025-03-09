package http

import (
	"github.com/gin-gonic/gin"
)

const (
	//*表示请求成功 
	OK = 0
	//*请求失败 
	RequestErr = -400
	//*服务器错误 
	ServerErr = -500
	//*用于存储错误码的键名
	contextErrCode = "context/err/code"
)


//*定义 resp 结构体，用于封装HTTP响应
type resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}


//*定义 errors 函数，用于返回错误响应
func errors(c *gin.Context, code int, msg string) {
	c.Set(contextErrCode, code)
	c.JSON(200, resp{
		Code:    code,
		Message: msg,
	})
}

//*定义 result 函数，用于返回成功响应
func result(c *gin.Context, data interface{}, code int) {
	c.Set(contextErrCode, code)
	c.JSON(200, resp{
		Code: code,
		Data: data,
	})
}
