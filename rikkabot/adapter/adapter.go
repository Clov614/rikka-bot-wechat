package adapter

import (
	"bytes"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/message"
)

type Adapter struct {
	openwcBot *openwechat.Bot
	selfBot   *rikkabot.RikkaBot
	done      chan struct{}
}

func NewAdapter(openwcBot *openwechat.Bot, selfBot *rikkabot.RikkaBot) *Adapter {
	return &Adapter{openwcBot: openwcBot, selfBot: selfBot, done: make(chan struct{})}
}

func (a *Adapter) HandleCovert() {
	a.openwcBot.MessageHandler = func(msg *openwechat.Message) {
		a.receiveMsg(msg)
	}
	go func() {
		respMsgRecvChan := a.selfBot.GetRespMsgRecvChan()
		for {
			select {
			case <-a.done:
				return
			case respMsg := <-respMsgRecvChan:
				a.sendMsg(respMsg) // todo 错误处理
			}
		}
	}()
}

func (a *Adapter) Close() {
	a.done <- struct{}{}
}

//<editor-fold desc="MetaData">

// MetaData message.IMeta impl
type MetaData struct {
	Self   *openwechat.Self
	RawMsg *openwechat.Message
}

func NewMetaData(self *openwechat.Self, rawMsg *openwechat.Message) *MetaData {
	return &MetaData{Self: self, RawMsg: rawMsg}
}

func (md *MetaData) GetRawMsg() interface{} {
	return md.RawMsg
}

func (md *MetaData) GetISelf() interface{} {
	return md.Self
}

//</editor-fold>

func (a *Adapter) covert(msg *openwechat.Message) *message.Message {
	var rikkaMsgType message.MsgType
	switch msg.MsgType {
	case openwechat.MsgTypeText:
		rikkaMsgType = message.MsgTypeText
	case openwechat.MsgTypeImage:
		rikkaMsgType = message.MsgTypeImage
	case openwechat.MsgTypeVoice:
		rikkaMsgType = message.MsgTypeVoice
	case openwechat.MsgTypeVideo:
		rikkaMsgType = message.MsgTypeVideo
	}

	self, _ := a.openwcBot.GetCurrentUser() // ignore err

	metaData := NewMetaData(self, msg)

	// 获取 ID
	isSendByGroup := msg.IsSendByGroup()
	GroupId := ""
	ReceiveId := ""
	SenderId := ""
	sender, _ := msg.Sender()     // ignore err
	receiver, _ := msg.Receiver() // ignore erryu

	if isSendByGroup {
		senderInGroup, _ := msg.SenderInGroup() // ignore err
		senderInGroup.Detail()                  // 忽略错误
		SenderId = senderInGroup.AvatarID()
		GroupId = sender.AvatarID() // GroupSenderID
		ReceiveId = receiver.AvatarID()
		// 自己发送的ID群号跟接收者号反转
		if msg.IsSendBySelf() {
			GroupId, ReceiveId = ReceiveId, SenderId
		}
	} else {
		SenderId = sender.AvatarID()
		ReceiveId = receiver.AvatarID()
	}

	return &message.Message{
		Msgtype:    rikkaMsgType,
		MetaData:   metaData,
		Raw:        handleSpecialRaw(msg),
		RawContext: msg.RawContent,
		GroupId:    GroupId,
		SenderId:   SenderId,
		ReceiverId: ReceiveId,
		IsAt:       msg.IsAt(), // todo 获取 at的内容 （哪个用户）
		IsGroup:    isSendByGroup,
		IsFriend:   msg.IsSendByFriend(),
		IsMySelf:   msg.IsSendBySelf(), // 是否为自己发送的消息
	}
}

func handleSpecialRaw(msg *openwechat.Message) []byte {
	if msg.MsgType == openwechat.MsgTypeImage {
		var buf bytes.Buffer
		msg.SaveFile(&buf)
		return buf.Bytes()
	}
	return msg.Raw
}

// @Author By Clover 2024/7/5 下午5:28:00
// @Reason 处理外部平台消息，转为自身消息
// @Demand Version
func (a *Adapter) receiveMsg(msg *openwechat.Message) {
	selfMsg := a.covert(msg)
	a.selfBot.GetReqMsgSendChan() <- selfMsg
}

func (a *Adapter) sendMsg(sendMsg *message.Message) error {
	if sendMsg.MetaData == nil {
		panic(fmt.Errorf("sendMsg err: MetaData is nil, can't send msg"))
	}
	rawMsg, ok := sendMsg.MetaData.GetRawMsg().(*openwechat.Message)
	if !ok {
		panic(fmt.Errorf("sendMsg err: MetaData is %#v, can't send msg", sendMsg.MetaData))
	}
	switch sendMsg.Msgtype {
	case message.MsgTypeText:
		rawMsg.ReplyText(sendMsg.RawContext)
	case message.MsgTypeImage:
		rawMsg.ReplyImage(bytes.NewReader(sendMsg.Raw))
	}
	return nil // todo 完善错误处理
}
