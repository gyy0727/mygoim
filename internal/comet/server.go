package comet

import (
	"context"
	"math/rand"
	"time"
	"github.com/gyy0727/mygoim/api/logic"
	"github.com/gyy0727/mygoim/internal/comet/conf"
	"github.com/zhenjl/cityhash"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	minServerHeartbeat        = time.Minute * 10 //*服务器最小心跳超时
	maxServerHeartbeat        = time.Minute * 30 //*服务器最大心跳超时
	grpcInitialWindowSize     = 1 << 24          //*gRPC单连接初始流量控制窗口（默认16MB）
	grpcInitialConnWindowSize = 1 << 24          //*gRPC全局初始流量控制窗口（默认16MB）
	grpcMaxSendMsgSize        = 1 << 24          //*单条发送消息最大尺寸限制（16MB）
	grpcMaxCallMsgSize        = 1 << 24          //*单条调用消息最大尺寸限制（16MB）
	grpcKeepAliveTime         = time.Second * 10 //*gRPC保活探测间隔（10秒发送一次ping）
	grpcKeepAliveTimeout      = time.Second * 3  //* gRPC保活超时时间（3秒未响应视为连接断开）
	grpcBackoffMaxDelay       = time.Second * 3  //*gRPC连接重试最大退避延迟（指数退避上限3秒）
)

// *新建一个rpc客户端
func newLogicClient(c *conf.RPCClient) logic.LogicClient {
	//*TODO 
	// return nil
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Dial))
	defer cancel()
	conn, err := grpc.DialContext(ctx, "etcd:///goim.logic",
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
			grpc.WithDefaultServiceConfig(`{
                "loadBalancingConfig": [{"round_robin": {}}]
            }`),
		}...)
	if err != nil {
		panic(err)
	}
	return logic.NewLogicClient(conn)
}

//*comet-logic服务器
type Server struct {
	c         *conf.Config      //*配置文件对象
	round     *Round            //*池管理器（如连接池管理）
	buckets   []*Bucket         //*分桶存储结构（用于负载均衡或资源分区）
	bucketIdx uint32            //*当前服务器包含的bucket的数量
	serverID  string            //*服务实例唯一标识
	rpcClient logic.LogicClient //*gRPC客户端接口
}

// *新建一个server
func NewServer(c *conf.Config) *Server {
	s := &Server{
		c:         c,
		round:     NewRound(c),
		rpcClient: newLogicClient(c.RPCClient),
	}
	s.buckets = make([]*Bucket, c.Bucket.Size)
	s.bucketIdx = uint32(c.Bucket.Size)
	
	for i := 0; i < c.Bucket.Size; i++ {
		s.buckets[i] = NewBucket(c.Bucket)
	}
	s.serverID = c.Env.Host //*当前主机的主机名
	logger.Info("初始化server",zap.Int("bucket size",c.Bucket.Size))
	go s.onlineproc()
	return s
}

// *返回server的Buckets
func (s *Server) Buckets() []*Bucket {
	return s.buckets
}

//*根据subkey的值得出bucket的位置 
func (s *Server) Bucket(subKey string) *Bucket {
	
	idx := cityhash.CityHash32([]byte(subKey), uint32(len(subKey))) % s.bucketIdx
	if conf.Conf.Debug {
		logger.Info("hit channel bucket",
			zap.String("subKey", subKey),
			zap.Uint32("index", idx),
			zap.String("hash", "cityhash"),
		)
	}
	return s.buckets[idx]
}

//*返回一个随机的 time.Duration 值，表示服务器的心跳时间
func (s *Server) RandServerHearbeat() time.Duration {
	return (minServerHeartbeat + time.Duration(rand.Int63n(int64(maxServerHeartbeat-minServerHeartbeat))))
}

//*关闭server 
func (s *Server) Close() (err error) {
	return
}

//*server的主函数, 用于处理在线人数的更新
func (s *Server) onlineproc() {
	logger.Info("onlineproc执行中")
	for {
		var (
			allRoomsCount map[string]int32
			err           error
		)
		roomCount := make(map[string]int32)
		for _, bucket := range s.buckets {
			for roomID, count := range bucket.RoomsCount() {
				roomCount[roomID] += count
			}
		}
		//*将在线人数存储到redis中
		if allRoomsCount, err = s.RenewOnline(context.Background(), s.serverID, roomCount); err != nil {
			time.Sleep(time.Second)
			continue
		}
		for _, bucket := range s.buckets {
			bucket.UpRoomsCount(allRoomsCount)
		}
		time.Sleep(time.Second * 10)
	}
}
