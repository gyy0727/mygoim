package model

//*Online 表示在线状态的数据模型。
//*包含服务器信息、房间数量统计以及最后更新时间。
type Online struct {
	Server    string           `json:"server"`      //*服务器标识，表示当前在线状态所属的服务器
	RoomCount map[string]int32 `json:"room_count"`  //*房间数量统计，键为房间类型，值为对应房间的数量
	Updated   int64            `json:"updated"`    //*最后更新时间，使用 Unix 时间戳表示
}


//*Top 表示排序后的顶部房间数据模型。
//*包含房间 ID 和对应的统计数量。
type Top struct {
	RoomID string `json:"room_id"` //*房间 ID，表示唯一标识一个房间
	Count  int32  `json:"count"`   //*统计数量，表示该房间的某种统计值（如在线人数、消息数量等）
}
