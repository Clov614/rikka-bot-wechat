package adapter

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"math/rand"
	"regexp"
	"time"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/message"
)

type Adapter struct {
	openwcBot *openwechat.Bot
	selfBot   *rikkabot.RikkaBot
	done      chan struct{}
}

var (
	ErrNotGroupMsg = errors.New("not a group msg")
	ErrMetaDateNil = errors.New("meta date is nil")
	ErrRawMSgNil   = errors.New("raw message is nil")
)

func NewAdapter(openwcBot *openwechat.Bot, selfBot *rikkabot.RikkaBot) *Adapter {
	common.InitSelf(openwcBot) // 初始化 该用户数据（朋友、群组）
	selfBot.SetSelf(common.GetSelf())
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
				err := a.sendMsg(respMsg)
				if err != nil {
					logging.ErrorWithErr(err, "sendMsg fail skip send")
				}
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
	Self        *openwechat.Self
	RawMsg      *openwechat.Message
	GroupMember openwechat.Members // 群组成员（群组消息才会有）
	delayToken  chan struct{}      // 控制消息的接收与发送的随机间隔
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

// GetMsgSenderNickname 获取消息发送者昵称
func (md *MetaData) GetMsgSenderNickname() string {
	sender, _ := md.RawMsg.Sender()
	if sender.IsGroup() { // 获取群组内真实发送者
		senderInGroup, _ := md.RawMsg.SenderInGroup()
		err := senderInGroup.Detail()
		if err != nil {
			logging.ErrorWithErr(err, "GetMsgSenderNickname fail, get sender detail fail")
			return ""
		}
		return senderInGroup.NickName
	}
	if sender == nil {
		return ""
	}
	return sender.NickName
}

// GetGroupNickname 获取群组消息的群名
func (md *MetaData) GetGroupNickname() string {
	if !md.RawMsg.IsSendByGroup() {
		return ""
	}
	sender, _ := md.RawMsg.Sender()
	return sender.NickName
}

// GetGroupMemberIdByNickname 获取群成员的user_id根据nickname
func (md *MetaData) GetGroupMemberIdByNickname(nickname string) (string, error) {
	if !md.RawMsg.IsSendByGroup() { // 不是群组消息，无法获取群成员id
		return "", fmt.Errorf("cannot get member id, %w", ErrNotGroupMsg)
	}
	member, ok := md.GroupMember.GetByNickName(nickname)
	if !ok && member == nil {
		return nickname, nil // todo 自己发送的消息无法获取 member
		//return "", fmt.Errorf("get group member failed by the nickname")
	}
	return member.NickName, nil // todo member只能获取到 username 但是 username 每次登录都会改变 故这临时使用nickname作为标识
}

func (md *MetaData) runDelayTimer(delayMin int, delayMax int) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(rnd.Intn(1000*delayMax-1000*delayMin)+1000*delayMin) * time.Millisecond)
	close(md.delayToken)
}

//</editor-fold>

