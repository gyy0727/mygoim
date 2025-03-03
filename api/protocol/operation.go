package protocol

const (
	//*用于表示握手操作
	OpHandshake = int32(0)
	//*用于表示握手操作的回复
	OpHandshakeReply = int32(1)

	//*用于表示心跳操作
	OpHeartbeat = int32(2)
	//*用于表示心跳操作的回复
	OpHeartbeatReply = int32(3)

	//*用于表示发送消息操作
	OpSendMsg = int32(4)
	//*用于表示发送消息操作的回复
	OpSendMsgReply = int32(5)

	//*用于表示断开连接操作的回复
	OpDisconnectReply = int32(6)

	//*用于表示认证操作
	OpAuth = int32(7)
	//*用于表示认证操作的回复
	OpAuthReply = int32(8)

	//*用于表示原始消息操作
	OpRaw = int32(9)

	//*用于表示协议准备操作
	OpProtoReady = int32(10)
	//*用于表示协议完成操作
	OpProtoFinish = int32(11)

	//*用于表示切换房间操作
	OpChangeRoom = int32(12)
	//*用于表示切换房间操作的回复
	OpChangeRoomReply = int32(13)

	//*用于表示订阅操作
	OpSub = int32(14)
	//*用于表示订阅操作的回复
	OpSubReply = int32(15)

	//*用于表示取消订阅操作
	OpUnsub = int32(16)
	//*用于表示取消订阅操作的回复
	OpUnsubReply = int32(17)
)
