package bufio

import (
	"bytes"
	"errors"
	"io"
)

const defaultBufSize = 4096

var (
	//*表示在 bufio 包中，UnreadByte 方法被错误地使用
	ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte")
	//*该错误表示在 bufio 包中，UnreadRune 方法被错误地使用
	ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune")
	//*该错误表示缓冲区已满，无法继续写入或读取数据
	ErrBufferFull = errors.New("bufio: buffer full")
	//*该错误表示在 bufio 包中，某些方法接收到了负数的参数（例如读取或写入的字节数）
	ErrNegativeCount = errors.New("bufio: negative count")
)

// *带缓冲区的读结构体
type Reader struct {
	buf  []byte    //*缓冲区
	rd   io.Reader //*读结构体
	r, w int       //*分别表示缓冲区中数据的读取位置和写入位置
	err  error     //*错误
}

// *示缓冲区的最小大小
const minReadBufferSize = 16

// *表示连续空读取的最大次数
const maxConsecutiveEmptyReads = 100

// *新建一个指定大小的reader
func NewReaderSize(rd io.Reader, size int) *Reader {
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size {
		return b
	}
	if size < minReadBufferSize {
		size = minReadBufferSize
	}
	r := new(Reader)
	r.reset(make([]byte, size), rd)
	return r
}

// *新建一个默认大小的reader
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

// *根据传入的io.reader重置
func (b *Reader) Reset(r io.Reader) {
	b.reset(b.buf, r)
}

// *根据传入的buf重置
func (b *Reader) ResetBuffer(r io.Reader, buf []byte) {
	b.reset(buf, r)
}

// *根据传入的两个缓冲区和reader重置
func (b *Reader) reset(buf []byte, r io.Reader) {
	*b = Reader{
		buf: buf,
		rd:  r,
	}
}

// *该错误表示 bufio.Reader 的底层 Read 方法返回了负数的字节数，这违反了 io.Reader 接口的约定
var errNegativeRead = errors.New("bufio: reader returned negative count from Read")

// *如果缓冲区中有未读取的数据（即 b.r > 0），将这些数据移动到缓冲区的开头,如果缓冲区为空,那就填充数据
func (b *Reader) fill() {
	//*检查是否有未读取的数据（b.r 是读指针，如果大于 0，说明有未读取的数据）
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}
	//*检查缓冲区是否已满（b.w 是写指针，如果大于等于缓冲区长度，说明缓冲区已满）
	if b.w >= len(b.buf) {
		panic("bufio: tried to fill full buffer")
	}

	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		//*从io流读出出数据存入到[b.w:缓冲区结尾]
		n, err := b.rd.Read(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
		b.w += n
		if err != nil {
			b.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	b.err = io.ErrNoProgress
}

// *返回当前reader结构体的错误,然后置空reader的错误
func (b *Reader) readErr() error {
	err := b.err
	b.err = nil
	return err
}

// *用于查看缓冲区中的前 n 个字节，但不移动读指针
func (b *Reader) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	if n > len(b.buf) {
		return nil, ErrBufferFull
	}

	for b.w-b.r < n && b.err == nil {
		b.fill()
	}

	var err error
	if avail := b.w - b.r; avail < n {
		n = avail
		err = b.readErr()
		if err == nil {
			err = ErrBufferFull
		}
	}
	return b.buf[b.r : b.r+n], err
}

// *用于读取并返回缓冲区中的前 n 个字节，同时移动读指针
func (b *Reader) Pop(n int) ([]byte, error) {
	d, err := b.Peek(n)
	if err == nil {
		b.r += n
		return d, err
	}
	return nil, err
}

// *用于跳过缓冲区中的前 n 个字节，并返回实际跳过的字节数和可能的错误
func (b *Reader) Discard(n int) (discarded int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return
	}
	remain := n
	for {
		//*获取当前的可读数据的大小
		skip := b.Buffered()
		if skip == 0 {
			//*没有可读的数据
			b.fill()
			skip = b.Buffered()
		}
		//*可读的数据大小大于要跳过的数据
		if skip > remain {
			skip = remain
		}
		//*更新读指针
		b.r += skip
		remain -= skip
		if remain == 0 {
			return n, nil
		}
		//*,没有这么多数据可以 被跳过
		if b.err != nil {
			return n - remain, b.readErr()
		}
	}
}

// *读取数据到 p 中
func (b *Reader) Read(p []byte) (n int, err error) {
	//*如果目标切片为空
	n = len(p)
	if n == 0 {
		return 0, b.readErr()
	}
	//*没有可读数据
	if b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		//*用户提供的切片大于自带的缓冲区
		if len(p) >= len(b.buf) {
			n, b.err = b.rd.Read(p)
			if n < 0 {
				panic(errNegativeRead)
			}
			return n, b.readErr()
		}
		b.fill() //*这是为了填充缓冲区
		if b.r == b.w {
			return 0, b.readErr()
		}
	}

	//*前面填充完缓冲区,现在的目的是将缓冲区的数据拷贝到用户提供的缓冲区
	n = copy(p, b.buf[b.r:b.w])
	//*更新读指针
	b.r += n
	return n, nil
}

// *返回当前的可读数据的长度
func (b *Reader) Buffered() int { return b.w - b.r }

// *读取并返回一个字节
func (b *Reader) ReadByte() (c byte, err error) {
	//*无可读数据
	for b.r == b.w {
		if b.err != nil {
			return 0, b.readErr()
		}
		//*填充缓冲区
		b.fill()
	}
	//*一个字节的数据
	c = b.buf[b.r]
	b.r++
	return c, nil
}

