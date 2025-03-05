package websocket

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"github.com/gyy0727/mygoim/pkg/bufio"
)

type Request struct {
	Method     string        //*请求方法
	RequestURI string        //*请求的url
	Proto      string        //*协议版本
	Host       string        //*请求的主机名
	Header     http.Header   //*请求头
	reader     *bufio.Reader //*用于读取请求数据的 bufio.Reader
}

//*读取解析http请求 
func ReadRequest(r *bufio.Reader) (req *Request, err error) {
	var (
		b  []byte
		ok bool
	)
	req = &Request{reader: r}
	//*读取一行数据 
	if b, err = req.readLine(); err != nil {
		return
	}
	//*解析请求行 
	if req.Method, req.RequestURI, req.Proto, ok = parseRequestLine(string(b)); !ok {
		return nil, fmt.Errorf("malformed HTTP request %s", b)
	}
	//*解析请求头 
	if req.Header, err = req.readMIMEHeader(); err != nil {
		return
	}
	req.Host = req.Header.Get("Host")
	return req, nil
}

//*读取一行数据 
func (r *Request) readLine() ([]byte, error) {
	var line []byte
	for {
		//*读取一整行
		l, more, err := r.reader.ReadLine()
		if err != nil {
			return nil, err
		}
		//*后续没有数据 
		if line == nil && !more {
			return l, nil
		}
		line = append(line, l...)
		if !more {
			break
		}
	}
	return line, nil
}

// *用于从请求中读取并解析 MIME 头部（HTTP 头部）
func (r *Request) readMIMEHeader() (header http.Header, err error) {
	var (
		line []byte //*用于存储读取的每一行数据
		i    int    //*用于存储冒号 : 的位置
		k, v string //*存储键值对
	)
	//*初始化 header，创建一个 http.Header 对象，容量为 16
	header = make(http.Header, 16)
	for {
		if line, err = r.readLine(); err != nil {
			return
		}
		//*去除特殊字符 
		line = trim(line)
		if len(line) == 0 {
			return
		}
		//*查找出字符第一次出现的位置 
		if i = bytes.IndexByte(line, ':'); i <= 0 {
			err = fmt.Errorf("malformed MIME header line: " + string(line))
			return
		}
		k = string(line[:i])
		i++
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		v = string(line[i:])
		//*将提取出的键值对存入http.header结构体 
		header.Add(k, v)
	}
}

// *用于解析 HTTP 请求行
// *line := "GET /index.html HTTP/1.1"
func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

// *用于去除字节切片（[]byte）开头和结尾的空格和制表符
func trim(s []byte) []byte {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	n := len(s)
	for n > i && (s[n-1] == ' ' || s[n-1] == '\t') {
		n--
	}
	return s[i:n]
}
