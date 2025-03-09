package http

import (
	"context"

	"github.com/gin-gonic/gin"
)

//*用于处理加权节点的请求
func (s *Server) nodesWeighted(c *gin.Context) {
	//*定义一个结构体 arg，用于绑定查询参数 platform
	var arg struct {
		Platform string `form:"platform"`
	}
	//*使用 c.BindQuery 将查询参数绑定到 arg 结构体
	if err := c.BindQuery(&arg); err != nil {
		errors(c, RequestErr, err.Error())
		return
	}
	//*调用 s.logic.NodesWeighted 方法，传入上下文、平台信息和客户端IP，获取加权节点信息
	res := s.logic.NodesWeighted(c, arg.Platform, c.ClientIP())
	result(c, res, OK)
}

//*处理节点实例请求的方法
func (s *Server) nodesInstances(c *gin.Context) {
	res := s.logic.NodesInstances(context.TODO())
	result(c, res, OK)
}
