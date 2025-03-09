package comet

import (
	"context"
	"io"
	"net"
	"strings"
	"time"

	log "github.com/golang/glog"
	"github.com/gyy0727/mygoim/api/protocol"
	"github.com/gyy0727/mygoim/internal/comet/conf"
	"github.com/gyy0727/mygoim/pkg/bufio"
	"github.com/gyy0727/mygoim/pkg/bytes"
	xtime "github.com/gyy0727/mygoim/pkg/time"
)

const (
	maxInt = 1<<31 - 1
)

// *初始化一个tcpserver并开启监听
func InitTCP(server *Server, addrs []string, accept int) (err error) {
	var (
		bind     string           //*绑定的地址
		listener *net.TCPListener //*TCP 监听器，用于接受客户端连接
		addr     *net.TCPAddr     //*解析后的 TCP 地址，包含 IP 和端口信息
	)
	for _, bind = range addrs {
		//*解析地址，返回 *net.TCPAddr
		if addr, err = net.ResolveTCPAddr("tcp", bind); err != nil {
			log.Errorf("net.ResolveTCPAddr(tcp, %s) error(%v)", bind, err)
			return
		}
		//*在解析后的地址上创建 TCP 监听器
		if listener, err = net.ListenTCP("tcp", addr); err != nil {
			log.Errorf("net.ListenTCP(tcp, %s) error(%v)", bind, err)
			return
		}
		log.Infof("start tcp listen: %s", bind)
		//*accept每个地址启动的 goroutine 数量，用于并发处理客户端连接
		for i := 0; i < accept; i++ {
			go acceptTCP(server, listener)
		}
	}
	return
}

// *接受客户端 TCP 连接，并设置连接的参数（如 KeepAlive、读写缓冲区大小），然后启动 serveTCP 处理连接
func acceptTCP(server *Server, lis *net.TCPListener) {
	var (
		conn *net.TCPConn //*连接
		err  error
		r    int
	)
	for {
		if conn, err = lis.AcceptTCP(); err != nil {

			log.Errorf("listener.Accept(\"%s\") error(%v)", lis.Addr().String(), err)
			return
		}
		if err = conn.SetKeepAlive(server.c.TCP.KeepAlive); err != nil {
			log.Errorf("conn.SetKeepAlive() error(%v)", err)
			return
		}
		if err = conn.SetReadBuffer(server.c.TCP.Rcvbuf); err != nil {
			log.Errorf("conn.SetReadBuffer() error(%v)", err)
			return
		}
		if err = conn.SetWriteBuffer(server.c.TCP.Sndbuf); err != nil {
			log.Errorf("conn.SetWriteBuffer() error(%v)", err)
			return
		}
		go serveTCP(server, conn, r)
		if r++; r == maxInt {
			r = 0
		}
	}
}

// *分配读写缓冲区和定时器
func serveTCP(s *Server, conn *net.TCPConn, r int) {
	var (
		tr    = s.round.Timer(r)
		rp    = s.round.Reader(r)
		wp    = s.round.Writer(r)
		lAddr = conn.LocalAddr().String()  //*本地地址
		rAddr = conn.RemoteAddr().String() //*远端地址
	)
	if conf.Conf.Debug {
		log.Infof("start tcp serve \"%s\" with \"%s\"", lAddr, rAddr)
	}
	s.ServeTCP(conn, rp, wp, tr)
}

