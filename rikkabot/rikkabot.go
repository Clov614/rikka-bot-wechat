package rikkabot

import "context"

type RikkaBot struct {
	Botname string // 机器人昵称
	ctx     context.Context
	sendMsg chan *Message
	recvMsg chan *Message
	err     error
}

var DefaultBot *RikkaBot

func init() {
	DefaultBot = &RikkaBot{
		Botname: "rikka",
		ctx:     context.Background(),
		sendMsg: make(chan *Message),
		recvMsg: make(chan *Message),
	}
}

func GetDefaultBot() *RikkaBot {
	return DefaultBot
}

func GetBot(botname string) *RikkaBot {
	DefaultBot.Botname = botname
	return DefaultBot
}

// region Msg Chan Operation

// 获取消息接收通道 写通道
func (rb *RikkaBot) GetReqMsgSendChan() chan<- *Message {
	return rb.recvMsg
}

// 获取消息接收通道 读通道
func (rb *RikkaBot) GetReqMsgRecvChan() <-chan *Message {
	return rb.recvMsg
}

// 获取消息发送通道 写通道
func (rb *RikkaBot) GetRespMsgSendChan() chan<- *Message {
	return rb.sendMsg
}

// 获取消息发送通道 读通道
func (rb *RikkaBot) GetRespMsgRecvChan() <-chan *Message {
	return rb.sendMsg
}

//endregion

type MsgType int

const (
	MsgTypeText MsgType = iota
	MsgTypeImage
	MsgTypeVoice
	MsgTypeVideo
	MsgTypeFile
	MsgTypeUrl
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
	Raw        []byte      `json:"raw"`
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
