// @Author Clover
// @Data 2024/7/7 下午1:09:00
// @Desc rikkaMsg
package message

type MsgType int

const (
	MsgTypeText MsgType = iota
	MsgTypeImage
	MsgTypeVoice
	MsgTypeVideo
	MsgTypeFile
)

type MsgMetaType int

const (
	MsgDefault MsgMetaType = iota
	MsgRequest
	MsgResponse
)

// todo 增加对主动发送消息的支持

type Message struct {
	Msgtype    MsgType     `json:"msg_type"`
	MetaType   MsgMetaType `json:"meta"` // 消息的传递类型: send or receive
	Raw        []byte      `json:"-"`
	RawContext string      `json:"raw_context"`
	IsAt       bool        `json:"is_at"`      // 群组中是否艾特本人
	IsGroup    bool        `json:"is_group"`   // 是否为群聊消息
	IsFriend   bool        `json:"is_friend"`  // 是否为好友私聊消息
	IsMySelf   bool        `json:"is_my_self"` // 是否是自己给自己发送

	RawMsg    IRawMsg            `json:"raw_msg"` // 原先平台对应对象
	ReplyFunc func(msg *Message) `json:"-"`       // 回复消息的方法
}

type IRawMsg interface {
	GetSenderId() string   // 获取发送者唯一用户标识
	GetReceiverId() string // 获取接收者唯一用户标识
	GetRawMsg() any        // 返回本体
}