func (s *Server) ServeTCP(conn *net.TCPConn, rp, wp *bytes.Pool, tr *xtime.Timer) {
	var (
		err     error                                                      //*错误信息
		rid     string                                                     //*客户端唯一标识
		accepts []int32                                                    //*客户端订阅的频道列表
		hb      time.Duration                                              //*心跳超时
		white   bool                                                       //*是否在白名单
		p       *protocol.Proto                                            //*协议消息
		b       *Bucket                                                    //*所属的bucket
		trd     *xtime.TimerData                                           //*定时器数据
		lastHb  = time.Now()                                               //*上次心跳时间
		rb      = rp.Get()                                                 //*读缓冲区
		wb      = wp.Get()                                                 //*写缓冲区
		ch      = NewChannel(s.c.Protocol.CliProto, s.c.Protocol.SvrProto) //*客户端 Channel，用于管理客户端连接状态
		rr      = &ch.Reader                                               //*读缓冲区的 Reader
		wr      = &ch.Writer                                               //*写缓冲区的 Writer
	)
	ch.Reader.ResetBuffer(conn, rb.Bytes())
	ch.Writer.ResetBuffer(conn, wb.Bytes())
	//*创建上下文，用于控制 goroutine 的生命周期。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//*作用：设置握手超时定时器，并记录客户端 IP
	step := 0
	trd = tr.Add(time.Duration(s.c.Protocol.HandshakeTimeout), func() {
		conn.Close()
		log.Errorf("key: %s remoteIP: %s step: %d tcp handshake timeout", ch.Key, conn.RemoteAddr().String(), step)
	})
	//*解析客户端地址，获取 IP
	ch.IP, _, _ = net.SplitHostPort(conn.RemoteAddr().String())
	//*进行客户端认证，并初始化客户端连接
	step = 1
	if p, err = ch.CliProto.Set(); err == nil {
		if ch.Mid, ch.Key, rid, accepts, hb, err = s.authTCP(ctx, rr, wr, p); err == nil {
			ch.Watch(accepts...)
			b = s.Bucket(ch.Key)
			err = b.Put(rid, ch)
			if conf.Conf.Debug {
				log.Infof("tcp connnected key:%s mid:%d proto:%+v", ch.Key, ch.Mid, p)
			}
		}
	}
	step = 2
	if err != nil {
		conn.Close()
		rp.Put(rb)
		wp.Put(wb)
		tr.Del(trd)
		log.Errorf("key: %s handshake failed error(%v)", ch.Key, err)
		return
	}
	trd.Key = ch.Key
	tr.Set(trd, hb)
	white = whitelist.Contains(ch.Mid)
	if white {
		whitelist.Printf("key: %s[%s] auth\n", ch.Key, rid)
	}
	step = 3
	// hanshake ok start dispatch goroutine
	go s.dispatchTCP(conn, wr, wp, wb, ch)
	serverHeartbeat := s.RandServerHearbeat()
	for {
		if p, err = ch.CliProto.Set(); err != nil {
			break
		}
		if white {
			whitelist.Printf("key: %s start read proto\n", ch.Key)
		}
		if err = p.ReadTCP(rr); err != nil {
			break
		}
		if white {
			whitelist.Printf("key: %s read proto:%v\n", ch.Key, p)
		}
		if p.Op == protocol.OpHeartbeat {
			tr.Set(trd, hb)
			p.Op = protocol.OpHeartbeatReply
			p.Body = nil
			// NOTE: send server heartbeat for a long time
			if now := time.Now(); now.Sub(lastHb) > serverHeartbeat {
				if err1 := s.Heartbeat(ctx, ch.Mid, ch.Key); err1 == nil {
					lastHb = now
				}
			}
			if conf.Conf.Debug {
				log.Infof("tcp heartbeat receive key:%s, mid:%d", ch.Key, ch.Mid)
			}
			step++
		} else {
			if err = s.Operate(ctx, p, ch, b); err != nil {
				break
			}
		}
		if white {
			whitelist.Printf("key: %s process proto:%v\n", ch.Key, p)
		}
		ch.CliProto.SetAdv()
		ch.Signal()
		if white {
			whitelist.Printf("key: %s signal\n", ch.Key)
		}
	}
	if white {
		whitelist.Printf("key: %s server tcp error(%v)\n", ch.Key, err)
	}
	if err != nil && err != io.EOF && !strings.Contains(err.Error(), "closed") {
		log.Errorf("key: %s server tcp failed error(%v)", ch.Key, err)
	}
	b.Del(ch)
	tr.Del(trd)
	rp.Put(rb)
	conn.Close()
	ch.Close()
	if err = s.Disconnect(ctx, ch.Mid, ch.Key); err != nil {
		log.Errorf("key: %s mid: %d operator do disconnect error(%v)", ch.Key, ch.Mid, err)
	}
	if white {
		whitelist.Printf("key: %s mid: %d disconnect error(%v)\n", ch.Key, ch.Mid, err)
	}
	if conf.Conf.Debug {
		log.Infof("tcp disconnected key: %s mid: %d", ch.Key, ch.Mid)
	}
}

