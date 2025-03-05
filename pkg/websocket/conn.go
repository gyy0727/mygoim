package websocket

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/gyy0727/mygoim/pkg/bufio"
)

const (
	finBit                   = 1 << 7 //*表示是否为最后一帧
	rsv1Bit                  = 1 << 6 //*保留位
	rsv2Bit                  = 1 << 5 //*保留位
	rsv3Bit                  = 1 << 4 //*保留位
	opBit                    = 0x0f   //*操作码掩码
	maskBit                  = 1 << 7 //*表示是否使用掩码
	lenBit                   = 0x7f   //*负载长度掩码
	continuationFrame        = 0
	continuationFrameMaxRead = 100
)

const (
	//*文本消息
	TextMessage = 1

	//*二进制消息
	BinaryMessage = 2

	//*关闭消息
	CloseMessage = 8

	//*ping消息
	PingMessage = 9

	//*Pong消息
	PongMessage = 10
)

var (
	//*表示接收到关闭控制消息（CloseMessage）时的错误
	//*当 WebSocket 连接接收到关闭消息时，可以返回此错误以通知调用方连接即将关闭
	ErrMessageClose = errors.New("close control message")
	//*表示在读取连续帧（continuation frame）时超出了最大允许读取大小的错误
	ErrMessageMaxRead = errors.New("continuation frame max read")
)

type Conn struct {
	rwc io.ReadWriteCloser //*表示底层的网络连接。它提供了读取、写入和关闭连接的基本功能
	r   *bufio.Reader      //*用于缓冲读取数据。通过缓冲读取，可以提高读取效率，减少系统调用的次数
	w   *bufio.Writer      //*用于缓冲写入数据。通过缓冲写入，可以提高写入效率，减少系统调用的次数
	//*用于存储 WebSocket 帧的掩码密钥
	//*根据 WebSocket 协议，客户端发送的帧必须使用掩码密钥对数据进行掩码处理
	//*而服务器发送的帧则不需要.
	maskKey []byte
}

// *新建连接
func newConn(rwc io.ReadWriteCloser, r *bufio.Reader, w *bufio.Writer) *Conn {
	return &Conn{rwc: rwc, r: r, w: w, maskKey: make([]byte, 4)}
}

// *用于将指定类型的消息写入 WebSocket 连接
func (c *Conn) WriteMessage(msgType int, msg []byte) (err error) {
	if err = c.WriteHeader(msgType, len(msg)); err != nil {
		return
	}
	err = c.WriteBody(msg)
	return
}

// *用于写入 WebSocket 帧的头部
func (c *Conn) WriteHeader(msgType int, length int) (err error) {
	var h []byte
	//*使用 Peek 方法预取 2 字节的缓冲区，用于写入帧头部
	if h, err = c.w.Peek(2); err != nil {
		return
	}

	//*将 FIN 位和 OpCode 写入第一个字节
	h[0] = 0
	h[0] |= finBit | byte(msgType)
	//*表示是否使用掩码（通常为 0，因为服务器发送的帧不需要掩码）
	h[1] = 0
	//*根据消息体长度的大小，选择不同的编码方式
	switch {
	//*直接使用 7 位表示
	case length <= 125:
		h[1] |= byte(length)

	case length < 65536:
		//*使用 16 位表示，并写入额外的 2 字节
		h[1] |= 126
		if h, err = c.w.Peek(2); err != nil {
			return
		}
		binary.BigEndian.PutUint16(h, uint16(length))
	default:
		//*使用 64 位表示，并写入额外的 8 字节
		h[1] |= 127
		if h, err = c.w.Peek(8); err != nil {
			return
		}
		binary.BigEndian.PutUint64(h, uint64(length))
	}
	return
}

// *写入消息体
func (c *Conn) WriteBody(b []byte) (err error) {
	if len(b) > 0 {
		_, err = c.w.Write(b)
	}
	return
}

