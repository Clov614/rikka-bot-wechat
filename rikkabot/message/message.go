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

// todo 增加对主动发送消息的支持

type Message struct {
	Msgtype         MsgType  `json:"msg_type"`
	MetaData        IMeta    `json:"meta_data"` // todo 元数据 （封装关于Sender Receiver Self 的 数据/调用）
	Raw             []byte   `json:"-"`
	RawContent      string   `json:"-"`
	Content         string   `json:"content"`            // 消息内容
	GroupId         string   `json:"group_id"`           // 唯一标识
	SenderId        string   `json:"sender_id"`          // 唯一标识
	ReceiverId      string   `json:"receiver_id"`        // 唯一标识
	GroupNameList   []string `json:"group_name_list"`    // 群组成员的昵称（nickname）
	GroupAtNameList []string `json:"group_at_name_list"` // 群组中艾特的成员昵称（nickname）
	IsAtMe          bool     `json:"is_at"`              // 群组中是否艾特本人
	IsGroup         bool     `json:"is_group"`           // 是否为群聊消息
	IsFriend        bool     `json:"is_friend"`          // 是否为好友私聊消息
	IsMySelf        bool     `json:"is_my_self"`         // 消息是否为自己发送的
	IsSystem        bool     `json:"is_system"`          // 是否为系统消息

	//Self      ISelf              `json:"raw_msg"` // 原先平台对应对象
	ReplyFunc func(msg *Message) `json:"-"` // todo 回复消息的方法
}

type IMeta interface {
	GetISelf() interface{}
	GetRawMsg() interface{}
}

//type ISelf interface {
//	Self() interface{}
//}