//*分发 TCP 消息，负责将消息写入客户端连接
func (s *Server) dispatchTCP(conn *net.TCPConn, wr *bufio.Writer, wp *bytes.Pool, wb *bytes.Buffer, ch *Channel) {
	var (
		err    error
		finish bool
		online int32
		white  = whitelist.Contains(ch.Mid)
	)
	if conf.Conf.Debug {
		log.Infof("key: %s start dispatch tcp goroutine", ch.Key)
	}
	for {
		if white {
			whitelist.Printf("key: %s wait proto ready\n", ch.Key)
		}
		var p = ch.Ready()
		if white {
			whitelist.Printf("key: %s proto ready\n", ch.Key)
		}
		if conf.Conf.Debug {
			log.Infof("key:%s dispatch msg:%v", ch.Key, *p)
		}
		switch p {
		case protocol.ProtoFinish:
			if white {
				whitelist.Printf("key: %s receive proto finish\n", ch.Key)
			}
			if conf.Conf.Debug {
				log.Infof("key: %s wakeup exit dispatch goroutine", ch.Key)
			}
			finish = true
			goto failed
		case protocol.ProtoReady:
			// fetch message from svrbox(client send)
			for {
				if p, err = ch.CliProto.Get(); err != nil {
					break
				}
				if white {
					whitelist.Printf("key: %s start write client proto%v\n", ch.Key, p)
				}
				if p.Op == protocol.OpHeartbeatReply {
					if ch.Room != nil {
						online = ch.Room.OnlineNum()
					}
					if err = p.WriteTCPHeart(wr, online); err != nil {
						goto failed
					}
				} else {
					if err = p.WriteTCP(wr); err != nil {
						goto failed
					}
				}
				if white {
					whitelist.Printf("key: %s write client proto%v\n", ch.Key, p)
				}
				p.Body = nil // avoid memory leak
				ch.CliProto.GetAdv()
			}
		default:
			if white {
				whitelist.Printf("key: %s start write server proto%v\n", ch.Key, p)
			}
			// server send
			if err = p.WriteTCP(wr); err != nil {
				goto failed
			}
			if white {
				whitelist.Printf("key: %s write server proto%v\n", ch.Key, p)
			}
			if conf.Conf.Debug {
				log.Infof("tcp sent a message key:%s mid:%d proto:%+v", ch.Key, ch.Mid, p)
			}
		}
		if white {
			whitelist.Printf("key: %s start flush \n", ch.Key)
		}
		// only hungry flush response
		if err = wr.Flush(); err != nil {
			break
		}
		if white {
			whitelist.Printf("key: %s flush\n", ch.Key)
		}
	}
failed:
	if white {
		whitelist.Printf("key: %s dispatch tcp error(%v)\n", ch.Key, err)
	}
	if err != nil {
		log.Errorf("key: %s dispatch tcp error(%v)", ch.Key, err)
	}
	conn.Close()
	wp.Put(wb)
	// must ensure all channel message discard, for reader won't blocking Signal
	for !finish {
		finish = (ch.Ready() == protocol.ProtoFinish)
	}
	if conf.Conf.Debug {
		log.Infof("key: %s dispatch goroutine exit", ch.Key)
	}
}


func (s *Server) authTCP(ctx context.Context, rr *bufio.Reader, wr *bufio.Writer, p *protocol.Proto) (mid int64, key, rid string, accepts []int32, hb time.Duration, err error) {
	for {
		if err = p.ReadTCP(rr); err != nil {
			return
		}
		if p.Op == protocol.OpAuth {
			break
		} else {
			log.Errorf("tcp request operation(%d) not auth", p.Op)
		}
	}
	if mid, key, rid, accepts, hb, err = s.Connect(ctx, p, ""); err != nil {
		log.Errorf("authTCP.Connect(key:%v).err(%v)", key, err)
		return
	}
	p.Op = protocol.OpAuthReply
	p.Body = nil
	if err = p.WriteTCP(wr); err != nil {
		log.Errorf("authTCP.WriteTCP(key:%v).err(%v)", key, err)
		return
	}
	err = wr.Flush()
	return
}
