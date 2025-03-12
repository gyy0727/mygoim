package model

import (
	"testing"
)

// 测试 EncodeRoomKey 函数
func TestEncodeRoomKey(t *testing.T) {
	tests := []struct {
		typ  string
		room string
		want string
	}{
		{"chat", "123", "chat://123"},
		{"video", "456", "video://456"},
		{"audio", "789", "audio://789"},
		{"", "empty", "://empty"}, // 空类型
		{"empty", "", "empty://"}, // 空房间 ID
	}

	for _, tt := range tests {
		t.Run(tt.typ+"_"+tt.room, func(t *testing.T) {
			got := EncodeRoomKey(tt.typ, tt.room)
			t.Logf("Input: typ=%q, room=%q", tt.typ, tt.room)
			t.Logf("Output: %q", got)
			if got != tt.want {
				t.Errorf("EncodeRoomKey(%q, %q) = %q; want %q", tt.typ, tt.room, got, tt.want)
			}
		})
	}
}

// 测试 DecodeRoomKey 函数
func TestDecodeRoomKey(t *testing.T) {
	tests := []struct {
		key       string
		wantTyp   string
		wantRoom  string
		wantError bool
	}{
		{"chat://123", "chat", "123", false},
		{"video://456", "video", "456", false},
		{"audio://789", "audio", "789", false},
		{"://empty", "", "empty", false},     // 空类型
		{"empty://", "empty", "", false},     // 空房间 ID
		{"invalid", "", "", true},            // 无效格式
		{"invalid://", "invalid", "", false}, // 无效格式但可解析
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotTyp, gotRoom, err := DecodeRoomKey(tt.key)
			t.Logf("Input: key=%q", tt.key)
			t.Logf("Output: typ=%q, room=%q, error=%v", gotTyp, gotRoom, err)
			if (err != nil) != tt.wantError {
				t.Errorf("DecodeRoomKey(%q) error = %v; wantError %v", tt.key, err, tt.wantError)
				return
			}
			if gotTyp != tt.wantTyp || gotRoom != tt.wantRoom {
				t.Errorf("DecodeRoomKey(%q) = (%q, %q); want (%q, %q)", tt.key, gotTyp, gotRoom, tt.wantTyp, tt.wantRoom)
			}
		})
	}
}
