// Package message
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
	MsgTypeApp
	MsgTypeNewFriendVerify
	//MsgTypeFile todo 待完善消息类型
)

type MsgMetaType int

type Message struct {
	Msgtype         MsgType  `json:"msg_type"`
	MetaData        IMeta    `json:"-"` // `json:"meta_data"` todo 元数据 （封装关于Sender Receiver Self 的 数据/调用）
	Raw             []byte   `json:"-"` // 图片数据
	RawContent      string   `json:"-"`
	ChatImgUrl      string   `json:"chat_img_url,omitempty"` // 图片url (只有图片类型消息存在该字段)
	Content         string   `json:"content"`                // 消息内容
	MsgId           uint64   `json:"msg_id"`                 // 唯一标识
	WxId            string   `json:"wx_id"`                  // wxid
	RoomId          string   `json:"room_id,omitempty"`      // roomId
	GroupName       string   `json:"group_name"`             // 群昵称
	SenderName      string   `json:"sender_name"`            // 消息发送者用户昵称
	GroupAtNameList []string `json:"group_at_name_list"`     // todo [改为meta中实现]  群组中艾特的成员昵称（nickname）
	GroupAtWxIdList []string `json:"group_at_wx_id_list"`
	IsAtMe          bool     `json:"is_at"`      // 群组中是否艾特本人
	IsGroup         bool     `json:"is_group"`   // 是否为群聊消息
	IsFriend        bool     `json:"is_friend"`  // 是否为好友私聊消息
	IsMySelf        bool     `json:"is_my_self"` // 消息是否为自己发送的
	IsSystem        bool     `json:"is_system"`  // 是否为系统消息

	//Self      ISelf              `json:"raw_msg"` // 原先平台对应对象
	//ReplyFunc func(msg *Message) `json:"-"` // todo 回复消息的方法
}

type IMeta interface {
	GetRawMsg() interface{}
	GetMsgSenderNickname() string                        // todo 获取消息发送者昵称
	GetGroupNickname() string                            // todo  获取群组消息的群名
	GetRoomNameByRoomId(nickname string) (string, error) // todo 根据RoomId 获得RoomName
}

//type ISelf interface {
//	Self() interface{}
//}
