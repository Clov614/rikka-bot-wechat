package adapter

import (
	"bytes"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"math/rand"
	"strings"
	"time"
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
				//rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				//time.Sleep(time.Duration((rnd.Intn(1000) + 1000)) * time.Millisecond) // sui
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
	Self       *openwechat.Self
	RawMsg     *openwechat.Message
	delayToken chan struct{} // 控制消息的接收与发送的随机间隔
}

func NewMetaData(self *openwechat.Self, rawMsg *openwechat.Message) *MetaData {
	return &MetaData{Self: self, RawMsg: rawMsg, delayToken: make(chan struct{})}
}

func (md *MetaData) GetRawMsg() interface{} {
	return md.RawMsg
}

func (md *MetaData) GetISelf() interface{} {
	return md.Self
}

func (md *MetaData) runDelayTimer(delayMin int, delayMax int) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration((rnd.Intn(1000*delayMax-1000*delayMin) + 1000*delayMin)) * time.Millisecond)
	close(md.delayToken)
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
	rikkacfg := a.selfBot.Config
	go metaData.runDelayTimer(rikkacfg.AnswerDelayRandMin, rikkacfg.AnswerDelayRandMax) // 消息随机延迟

	// 获取 ID
	isSendByGroup := msg.IsSendByGroup()
	GroupId := ""
	ReceiveId := ""
	SenderId := ""
	sender, _ := msg.Sender()     // ignore err
	receiver, _ := msg.Receiver() // ignore erryu
	rawContent := msg.RawContent

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
		// 处理群组消息
		rawContent = strings.TrimPrefix(msg.RawContent, senderInGroup.UserName+":<br/>")
	} else {
		SenderId = sender.AvatarID()
		ReceiveId = receiver.AvatarID()
	}

	return &message.Message{
		Msgtype:    rikkaMsgType,
		MetaData:   metaData,
		Raw:        handleSpecialRaw(msg),
		RawContext: rawContent,
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
		msg.SaveFile(&buf) // 图片转为正确 []byte
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
	<-sendMsg.MetaData.(*MetaData).delayToken // 需要延迟随机时间后，才能发送消息
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
