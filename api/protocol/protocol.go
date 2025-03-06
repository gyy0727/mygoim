package protocol

import (
	"errors"

	"github.com/gyy0727/mygoim/pkg/bufio"
	"github.com/gyy0727/mygoim/pkg/bytes"
	"github.com/gyy0727/mygoim/pkg/endian/binary"
	"github.com/gyy0727/mygoim/pkg/websocket"
)

const (
	//*最大的消息体大小 4096
	MaxBodySize = int32(1 << 12)
)

const (
	_packSize      = 4                                                       //*协议包长度字段的大小
	_headerSize    = 2                                                       //*协议头长度字段的大小
	_verSize       = 2                                                       //*协议版本字段的大小
	_opSize        = 4                                                       //*协议操作码字段的大小
	_seqSize       = 4                                                       //*协议序列号字段的大小
	_heartSize     = 4                                                       //*心跳数据字段的大小
	_rawHeaderSize = _packSize + _headerSize + _verSize + _opSize + _seqSize //*协议头的总大小
	_maxPackSize   = MaxBodySize + int32(_rawHeaderSize)                     //*协议包的最大大小
	_packOffset    = 0                                                       //*协议包长度字段的偏移量
	_headerOffset  = _packOffset + _packSize                                 //*协议头长度字段的偏移量
	_verOffset     = _headerOffset + _headerSize                             //*协议版本字段的偏移量
	_opOffset      = _verOffset + _verSize                                   //*协议操作码字段的偏移量
	_seqOffset     = _opOffset + _opSize                                     //*协议序列号字段的偏移量
	_heartOffset   = _seqOffset + _seqSize                                   //*心跳数据字段的偏移量
)

var (
	//*用于表示协议包长度解析或验证时发生的错误
	ErrProtoPackLen = errors.New("default server codec pack length error")
	//*协议头长度解析或验证时发生的错误
	ErrProtoHeaderLen = errors.New("default server codec header length error")
)

var (
	//*准备就绪
	ProtoReady = &Proto{Op: OpProtoReady}
	//*完成
	ProtoFinish = &Proto{Op: OpProtoFinish}
)

// *用于将协议包写入到 bytes.Writer 中
func (p *Proto) WriteTo(b *bytes.Writer) {
	var (
		packLen = _rawHeaderSize + int32(len(p.Body))
		buf     = b.Peek(_rawHeaderSize)
	)
	binary.BigEndian.PutInt32(buf[_packOffset:], packLen)
	binary.BigEndian.PutInt16(buf[_headerOffset:], int16(_rawHeaderSize))
	binary.BigEndian.PutInt16(buf[_verOffset:], int16(p.Ver))
	binary.BigEndian.PutInt32(buf[_opOffset:], p.Op)
	binary.BigEndian.PutInt32(buf[_seqOffset:], p.Seq)
	if p.Body != nil {
		b.Write(p.Body)
	}
}

func (p *Proto) ReadTCP(rr *bufio.Reader) (err error) {
	var (
		bodyLen   int    //*消息体长度
		headerLen int16  //*消息头长度
		packLen   int32  //* 协议包总长度
		buf       []byte //*缓冲区
	)
	//*使用 rr.Pop 方法从缓冲区中读取 _rawHeaderSize 字节的数据到 buf 中
	if buf, err = rr.Pop(_rawHeaderSize); err != nil {
		return
	}
	//*从 buf 中解析协议包的总长度 packLen，使用大端序（BigEndian）解码
	packLen = binary.BigEndian.Int32(buf[_packOffset:_headerOffset])
	//*从 buf 中解析消息头的长度 headerLen，使用大端序解码
	headerLen = binary.BigEndian.Int16(buf[_headerOffset:_verOffset])
	//*从 buf 中解析协议版本 p.Ver，使用大端序解码
	p.Ver = int32(binary.BigEndian.Int16(buf[_verOffset:_opOffset]))
	//*从 buf 中解析操作码 p.Op，使用大端序解码
	p.Op = binary.BigEndian.Int32(buf[_opOffset:_seqOffset])
	//*从 buf 中解析序列号 p.Seq，使用大端序解码
	p.Seq = binary.BigEndian.Int32(buf[_seqOffset:])
	//*如果协议包的总长度 packLen 超过最大允许长度
	if packLen > _maxPackSize {
		return ErrProtoPackLen
	}
	//*如果消息头的长度 headerLen 不等于预期的 _rawHeaderSize
	if headerLen != _rawHeaderSize {
		return ErrProtoHeaderLen
	}
	//*计算消息体的长度 bodyLen，即协议包总长度减去消息头长度
	if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
		p.Body, err = rr.Pop(bodyLen)
	} else {
		p.Body = nil
	}
	return
}

