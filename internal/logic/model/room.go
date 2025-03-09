package model

import (
	"fmt"
	"net/url"
)

// *EncodeRoomKey 将房间类型和房间 ID 编码为一个房间键。
// *房间键的格式为 "类型://房间ID"。
func EncodeRoomKey(typ string, room string) string {
	return fmt.Sprintf("%s://%s", typ, room)
}

// *DecodeRoomKey 解码房间键，返回房间类型和房间 ID。
// *如果房间键的格式无效，则返回错误。
func DecodeRoomKey(key string) (string, string, error) {
	u, err := url.Parse(key)
	if err != nil {
		return "", "", err
	}
	return u.Scheme, u.Host, nil
}
