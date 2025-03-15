package logic

// import (
// 	"context"
// 	"time"

// 	"github.com/bilibili/discovery/naming"
// 	log "github.com/golang/glog"
// 	pb "github.com/gyy0727/mygoim/api/logic"
// 	"github.com/gyy0727/mygoim/internal/logic/model"
// )

// //*返回node实例
// func (l *Logic) NodesInstances(c context.Context) (res []*naming.Instance) {
// 	return l.nodes
// }

// //*该方法根据平台和客户端 IP 返回加权节点列表
// func (l *Logic) NodesWeighted(c context.Context, platform, clientIP string) *pb.NodesReply {
// 	reply := &pb.NodesReply{
// 		Domain:       l.c.Node.DefaultDomain,
// 		TcpPort:      int32(l.c.Node.TCPPort),
// 		WsPort:       int32(l.c.Node.WSPort),
// 		WssPort:      int32(l.c.Node.WSSPort),
// 		Heartbeat:    int32(time.Duration(l.c.Node.Heartbeat) / time.Second),
// 		HeartbeatMax: int32(l.c.Node.HeartbeatMax),
// 		Backoff: &pb.Backoff{
// 			MaxDelay:  l.c.Backoff.MaxDelay,
// 			BaseDelay: l.c.Backoff.BaseDelay,
// 			Factor:    l.c.Backoff.Factor,
// 			Jitter:    l.c.Backoff.Jitter,
// 		},
// 	}
// 	domains, addrs := l.nodeAddrs(c, clientIP)
// 	if platform == model.PlatformWeb {
// 		reply.Nodes = domains
// 	} else {
// 		reply.Nodes = addrs
// 	}
// 	if len(reply.Nodes) == 0 {
// 		reply.Nodes = []string{l.c.Node.DefaultDomain}
// 	}
// 	return reply
// }

// //*该方法根据客户端 IP 获取域名和地址列表
// func (l *Logic) nodeAddrs(c context.Context, clientIP string) (domains, addrs []string) {
// 	var (
// 		region string
// 	)
// 	province, err := l.location(c, clientIP)
// 	if err == nil {
// 		region = l.regions[province]
// 	}
// 	log.Infof("nodeAddrs clientIP:%s region:%s province:%s domains:%v addrs:%v", clientIP, region, province, domains, addrs)
// 	return l.loadBalancer.NodeAddrs(region, l.c.Node.HostDomain, l.c.Node.RegionWeight)
// }

// //*该方法根据客户端 IP 获取地理位置信息
// func (l *Logic) location(c context.Context, clientIP string) (province string, err error) {

// 	return
// }
