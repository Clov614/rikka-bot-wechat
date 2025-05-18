// Package message
// @Author Clover
// @Data 2024/7/7 下午1:09:00
// @Desc rikkaMsg
package message

import wcf "github.com/Clov614/wcf-rpc-sdk"

type MsgType int

const (
	MsgTypeText MsgType = 1 << iota
	MsgTypeImage
	MsgTypeVoice
	MsgTypeVideo
	MsgTypeApp
	MsgTypeNewFriendVerify
	//MsgTypeFile todo 待完善消息类型
)

type MsgMetaType int

type Message struct {
	Msgtype    MsgType            `json:"msg_type"`
	MetaData   IMeta              `json:"-"` // `json:"meta_data"` todo 元数据 （封装关于Sender Receiver Self 的 数据/调用）
	RawContent string             `json:"-"`
	ChatImgUrl string             `json:"chat_img_url,omitempty"` // 图片url (只有图片类型消息存在该字段)
	Content    string             `json:"content"`                // 消息内容
	MsgId      uint64             `json:"msg_id"`                 // 唯一标识
	WxId       string             `json:"wx_id"`                  // wxid
	RoomId     string             `json:"room_id,omitempty"`      // roomId
	RoomName   string             `json:"room_name"`              // 群昵称
	RoomAts    []*wcf.ContactInfo `json:"room_ats"`               // 被艾特群成员
	SenderName string             `json:"sender_name"`            // 消息发送者用户昵称
	IsAtMe     bool               `json:"is_at"`                  // 群组中是否艾特本人
	IsGroup    bool               `json:"is_group"`               // 是否为群聊消息
	IsFriend   bool               `json:"is_friend"`              // 是否为好友发送的消息 （注意 不是私聊消息）
	IsGH       bool               `json:"is_gh"`                  // 是否为公众号
	IsMySelf   bool               `json:"is_my_self"`             // 消息是否为自己发送的
	IsSystem   bool               `json:"is_system"`              // 是否为系统消息
	FileInfo   *wcf.FileInfo      `json:"-"`                      // 文件信息
	//Self      ISelf              `json:"raw_msg"` // 原先平台对应对象
	//ReplyFunc func(msg *Message) `json:"-"` // todo 回复消息的方法
	FriendReq *wcf.NewFriendReq `json:"friend_req,omitempty"` // 新好友请求参数
}

type IMeta interface {
	GetRawMsg() interface{}
	GetMsgSenderNickname() string // todo 获取消息发送者昵称
	GetMsgSenderAlias() string
	GetGroupNickname() string                            // todo  获取群组消息的群名
	GetRoomNameByRoomId(nickname string) (string, error) // todo 根据RoomId 获得RoomName
	GetImgData() []byte
}

//type ISelf interface {
//	Self() interface{}
//}

func (m *Message) DeepCopy() *Message {
	newMessage := &Message{
		Msgtype:    m.Msgtype,
		MetaData:   m.MetaData, // 注意：这里是浅拷贝
		RawContent: m.RawContent,
		ChatImgUrl: m.ChatImgUrl,
		Content:    m.Content,
		MsgId:      m.MsgId,
		WxId:       m.WxId,
		RoomId:     m.RoomId,
		RoomName:   m.RoomName,
		SenderName: m.SenderName,
		IsAtMe:     m.IsAtMe,
		IsGroup:    m.IsGroup,
		IsFriend:   m.IsFriend,
		IsGH:       m.IsGH,
		IsMySelf:   m.IsMySelf,
		IsSystem:   m.IsSystem,
		FileInfo:   m.FileInfo, // 注意：这里是浅拷贝，FileInfo 通常包含指针
	}

	if m.RoomAts != nil {
		newMessage.RoomAts = make([]*wcf.ContactInfo, len(m.RoomAts))
		for i, v := range m.RoomAts {
			if v != nil {
				newContactInfo := *v
				newMessage.RoomAts[i] = &newContactInfo
			}
		}
	}

	return newMessage
}