// *用于将协议包数据写入到 bufio.Writer 中
func (p *Proto) WriteTCP(wr *bufio.Writer) (err error) {
	var (
		buf     []byte
		packLen int32
	)
	//*如果操作码是 OpRaw，表示这是一个原始数据包，直接写入消息体 p.Body，无需添加协议
	if p.Op == OpRaw {
		_, err = wr.WriteRaw(p.Body)
		return
	}
	packLen = _rawHeaderSize + int32(len(p.Body))
	if buf, err = wr.Peek(_rawHeaderSize); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[_packOffset:], packLen)
	binary.BigEndian.PutInt16(buf[_headerOffset:], int16(_rawHeaderSize))
	binary.BigEndian.PutInt16(buf[_verOffset:], int16(p.Ver))
	binary.BigEndian.PutInt32(buf[_opOffset:], p.Op)
	binary.BigEndian.PutInt32(buf[_seqOffset:], p.Seq)
	if p.Body != nil {
		_, err = wr.Write(p.Body)
	}
	return
}

// *用于写入 TCP 心跳包，并携带房间在线人数信息
func (p *Proto) WriteTCPHeart(wr *bufio.Writer, online int32) (err error) {
	var (
		buf     []byte
		packLen int
	)
	packLen = _rawHeaderSize + _heartSize
	if buf, err = wr.Peek(packLen); err != nil {
		return
	}

	binary.BigEndian.PutInt32(buf[_packOffset:], int32(packLen))
	binary.BigEndian.PutInt16(buf[_headerOffset:], int16(_rawHeaderSize))
	binary.BigEndian.PutInt16(buf[_verOffset:], int16(p.Ver))
	binary.BigEndian.PutInt32(buf[_opOffset:], p.Op)
	binary.BigEndian.PutInt32(buf[_seqOffset:], p.Seq)
	binary.BigEndian.PutInt32(buf[_heartOffset:], online)
	return
}

// *从websocket连接中读取消息
func (p *Proto) ReadWebsocket(ws *websocket.Conn) (err error) {
	var (
		bodyLen   int    //*消息体长度
		headerLen int16  //*头部长度
		packLen   int32  //*总长度
		buf       []byte //*缓冲区
	)
	//*读出消息  
	if _, buf, err = ws.ReadMessage(); err != nil {
		return
	}
	//*如果总长度加起来还没头部的固定长度长,证明消息没读取完整 
	if len(buf) < _rawHeaderSize {
		return ErrProtoPackLen
	}
	//*解析总长度 
	packLen = binary.BigEndian.Int32(buf[_packOffset:_headerOffset])
	//*解析请求头长度
	headerLen = binary.BigEndian.Int16(buf[_headerOffset:_verOffset])
	//*解析请求版本  
	p.Ver = int32(binary.BigEndian.Int16(buf[_verOffset:_opOffset]))
	//*解析请求操作码
	p.Op = binary.BigEndian.Int32(buf[_opOffset:_seqOffset])
	//*解析请求序列号
	p.Seq = binary.BigEndian.Int32(buf[_seqOffset:])
	if packLen < 0 || packLen > _maxPackSize {
		return ErrProtoPackLen
	}
	if headerLen != _rawHeaderSize {
		return ErrProtoHeaderLen
	}
	if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
		p.Body = buf[headerLen:packLen]
	} else {
		p.Body = nil
	}
	return
}


//*发送消息
func (p *Proto) WriteWebsocket(ws *websocket.Conn) (err error) {
	var (
		buf     []byte
		packLen int
	)
	//*头部长度+请求体长度 
	packLen = _rawHeaderSize + len(p.Body)
	if err = ws.WriteHeader(websocket.BinaryMessage, packLen); err != nil {
		return
	}
	if buf, err = ws.Peek(_rawHeaderSize); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[_packOffset:], int32(packLen))
	binary.BigEndian.PutInt16(buf[_headerOffset:], int16(_rawHeaderSize))
	binary.BigEndian.PutInt16(buf[_verOffset:], int16(p.Ver))
	binary.BigEndian.PutInt32(buf[_opOffset:], p.Op)
	binary.BigEndian.PutInt32(buf[_seqOffset:], p.Seq)
	if p.Body != nil {
		err = ws.WriteBody(p.Body)
	}
	return
}

//*发送心跳消息 
func (p *Proto) WriteWebsocketHeart(wr *websocket.Conn, online int32) (err error) {
	var (
		buf     []byte
		packLen int
	)
	packLen = _rawHeaderSize + _heartSize
	
	if err = wr.WriteHeader(websocket.BinaryMessage, packLen); err != nil {
		return
	}
	if buf, err = wr.Peek(packLen); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[_packOffset:], int32(packLen))
	binary.BigEndian.PutInt16(buf[_headerOffset:], int16(_rawHeaderSize))
	binary.BigEndian.PutInt16(buf[_verOffset:], int16(p.Ver))
	binary.BigEndian.PutInt32(buf[_opOffset:], p.Op)
	binary.BigEndian.PutInt32(buf[_seqOffset:], p.Seq)
	binary.BigEndian.PutInt32(buf[_heartOffset:], online)
	return
}
