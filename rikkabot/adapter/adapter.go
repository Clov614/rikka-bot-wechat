package adapter

import (
	"bytes"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"wechat-demo/rikkabot"
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
		a.recevieMsg(msg)
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

//<editor-fold desc="enhance RawMsg">

// EnhanceRawMsg rikka.IRawMsg impl
type EnhanceRawMsg struct {
	rawMsg *openwechat.Message // 原始消息
}

func NewEnhanceRawMsg(msg *openwechat.Message) *EnhanceRawMsg {
	return &EnhanceRawMsg{rawMsg: msg}
}

func (e *EnhanceRawMsg) GetRawMsg() any {
	return e.rawMsg
}

func (e *EnhanceRawMsg) GetSenderId() string {
	sender, err := e.rawMsg.Sender()
	if err != nil {
		panic(err) // 待完善的错误处理 todo
	}
	return sender.AvatarID()
}

func (e *EnhanceRawMsg) GetReceiverId() string {
	receiver, err := e.rawMsg.Receiver()
	if err != nil {
		panic(err) // 待完善的错误处理 todo
	}
	return receiver.AvatarID()
}

//</editor-fold>

func (a *Adapter) covert(msg *openwechat.Message) *rikkabot.Message {
	var rikkaMsgType rikkabot.MsgType
	switch msg.MsgType {
	case openwechat.MsgTypeText:
		rikkaMsgType = rikkabot.MsgTypeText
	case openwechat.MsgTypeImage:
		rikkaMsgType = rikkabot.MsgTypeImage
	case openwechat.MsgTypeVoice:
		rikkaMsgType = rikkabot.MsgTypeVoice
	case openwechat.MsgTypeVideo:
		rikkaMsgType = rikkabot.MsgTypeVideo
	}

	enhanceRawMsg := NewEnhanceRawMsg(msg)

	return &rikkabot.Message{
		Msgtype:    rikkaMsgType,
		MetaType:   rikkabot.MsgRequest,
		Raw:        handleSpecialRaw(msg),
		RawContext: msg.RawContent,
		IsAt:       msg.IsAt(),
		IsGroup:    msg.IsSendByGroup(),
		IsFriend:   msg.IsSendByFriend(),
		IsMySelf:   enhanceRawMsg.GetReceiverId() == enhanceRawMsg.GetSenderId(), // 发送者与接收者为同一个
		RawMsg:     enhanceRawMsg,
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
func (a *Adapter) recevieMsg(msg *openwechat.Message) {
	selfMsg := a.covert(msg)
	a.selfBot.GetReqMsgSendChan() <- selfMsg
}

func (a *Adapter) sendMsg(sendMsg *rikkabot.Message) error {
	if sendMsg.MetaType != rikkabot.MsgResponse {
		panic(fmt.Errorf("sendMsg err: metaType want ”MsgResponse“(2) but got %d", sendMsg.MetaType))
	}
	if sendMsg.RawMsg == nil {
		panic(fmt.Errorf("sendMsg err: rawMsg is nil, can't send msg"))
	}
	rawMsg, ok := sendMsg.RawMsg.GetRawMsg().(*openwechat.Message)
	if !ok {
		panic(fmt.Errorf("sendMsg err: rawMsg is %#v, can't send msg", sendMsg.RawMsg))
	}
	switch sendMsg.Msgtype {
	case rikkabot.MsgTypeText:
		rawMsg.ReplyText(sendMsg.RawContext)
	case rikkabot.MsgTypeImage:
		rawMsg.ReplyImage(bytes.NewReader(sendMsg.Raw))
	}
	return nil // todo 完善错误处理
}