// covert 消息转换处理
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
	default:
		return nil // 忽略未知的消息种类
	}

	self, _ := a.openwcBot.GetCurrentUser() // ignore err

	metaData := NewMetaData(self, msg)
	rikkacfg := a.selfBot.Config
	go metaData.runDelayTimer(rikkacfg.AnswerDelayRandMin, rikkacfg.AnswerDelayRandMax) // 消息随机延迟

	rself := common.GetSelf() // 获取rikka的self对象
	// 获取 ID
	isSendByGroup := msg.IsSendByGroup()
	uuid := ""
	GroupId := ""
	ReceiveId := ""
	SenderId := ""
	sender, _ := msg.Sender()     // ignore err
	receiver, _ := msg.Receiver() // ignore erryu
	var isAtMe = false
	var groupNameList []string
	var groupAtNameList []string

	// 获取uuid
	uuid = rself.GetUuidById(sender)

	if isSendByGroup {
		senderInGroup, _ := msg.SenderInGroup() // ignore err
		if senderInGroup == nil {
			return nil
		}
		err := senderInGroup.Detail()
		if err != nil {
			logging.ErrorWithErr(err, "GetMsgSenderNickname fail, get sender detail fail")

		} else {
			SenderId = senderInGroup.AvatarID()
		}

		GroupId = sender.AvatarID() // GroupSenderID
		ReceiveId = receiver.AvatarID()

		// 自己发送的ID群号跟接收者号反转
		if msg.IsSendBySelf() {
			GroupId, ReceiveId = ReceiveId, SenderId
			uuid = rself.GetUuidById(receiver) // uuid 改为接收者获取
		}
		// 获取群成员的用户名
		group, ok := sender.AsGroup()
		if ok {
			members, _ := group.Members()  // ignore err
			metaData.GroupMember = members // 加入meta中
			cnt := members.Count()
			groupNameList = make([]string, cnt)
			for i := 0; i < cnt; i++ {
				groupNameList[i] = members[i].NickName
			}
		}
		// 获取消息中艾特成员的成员名
		re := regexp.MustCompile(`@([^\s]+) `)
		matches := re.FindAllStringSubmatch(msg.Content, -1)
		groupAtNameList = make([]string, len(matches))
		for i, match := range matches {
			if len(match) > 1 {
				groupAtNameList[i] = match[1]
				isAtMe = match[1] == self.NickName // 是否艾特自己
			}
		}

	} else {
		SenderId = sender.AvatarID()
		ReceiveId = receiver.AvatarID()
	}

	return &message.Message{
		Msgtype:         rikkaMsgType,
		MetaData:        metaData,
		Raw:             handleSpecialRaw(msg),
		RawContent:      msg.RawContent,
		Content:         msg.Content,
		Uuid:            uuid, // 通过备注名获取的唯一用户标识
		GroupId:         GroupId,
		SenderId:        SenderId,
		ReceiverId:      ReceiveId,
		GroupNameList:   groupNameList,
		GroupAtNameList: groupAtNameList,
		IsAtMe:          isAtMe,
		IsGroup:         isSendByGroup,
		IsFriend:        msg.IsSendByFriend(),
		IsMySelf:        msg.IsSendBySelf(), // 是否为自己发送的消息
	}
}

func handleSpecialRaw(msg *openwechat.Message) []byte {
	if msg.MsgType == openwechat.MsgTypeImage {
		var buf bytes.Buffer
		err := msg.SaveFile(&buf) // 图片转为正确 []byte
		if err != nil {
			logging.ErrorWithErr(err, "SaveFile fail")
		}
		return buf.Bytes()
	} else if msg.MsgType == openwechat.MsgTypeText {
		return nil
	}
	return nil
}

// @Author By Clover 2024/7/5 下午5:28:00
// @Reason 处理外部平台消息，转为自身消息
// @Demand Version
func (a *Adapter) receiveMsg(msg *openwechat.Message) {
	selfMsg := a.covert(msg)
	if selfMsg == nil {
		return
	}
	copyMsg := *selfMsg
	a.selfBot.DispatchMsgEvent(copyMsg) // 存入事件池
	if a.selfBot.EnableProcess {        // 判断是否启动了处理器（防止没有消费者阻塞在此）
		a.selfBot.GetReqMsgSendChan() <- selfMsg
	}
}

func (a *Adapter) sendMsg(sendMsg *message.Message) error {
	if sendMsg.MetaData == nil {
		logging.Debug("MetaData is nil", map[string]interface{}{"sendMsg": sendMsg})
		return fmt.Errorf("can't send msg, sendMsg err: %w", ErrMetaDateNil)
	}
	<-sendMsg.MetaData.(*MetaData).delayToken // 需要延迟随机时间后，才能发送消息
	rawMsg, ok := sendMsg.MetaData.GetRawMsg().(*openwechat.Message)
	if !ok {
		logging.Debug("get metaData.rawMsg failed", map[string]interface{}{"sendMsg": sendMsg})
		return fmt.Errorf("get metaData.rawMsg failed, err: %w", ErrRawMSgNil)
	}
	switch sendMsg.Msgtype {
	case message.MsgTypeText:
		replyText, err := rawMsg.ReplyText(sendMsg.Content)
		if err != nil {
			logging.ErrorWithErr(err, "SendMsg fail", map[string]interface{}{"replyText": replyText})
		}
	case message.MsgTypeImage:
		replyImage, err := rawMsg.ReplyImage(bytes.NewReader(sendMsg.Raw))
		if err != nil {
			logging.ErrorWithErr(err, "SendMsg fail", map[string]interface{}{"replyImage": replyImage})
		}
	default:
		logging.Warn("unknown msgType do not handle send")
	}
	return nil
}
