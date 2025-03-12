package logic

import (
	"context"
	"github.com/gyy0727/mygoim/internal/logic/model"
	log "github.com/golang/glog"
)

//*推送消息给批量用户
func (l *Logic) PushKeys(c context.Context, op int32, keys []string, msg []byte) (err error) {
	servers, err := l.dao.ServersByKeys(c, keys)
	if err != nil {
		return
	}
	pushKeys := make(map[string][]string)
	for i, key := range keys {
		server := servers[i]
		if server != "" && key != "" {
			pushKeys[server] = append(pushKeys[server], key)
		}
	}
	for server := range pushKeys {
		if err = l.dao.PushMsg(c, op, server, pushKeys[server], msg); err != nil {
			return
		}
	}
	return
}

//*根据用户 ID（mids）推送消息到对应的服务器
func (l *Logic) PushMids(c context.Context, op int32, mids []int64, msg []byte) (err error) {
	keyServers, _, err := l.dao.KeysByMids(c, mids)
	if err != nil {
		return
	}
	keys := make(map[string][]string)
	for key, server := range keyServers {
		if key == "" || server == "" {
			log.Warningf("push key:%s server:%s is empty", key, server)
			continue
		}
		keys[server] = append(keys[server], key)
	}
	for server, keys := range keys {
		if err = l.dao.PushMsg(c, op, server, keys, msg); err != nil {
			return
		}
	}
	return
}

//*向指定房间广播消息
func (l *Logic) PushRoom(c context.Context, op int32, typ, room string, msg []byte) (err error) {
	return l.dao.BroadcastRoomMsg(c, op, model.EncodeRoomKey(typ, room), msg)
}

//*向所有用户广播消息
func (l *Logic) PushAll(c context.Context, op, speed int32, msg []byte) (err error) {
	return l.dao.BroadcastMsg(c, op, speed, msg)
}
