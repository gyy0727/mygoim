package logic

import (
	"context"
	"strconv"
	"time"
	"github.com/bilibili/discovery/naming"
	log "github.com/golang/glog"
	"github.com/gyy0727/mygoim/internal/logic/conf"
	"github.com/gyy0727/mygoim/internal/logic/dao"
	"github.com/gyy0727/mygoim/internal/logic/model"
)

//*用于实现 IM 系统的核心逻辑，包括节点管理、负载均衡、在线状态维护等功能

const (
	_onlineTick     = time.Second * 10 //*在线状态检查的时间间隔（10 秒）
	_onlineDeadline = time.Minute * 5  //*在线状态的超时时间（5 分钟）
)

type Logic struct {
	c            *conf.Config       //*配置信息
	dis          *naming.Discovery  //*服务发现模块
	dao          *dao.Dao           //*数据访问对象
	totalIPs     int64              //*总ip数
	totalConns   int64              //*总连接数
	roomCount    map[string]int32   //*房间在线人数统计
	nodes        []*naming.Instance //*节点列表
	loadBalancer *LoadBalancer      //*负载均衡器
	regions      map[string]string  //*省份到区域的映射
}

func New(c *conf.Config) (l *Logic) {
	l = &Logic{
		c:            c,
		dao:          dao.New(c),
		dis:          naming.New(c.Discovery),
		loadBalancer: NewLoadBalancer(),
		regions:      make(map[string]string),
	}
	l.initRegions()
	l.initNodes()
	_ = l.loadOnline()
	go l.onlineproc()
	return l
}

// *测试redis可用性
func (l *Logic) Ping(c context.Context) (err error) {
	return l.dao.Ping(c)
}

// *关闭数据库资源
func (l *Logic) Close() {
	l.dao.Close()
}

// *根据配置初始化省份到区域的映射关系
func (l *Logic) initRegions() {
	for region, ps := range l.c.Regions {
		for _, province := range ps {
			l.regions[province] = region
		}
	}
}

func (l *Logic) initNodes() {
	res := l.dis.Build("goim.comet")
	event := res.Watch()
	select {
	case _, ok := <-event:
		if ok {
			l.newNodes(res)
		} else {
			panic("discovery watch failed")
		}
	case <-time.After(10 * time.Second):
		log.Error("discovery start timeout")
	}
	go func() {
		for {
			if _, ok := <-event; !ok {
				return
			}
			l.newNodes(res)
		}
	}()
}

func (l *Logic) newNodes(res naming.Resolver) {
	//*res.Fetch() 从服务发现模块中获取当前的所有节点信息
	if zoneIns, ok := res.Fetch(); ok {
		var (
			totalConns int64
			totalIPs   int64
			allIns     []*naming.Instance //* 用于存储所有有效的节点实例
		)
		//*对每个节点，检查其元数据是否有效
		//*如果元数据缺失或无效，跳过该节点并记录错误日志
		for _, zins := range zoneIns.Instances {
			for _, ins := range zins {
				if ins.Metadata == nil {
					log.Errorf("node instance metadata is empty(%+v)", ins)
					continue
				}
				offline, err := strconv.ParseBool(ins.Metadata[model.MetaOffline])
				if err != nil || offline {
					log.Warningf("strconv.ParseBool(offline:%t) error(%v)", offline, err)
					continue
				}
				conns, err := strconv.ParseInt(ins.Metadata[model.MetaConnCount], 10, 32)
				if err != nil {
					log.Errorf("strconv.ParseInt(conns:%d) error(%v)", conns, err)
					continue
				}
				ips, err := strconv.ParseInt(ins.Metadata[model.MetaIPCount], 10, 32)
				if err != nil {
					log.Errorf("strconv.ParseInt(ips:%d) error(%v)", ips, err)
					continue
				}
				totalConns += conns
				totalIPs += ips
				allIns = append(allIns, ins)
			}
		}
		l.totalConns = totalConns
		l.totalIPs = totalIPs
		l.nodes = allIns
		l.loadBalancer.Update(allIns)
	}
}

// *一个后台 goroutine，用于定期检查和更新 IM 系统中用户的在线状态信息
func (l *Logic) onlineproc() {
	for {
		time.Sleep(_onlineTick)
		if err := l.loadOnline(); err != nil {
			log.Errorf("onlineproc error(%v)", err)
		}
	}
}

// *加载在线信息
func (l *Logic) loadOnline() (err error) {
	var (
		roomCount = make(map[string]int32)
	)
	for _, server := range l.nodes {
		//* // 定义一个变量，用于存储从数据库获取的在线信息
		var online *model.Online
		//* 调用 dao 层的 ServerOnline 方法，获取当前节点的在线信息
		online, err = l.dao.ServerOnline(context.Background(), server.Hostname)
		if err != nil {
			return
		}
		//* 检查在线信息的更新时间是否超过超时时间（_onlineDeadline）
		if time.Since(time.Unix(online.Updated, 0)) > _onlineDeadline {
			//* // 如果超时，删除该节点的在线信息
			_ = l.dao.DelServerOnline(context.Background(), server.Hostname)
			continue
		}
		for roomID, count := range online.RoomCount {
			//* 将房间在线人数累加到 roomCount 中
			roomCount[roomID] += count
		}
	}
	l.roomCount = roomCount
	return
}