func (c *Conn) Peek(n int) ([]byte, error) {
	return c.w.Peek(n)
}

func (c *Conn) Flush() error {
	return c.w.Flush()
}

// *读取消息
func (c *Conn) ReadMessage() (op int, payload []byte, err error) {
	var (
		fin         bool   //*是否是最后一帧
		finOp, n    int    //*记录消息的最终类型,已读取的帧数
		partPayload []byte //*当前帧的消息体数据
	)
	//*进入循环，逐帧读取消息
	for {

		if fin, op, partPayload, err = c.readFrame(); err != nil {
			return
		}
		switch op {
		case BinaryMessage, TextMessage, continuationFrame:
			if fin && len(payload) == 0 {
				return op, partPayload, nil
			}
			// continuation frame
			payload = append(payload, partPayload...)
			if op != continuationFrame {
				finOp = op
			}
			// final frame
			if fin {
				op = finOp
				return
			}
		case PingMessage:
			// handler ping
			if err = c.WriteMessage(PongMessage, partPayload); err != nil {
				return
			}
		case PongMessage:
			// handler pong
		case CloseMessage:
			// handler close
			err = ErrMessageClose
			return
		default:
			err = fmt.Errorf("unknown control message, fin=%t, op=%d", fin, op)
			return
		}
		if n > continuationFrameMaxRead {
			err = ErrMessageMaxRead
			return
		}
		n++
	}
}

// *用于从 WebSocket 连接中读取一个完整的帧
func (c *Conn) readFrame() (fin bool, op int, payload []byte, err error) {
	var (
		b          byte   //*临时存储读取的字节
		p          []byte //*临时存储读取的字节数组
		mask       bool   //*是否使用掩码
		maskKey    []byte //*掩码密钥
		payloadLen int64  //*消息体的长度
	)
	//*先读取一个字节 
	b, err = c.r.ReadByte()
	if err != nil {
		return
	}
	//*是否是最后一帧 
	fin = (b & finBit) != 0
	//*检查 RSV 位，确保其值为 0（根据 WebSocket 协议，RSV 位必须为 0）
	if rsv := b & (rsv1Bit | rsv2Bit | rsv3Bit); rsv != 0 {
		return false, 0, nil, fmt.Errorf("unexpected reserved bits rsv1=%d, rsv2=%d, rsv3=%d", b&rsv1Bit, b&rsv2Bit, b&rsv3Bit)
	}
	//*解析操作位 
	op = int(b & opBit)
	//*读取第二个字节 
	b, err = c.r.ReadByte()
	if err != nil {
		return
	}
	//*是否使用掩码 
	mask = (b & maskBit) != 0
	//*解析消息体长度
	switch b & lenBit {
	case 126:
		//*16位 
		if p, err = c.r.Pop(2); err != nil {
			return
		}
		payloadLen = int64(binary.BigEndian.Uint16(p))
	case 127:
		//*64 bits
		if p, err = c.r.Pop(8); err != nil {
			return
		}
		payloadLen = int64(binary.BigEndian.Uint64(p))
	default:
		//*7 bits
		payloadLen = int64(b & lenBit)
	}
	//*读取掩码密钥
	if mask {
		maskKey, err = c.r.Pop(4)
		if err != nil {
			return
		}
		if c.maskKey == nil {
			c.maskKey = make([]byte, 4)
		}
		copy(c.maskKey, maskKey)
	}
	//*读取消息体
	if payloadLen > 0 {
		if payload, err = c.r.Pop(int(payloadLen)); err != nil {
			return
		}
		//*解码消息体 
		if mask {
			maskBytes(c.maskKey, 0, payload)
		}
	}
	return
}

// *关闭连接
func (c *Conn) Close() error {
	return c.rwc.Close()
}

// *用于对 WebSocket 帧的消息体进行掩码处理
func maskBytes(key []byte, pos int, b []byte) int {
	for i := range b {
		b[i] ^= key[pos&3]
		pos++
	}
	return pos & 3
}
