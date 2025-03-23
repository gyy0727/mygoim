package job

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	log "github.com/golang/glog"
	"github.com/gyy0727/mygoim/api/comet"
	"github.com/gyy0727/mygoim/internal/job/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	//*grpc设置
	grpcKeepAliveTime    = time.Duration(10) * time.Second //*每隔10秒发送保活探测包
	grpcKeepAliveTimeout = time.Duration(3) * time.Second  //*保活响应超时时间
	grpcBackoffMaxDelay  = time.Duration(3) * time.Second  //*连接重试最大退避延迟
	grpcMaxSendMsgSize   = 1 << 24                         //*最大发送消息大小
	grpcMaxCallMsgSize   = 1 << 24                         //*最大接收消息大小
)

const (
	grpcInitialWindowSize     = 1 << 24 //*初始化的窗口大小
	grpcInitialConnWindowSize = 1 << 24 //*初始化的连接窗口大小
)

// *新建一个CometRPC客户端,传入comet的地址
func newCometClient(addr string) (comet.CometClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second))
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr,
		[]grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithInitialWindowSize(grpcInitialWindowSize),
			grpc.WithInitialConnWindowSize(grpcInitialConnWindowSize),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcMaxCallMsgSize)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(grpcMaxSendMsgSize)),
			grpc.WithBackoffMaxDelay(grpcBackoffMaxDelay),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                grpcKeepAliveTime,
				Timeout:             grpcKeepAliveTimeout,
				PermitWithoutStream: true,
			}),
		}...,
	)
	if err != nil {
		return nil, err
	}
	return comet.NewCometClient(conn), err
}

type Comet struct {
	serverID      string                         //*服务器id
	client        comet.CometClient              //*连接comet层的rpc客户端
	pushChan      []chan *comet.PushMsgReq       //*推送用户消息的管道
	roomChan      []chan *comet.BroadcastRoomReq //*推送房间消息
	broadcastChan chan *comet.BroadcastReq       //*广播消息的管道
	pushChanNum   uint64                         //*推送用户消息管道数量
	roomChanNum   uint64                         //*推送房间消息管道数量
	routineSize   uint64                         //*协程数量
	ctx           context.Context                //*上下文
	cancel        context.CancelFunc             //*上下文取消函数
}

func NewComet(addr string, c *conf.Comet) (*Comet, error) {
	fmt.Println("addr:", addr)
	
	cmt := &Comet{
		serverID:      addr,
		pushChan:      make([]chan *comet.PushMsgReq, c.RoutineSize),
		roomChan:      make([]chan *comet.BroadcastRoomReq, c.RoutineSize),
		broadcastChan: make(chan *comet.BroadcastReq, c.RoutineSize),
		routineSize:   uint64(c.RoutineSize),
	}
	var grpcAddr string

	if grpcAddr == "" {
		return nil, fmt.Errorf("invalid grpc address:%v", addr)
	}
	var err error
	if cmt.client, err = newCometClient(addr); err != nil {
		return nil, err
	}
	cmt.ctx, cmt.cancel = context.WithCancel(context.Background())

	for i := 0; i < c.RoutineSize; i++ {
		cmt.pushChan[i] = make(chan *comet.PushMsgReq, c.RoutineChan)
		cmt.roomChan[i] = make(chan *comet.BroadcastRoomReq, c.RoutineChan)
		go cmt.process(cmt.pushChan[i], cmt.roomChan[i], cmt.broadcastChan)
	}
	return cmt, nil
}

// Push push a user message.
func (c *Comet) Push(arg *comet.PushMsgReq) (err error) {
	idx := atomic.AddUint64(&c.pushChanNum, 1) % c.routineSize
	c.pushChan[idx] <- arg
	return
}

// BroadcastRoom broadcast a room message.
func (c *Comet) BroadcastRoom(arg *comet.BroadcastRoomReq) (err error) {
	idx := atomic.AddUint64(&c.roomChanNum, 1) % c.routineSize
	c.roomChan[idx] <- arg
	return
}

// Broadcast broadcast a message.
func (c *Comet) Broadcast(arg *comet.BroadcastReq) (err error) {
	c.broadcastChan <- arg
	return
}

func (c *Comet) process(pushChan chan *comet.PushMsgReq, roomChan chan *comet.BroadcastRoomReq, broadcastChan chan *comet.BroadcastReq) {
	for {
		select {
		case broadcastArg := <-broadcastChan:
			_, err := c.client.Broadcast(context.Background(), &comet.BroadcastReq{
				Proto:   broadcastArg.Proto,
				ProtoOp: broadcastArg.ProtoOp,
				Speed:   broadcastArg.Speed,
			})
			if err != nil {
				log.Errorf("c.client.Broadcast(%s, reply) serverId:%s error(%v)", broadcastArg, c.serverID, err)
			}
		case roomArg := <-roomChan:
			_, err := c.client.BroadcastRoom(context.Background(), &comet.BroadcastRoomReq{
				RoomID: roomArg.RoomID,
				Proto:  roomArg.Proto,
			})
			if err != nil {
				log.Errorf("c.client.BroadcastRoom(%s, reply) serverId:%s error(%v)", roomArg, c.serverID, err)
			}
		case pushArg := <-pushChan:
			_, err := c.client.PushMsg(context.Background(), &comet.PushMsgReq{
				Keys:    pushArg.Keys,
				Proto:   pushArg.Proto,
				ProtoOp: pushArg.ProtoOp,
			})
			if err != nil {
				log.Errorf("c.client.PushMsg(%s, reply) serverId:%s error(%v)", pushArg, c.serverID, err)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Close close the resources.
func (c *Comet) Close() (err error) {
	finish := make(chan bool)
	go func() {
		for {
			n := len(c.broadcastChan)
			for _, ch := range c.pushChan {
				n += len(ch)
			}
			for _, ch := range c.roomChan {
				n += len(ch)
			}
			if n == 0 {
				finish <- true
				return
			}
			time.Sleep(time.Second)
		}
	}()
	select {
	case <-finish:
		log.Info("close comet finish")
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("close comet(server:%s push:%d room:%d broadcast:%d) timeout", c.serverID, len(c.pushChan), len(c.roomChan), len(c.broadcastChan))
	}
	c.cancel()
	return
}
