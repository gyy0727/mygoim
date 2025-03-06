package errors

import (
	"errors"
)

// .
var (
	//!server
	//*握手失败
	ErrHandshake = errors.New("handshake failed")
	//*请求操作无效 
	ErrOperation = errors.New("request operation not valid")
	//!Ring Buffer 相关错误
	//*环形缓冲区为空
	ErrRingEmpty = errors.New("ring buffer empty")
	//*环形缓冲区已满
	ErrRingFull  = errors.New("ring buffer full")
	//!timer
	//*定时器已满 
	ErrTimerFull   = errors.New("timer full")
	//*定时器为空 
	ErrTimerEmpty  = errors.New("timer empty")
	//*定时器项不存在 
	ErrTimerNoItem = errors.New("timer item not exist")
	//!channel
	//*推送消息参数错误 
	ErrPushMsgArg           = errors.New("rpc pushmsg arg error")
	//*推送多条消息参数错误 
	ErrPushMsgsArg          = errors.New("rpc pushmsgs arg error")
	//*多播推送消息参数错误 
	ErrMPushMsgArg          = errors.New("rpc mpushmsg arg error")
	//*多播推送多条消息参数错误 
	ErrMPushMsgsArg         = errors.New("rpc mpushmsgs arg error")
	//*信号通道已满,丢弃消息 
	ErrSignalFullMsgDropped = errors.New("signal channel full, msg dropped")
	//!bucket
	//*广播参数错误 
	ErrBroadCastArg     = errors.New("rpc broadcast arg error")
	//*广播房间参数错误 
	ErrBroadCastRoomArg = errors.New("rpc broadcast  room arg error")
	//!room
	//*房间已丢弃 
	ErrRoomDroped = errors.New("room droped")
	//!rpc
	//*logic rpc不可用 
	ErrLogic = errors.New("logic rpc is not available")
)
