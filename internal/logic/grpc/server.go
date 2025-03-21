package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/gyy0727/mygoim/api/logic"
	"github.com/gyy0727/mygoim/internal/logic"
	"github.com/gyy0727/mygoim/internal/logic/conf"
	discovery "github.com/gyy0727/mygoim/pkg/discovery"
	ip "github.com/gyy0727/mygoim/pkg/ip"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
)

// New logic grpc server
func New(c *conf.RPCServer, l *logic.Logic) *grpc.Server {
	keepParams := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Duration(c.IdleTimeout),
		MaxConnectionAgeGrace: time.Duration(c.ForceCloseWait),
		Time:                  time.Duration(c.KeepAliveInterval),
		Timeout:               time.Duration(c.KeepAliveTimeout),
		MaxConnectionAge:      time.Duration(c.MaxLifeTime),
	})
	addr, _ := ip.GetPublicIP()
	fmt.Println("logic grpc server listen on %s", addr)
	node := &discovery.Node{
		Name: "goim.logic",
		Addr: addr,
	}
	discovery.ERegister.AddServiceNode(node)
	srv := grpc.NewServer(keepParams)
	pb.RegisterLogicServer(srv, &server{l})
	lis, err := net.Listen(c.Network, c.Addr)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := srv.Serve(lis); err != nil {
			panic(err)
		}
	}()
	return srv
}

type server struct {
	srv *logic.Logic
}

var _ pb.LogicServer = &server{}

// Connect connect a conn.
func (s *server) Connect(ctx context.Context, req *pb.ConnectReq) (*pb.ConnectReply, error) {
	mid, key, room, accepts, hb, err := s.srv.Connect(ctx, req.Server, req.Cookie, req.Token)
	if err != nil {
		return &pb.ConnectReply{}, err
	}
	return &pb.ConnectReply{Mid: mid, Key: key, RoomID: room, Accepts: accepts, Heartbeat: hb}, nil
}

// Disconnect disconnect a conn.
func (s *server) Disconnect(ctx context.Context, req *pb.DisconnectReq) (*pb.DisconnectReply, error) {
	has, err := s.srv.Disconnect(ctx, req.Mid, req.Key, req.Server)
	if err != nil {
		return &pb.DisconnectReply{}, err
	}
	return &pb.DisconnectReply{Has: has}, nil
}

// Heartbeat beartbeat a conn.
func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatReq) (*pb.HeartbeatReply, error) {
	if err := s.srv.Heartbeat(ctx, req.Mid, req.Key, req.Server); err != nil {
		return &pb.HeartbeatReply{}, err
	}
	return &pb.HeartbeatReply{}, nil
}

// RenewOnline renew server online.
func (s *server) RenewOnline(ctx context.Context, req *pb.OnlineReq) (*pb.OnlineReply, error) {
	allRoomCount, err := s.srv.RenewOnline(ctx, req.Server, req.RoomCount)
	if err != nil {
		return &pb.OnlineReply{}, err
	}
	return &pb.OnlineReply{AllRoomCount: allRoomCount}, nil
}

// Receive receive a message.
func (s *server) Receive(ctx context.Context, req *pb.ReceiveReq) (*pb.ReceiveReply, error) {
	if err := s.srv.Receive(ctx, req.Mid, req.Proto); err != nil {
		return &pb.ReceiveReply{}, err
	}
	return &pb.ReceiveReply{}, nil
}

// nodes return nodes.
func (s *server) Nodes(ctx context.Context, req *pb.NodesReq) (*pb.NodesReply, error) {
	// return s.srv.NodesWeighted(ctx, req.Platform, req.ClientIP), nil
	return nil, nil
}