// *读取数据直到遇到分隔符 delim
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	for {
		//*搜索字符delim在b.buf[b.r:b.w]中第一次出现的位置
		if i := bytes.IndexByte(b.buf[b.r:b.w], delim); i >= 0 {
			//*拷贝
			line = b.buf[b.r : b.r+i+1]
			b.r += i + 1
			break
		}

		//*找不到对应的分隔符,直接返回所有数据
		if b.err != nil {
			line = b.buf[b.r:b.w]
			b.r = b.w
			err = b.readErr()
			break
		}

		//*缓冲区已满
		if b.Buffered() >= len(b.buf) {
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}

		b.fill()
	}
	return
}

// *用于读取一行数据
func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
	//*调用readaslice读取直到换行符
	line, err = b.ReadSlice('\n')
	if err == ErrBufferFull {
		//*缓冲区已满但是没有找到换行符
		if len(line) > 0 && line[len(line)-1] == '\r' {
			//*检查行数据是否以 \r 结尾（可能是 \r\n 跨缓冲区的情况）：
			//*如果是，将 \r 放回缓冲区，并从 line 中移除 \r。
			//*返回当前读取的部分行数据，并设置 isPrefix = true，表示后续还有数据。
			if b.r == 0 {

				panic("bufio: tried to rewind past start of buffer")
			}
			b.r--
			line = line[:len(line)-1]
		}
		return line, true, nil
	}

	if len(line) == 0 {
		if err != nil {
			line = nil
		}
		return
	}
	err = nil
	//*确保返回的数据不包含换行符
	if line[len(line)-1] == '\n' {
		drop := 1
		if len(line) > 1 && line[len(line)-2] == '\r' {
			drop = 2
		}
		line = line[:len(line)-drop]
	}
	return
}

// *读结构体
type Writer struct {
	err error
	buf []byte //*缓冲区
	n   int    //*缓冲区已经写入的字节数
	wr  io.Writer
}

func NewWriterSize(w io.Writer, size int) *Writer {

	b, ok := w.(*Writer)
	if ok && len(b.buf) >= size {
		return b
	}
	if size <= 0 {
		size = defaultBufSize
	}
	return &Writer{
		buf: make([]byte, size),
		wr:  w,
	}
}

// *新建默认缓冲区大小的写结构体
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBufSize)
}

func (b *Writer) Reset(w io.Writer) {
	b.err = nil
	b.n = 0
	b.wr = w
}

// *根据传入的buf设置
func (b *Writer) ResetBuffer(w io.Writer, buf []byte) {
	b.buf = buf
	b.err = nil
	b.n = 0
	b.wr = w
}

// *刷新缓冲区到磁盘
func (b *Writer) Flush() error {
	err := b.flush()
	return err
}

func (b *Writer) flush() error {
	//*确保刷新到磁盘之前没有发生过错误
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	//*将缓冲区的数据写入流
	n, err := b.wr.Write(b.buf[0:b.n])
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	//*更新缓冲区的数据位置
	if err != nil {
		if n > 0 && n < b.n {
			copy(b.buf[0:b.n-n], b.buf[n:b.n])
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}

// *返回缓冲区空余的空间大小
func (b *Writer) Available() int { return len(b.buf) - b.n }

// *返回缓冲区已经写入的数据的大小
func (b *Writer) Buffered() int { return b.n }

// (将用户提供的缓冲区)
func (b *Writer) Write(p []byte) (nn int, err error) {
	//*可写的空余空间不足以全部装下
	for len(p) > b.Available() && b.err == nil {
		var n int
		//*已经写入的数据量为 0
		//*如果缓冲区为空（b.Buffered() == 0），直接将数据 p 写入底层 io.Writer，避免额外的拷贝操作。
		//*如果缓冲区不为空，将数据 p 的一部分拷贝到缓冲区中，然后调用 flush 方法将缓冲区数据写入底层 io.Writer。
		//*更新已写入的字节数 nn 和剩余数据 p。
		if b.Buffered() == 0 {
			//*直接写入io流
			n, b.err = b.wr.Write(p)
		} else {
			n = copy(b.buf[b.n:], p)
			b.n += n
			b.flush()
		}
		nn += n
		p = p[n:]
	}
	if b.err != nil {
		return nn, b.err
	}
	n := copy(b.buf[b.n:], p)
	b.n += n
	nn += n
	return nn, nil
}

// *该方法用于将数据 p 直接写入底层 io.Writer，绕过缓冲区，并返回实际写入的字节数和错误信息
func (b *Writer) WriteRaw(p []byte) (nn int, err error) {
	//*之前没有出现过错误
	if b.err != nil {
		return 0, b.err
	}
	//*缓冲区为空
	if b.Buffered() == 0 {
		//*直接写入底层
		nn, err = b.wr.Write(p)
		b.err = err
	} else {
		//*写入缓冲区
		nn, err = b.Write(p)
	}
	return
}

// *该方法用于返回缓冲区中的下 n 个字节，但不会推进写指针。返回的字节在下一次写入操作后失效
func (b *Writer) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	if n > len(b.buf) {
		return nil, ErrBufferFull
	}
	for b.Available() < n && b.err == nil {
		b.flush()
	}
	if b.err != nil {
		return nil, b.err
	}
	d := b.buf[b.n : b.n+n]
	b.n += n
	return d, nil
}

// *写入字符串
func (b *Writer) WriteString(s string) (int, error) {
	nn := 0
	//*字符串的长度大于空余空间
	for len(s) > b.Available() && b.err == nil {
		//*将s拷贝到缓冲区
		n := copy(b.buf[b.n:], s)
		b.n += n
		nn += n
		s = s[n:]
		//*刷入磁盘
		b.flush()
	}
	if b.err != nil {
		return nn, b.err
	}
	//*继续拷贝到缓冲区
	n := copy(b.buf[b.n:], s)
	b.n += n
	nn += n
	return nn, nil
}
