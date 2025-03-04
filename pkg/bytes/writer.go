package bytes

// *写缓冲区
type Writer struct {
	n   int
	buf []byte
}

// *指定写缓冲区的大小
func NewWriterSize(n int) *Writer {
	return &Writer{buf: make([]byte, n)}
}

// *返回当前缓冲区已写入的数据
func (w *Writer) Len() int {
	return w.n
}

// *返回当前缓冲区的大小
func (w *Writer) Size() int {
	return len(w.buf)
}

// *重置缓冲区大小为0
func (w *Writer) Reset() {
	w.n = 0
}

// *返回当前缓冲区保存的数据
func (w *Writer) Buffer() []byte {
	return w.buf[:w.n]
}

// *从缓冲区中“窥探”并返回一个长度为 n 的字节切片
func (w *Writer) Peek(n int) []byte {
	var buf []byte
	w.grow(n)
	buf = w.buf[w.n : w.n+n]
	w.n += n
	return buf
}

// *写入数据到缓冲区
func (w *Writer) Write(p []byte) {
	w.grow(len(p))
	w.n += copy(w.buf[w.n:], p)
}

// *缓冲区
func (w *Writer) grow(n int) {
	//*缓冲区
	var buf []byte
	//*扩充缓冲区
	if w.n+n < len(w.buf) {
		return
	}
	//*二倍扩容
	buf = make([]byte, 2*len(w.buf)+n)
	//*拷贝原数据到新缓冲区
	copy(buf, w.buf[:w.n])
	w.buf = buf
}
