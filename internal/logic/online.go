package logic

import (
	"context"
	"sort"
	"strings"
	"github.com/gyy0727/mygoim/internal/logic/model"
)

var (
	//*_emptyTops 是一个空的 Top 切片，用于在没有数据时返回空结果
	_emptyTops = make([]*model.Top, 0)
)

//*获取在线用户数最多的前 n 个房间
func (l *Logic) OnlineTop(c context.Context, typ string, n int) (tops []*model.Top, err error) {
	for key, cnt := range l.roomCount {
		if strings.HasPrefix(key, typ) {
			_, roomID, err := model.DecodeRoomKey(key)
			if err != nil {
				continue
			}
			top := &model.Top{
				RoomID: roomID,
				Count:  cnt,
			}
			tops = append(tops, top)
		}
	}
	sort.Slice(tops, func(i, j int) bool {
		return tops[i].Count > tops[j].Count
	})
	if len(tops) > n {
		tops = tops[:n]
	}
	if len(tops) == 0 {
		tops = _emptyTops
	}
	return
}

//*获取指定房间的在线用户数
func (l *Logic) OnlineRoom(c context.Context, typ string, rooms []string) (res map[string]int32, err error) {
	res = make(map[string]int32, len(rooms))
	for _, room := range rooms {
		res[room] = l.roomCount[model.EncodeRoomKey(typ, room)]
	}
	return
}

//*获取总的在线 IP 数和连接数
func (l *Logic) OnlineTotal(c context.Context) (int64, int64) {
	return l.totalIPs, l.totalConns
}
