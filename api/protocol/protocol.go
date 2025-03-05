package protocol

import (
	"errors"
	"github.com/gyy0727/mygoim/pkg/bufio"
	"github.com/gyy0727/mygoim/pkg/bytes"
	"github.com/gyy0727/mygoim/pkg/endian/binary"
)

const (
	//*最大的消息体大小 4096
	MaxBodySize = int32(1 << 12)
)

const (
	// size
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
		bodyLen   int
		headerLen int16
		packLen   int32
		buf       []byte
	)
	if buf, err = rr.Pop(_rawHeaderSize); err != nil {
		return
	}
	packLen = binary.BigEndian.Int32(buf[_packOffset:_headerOffset])
	headerLen = binary.BigEndian.Int16(buf[_headerOffset:_verOffset])
	p.Ver = int32(binary.BigEndian.Int16(buf[_verOffset:_opOffset]))
	p.Op = binary.BigEndian.Int32(buf[_opOffset:_seqOffset])
	p.Seq = binary.BigEndian.Int32(buf[_seqOffset:])
	if packLen > _maxPackSize {
		return ErrProtoPackLen
	}
	if headerLen != _rawHeaderSize {
		return ErrProtoHeaderLen
	}
	if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
		p.Body, err = rr.Pop(bodyLen)
	} else {
		p.Body = nil
	}
	return
}
