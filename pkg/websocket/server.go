package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"github.com/gyy0727/mygoim/pkg/bufio"
)

var (
	//*WebSocket 协议中用于计算 Sec-WebSocket-Accept 的固定 GUID
	keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	//*请求方法错误 
	ErrBadRequestMethod = errors.New("bad method")
	//*不符合websocket协议 
	ErrNotWebSocket = errors.New("not websocket protocol")
	//*错误的协议版本 
	ErrBadWebSocketVersion = errors.New("missing or bad WebSocket Version")
	//*不匹配的响应 
	ErrChallengeResponse = errors.New("mismatch challenge/response")
)

//*将 HTTP 连接升级为 WebSocket 连接
func Upgrade(rwc io.ReadWriteCloser, rr *bufio.Reader, wr *bufio.Writer, req *Request) (conn *Conn, err error) {
	//*检查请求方法是否为 GET
	if req.Method != "GET" {
		return nil, ErrBadRequestMethod
	}
	//*检查 WebSocket 版本是否为 13
	if req.Header.Get("Sec-Websocket-Version") != "13" {
		return nil, ErrBadWebSocketVersion
	}
	//*检查 Upgrade 头部是否为 websocket
	if strings.ToLower(req.Header.Get("Upgrade")) != "websocket" {
		return nil, ErrNotWebSocket
	}
	//*检查 Connection 头部是否包含 upgrade
	if !strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade") {
		return nil, ErrNotWebSocket
	}
	//*检查 Sec-WebSocket-Key 是否存在
	challengeKey := req.Header.Get("Sec-Websocket-Key")
	if challengeKey == "" {
		return nil, ErrChallengeResponse
	}
	//*写入 HTTP 101 响应，包括 Sec-WebSocket-Accept
	_, _ = wr.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n")
	_, _ = wr.WriteString("Sec-WebSocket-Accept: " + computeAcceptKey(challengeKey) + "\r\n\r\n")
	//*刷新缓冲区，确保响应发送到客户端
	if err = wr.Flush(); err != nil {
		return
	}
	//*创建并返回 WebSocket 连接
	return newConn(rwc, rr, wr), nil
}

//*计算 Sec-WebSocket-Accept 值
func computeAcceptKey(challengeKey string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(challengeKey))
	_, _ = h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
