package logic

import (
	"context"
	"encoding/json"
	"time"
	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gyy0727/mygoim/api/protocol"
	"github.com/gyy0727/mygoim/internal/logic/model"
)


//*主要的逻辑就是将用户和server的映射关系存储在redis中，
func (l *Logic) Connect(c context.Context, server, cookie string, token []byte) (mid int64, key, roomID string, accepts []int32, hb int64, err error) {
	var params struct {
		Mid      int64   `json:"mid"`
		Key      string  `json:"key"`
		RoomID   string  `json:"room_id"`
		Platform string  `json:"platform"`
		Accepts  []int32 `json:"accepts"`
	}
	if err = json.Unmarshal(token, &params); err != nil {
		log.Errorf("json.Unmarshal(%s) error(%v)", token, err)
		return
	}
	mid = params.Mid
	roomID = params.RoomID
	accepts = params.Accepts
	hb = 8
	if key = params.Key; key == "" {
		key = uuid.New().String()
	}
	if err = l.dao.AddMapping(c, mid, key, server); err != nil {
		log.Errorf("l.dao.AddMapping(%d,%s,%s) error(%v)", mid, key, server, err)
	}
	log.Infof("conn connected key:%s server:%s mid:%d token:%s", key, server, mid, token)
	return
}

//*将用户和server的映射关系从redis中删除
func (l *Logic) Disconnect(c context.Context, mid int64, key, server string) (has bool, err error) {
	if has, err = l.dao.DelMapping(c, mid, key, server); err != nil {
		log.Errorf("l.dao.DelMapping(%d,%s) error(%v)", mid, key, server)
		return
	}
	log.Infof("conn disconnected key:%s server:%s mid:%d", key, server, mid)
	return
}

//*更新用户和server的映射关系的过期时间
func (l *Logic) Heartbeat(c context.Context, mid int64, key, server string) (err error) {
	has, err := l.dao.ExpireMapping(c, mid, key)
	if err != nil {
		log.Errorf("l.dao.ExpireMapping(%d,%s,%s) error(%v)", mid, key, server, err)
		return
	}
	if !has {
		if err = l.dao.AddMapping(c, mid, key, server); err != nil {
			log.Errorf("l.dao.AddMapping(%d,%s,%s) error(%v)", mid, key, server, err)
			return
		}
	}
	log.Infof("conn heartbeat key:%s server:%s mid:%d", key, server, mid)
	return
}

//*获取用户和server的映射关系
func (l *Logic) RenewOnline(c context.Context, server string, roomCount map[string]int32) (map[string]int32, error) {
	online := &model.Online{
		Server:    server,
		RoomCount: roomCount,
		Updated:   time.Now().Unix(),
	}
	if err := l.dao.AddServerOnline(context.Background(), server, online); err != nil {
		return nil, err
	}
	return l.roomCount, nil
}

//*接受一个消息 
func (l *Logic) Receive(c context.Context, mid int64, proto *protocol.Proto) (err error) {
	log.Infof("receive mid:%d message:%+v", mid, proto)
	return
}
