package binary


var BigEndian bigEndian

type bigEndian struct{}

//*从字节切片 b 中解码出一个 int8 类型的值
func (bigEndian) Int8(b []byte) int8 { return int8(b[0]) }

//*将 int8 类型的值 v 编码到字节切片 b
func (bigEndian) PutInt8(b []byte, v int8) {
	b[0] = byte(v)
}

//*从字节切片 b 中解码出一个 int16 类型的值
func (bigEndian) Int16(b []byte) int16 { return int16(b[1]) | int16(b[0])<<8 }

//*将 int16 类型的值 v 编码到字节切片 b 中
func (bigEndian) PutInt16(b []byte, v int16) {
	_ = b[1]
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

//*从字节切片 b 中解码出一个 int32 类型的值
func (bigEndian) Int32(b []byte) int32 {
	return int32(b[3]) | int32(b[2])<<8 | int32(b[1])<<16 | int32(b[0])<<24
}

//*将 int32 类型的值 v 编码到字节切片 b 中
func (bigEndian) PutInt32(b []byte, v int32) {
	_ = b[3]
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}
