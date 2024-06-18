package autoResendMsg

import (
	"github.com/eatmoreapple/openwechat"
	"time"
)

type UserNameToTime map[string]time.Time

type ResendMsg struct {
	UserNameToTime
	CustomMsg string
	TTL       time.Duration
}

const DefaultCustomReplyMsg string = "消息被魔法给吞噬了，有什么紧急事件请拨打：13665916698"

func Init() *ResendMsg {
	return &ResendMsg{
		UserNameToTime: make(UserNameToTime, 0),
		CustomMsg:      DefaultCustomReplyMsg,
		TTL:            time.Hour,
	}
}

func (rm *ResendMsg) Reply(msg *openwechat.Message) (string, bool) {
	fromUserName := msg.FromUserName
	userTimestamp, ok := rm.UserNameToTime[fromUserName]
	if !ok || rm.isMessageExpired(userTimestamp) {
		rm.UserNameToTime[fromUserName] = time.Now()
		return rm.CustomMsg, true
	}
	return rm.CustomMsg, false
}

func (rm *ResendMsg) IsReply(msg *openwechat.Message) bool {
	if msg.IsSendByGroup() { // 忽略群聊消息
		return false
	}
	if !msg.IsSendByFriend() { // 只有私聊自己的消息才会回复
		return false
	}

	if msg.AppMsgType == 5 {
		return false
	}

	// 白名单取反 (取反剩余即为黑名单)
	if !(msg.MsgType == openwechat.MsgTypeText || msg.MsgType == openwechat.MsgTypeVoice ||
		msg.MsgType == openwechat.MsgTypeImage || msg.MsgType == openwechat.MsgTypeVideo ||
		msg.MsgType == openwechat.MsgTypeEmoticon || msg.MsgType == openwechat.MsgTypeLocation ||
		msg.MsgType == openwechat.MsgTypeVoip) {
		return false
	}

	fromUserName := msg.FromUserName
	userTimestamp, ok := rm.UserNameToTime[fromUserName]
	if !ok || rm.isMessageExpired(userTimestamp) {
		rm.UserNameToTime[fromUserName] = time.Now()
		return true
	}
	return false
}

// 消息是否过期
func (rm *ResendMsg) isMessageExpired(timestamp time.Time) bool {
	duration := time.Since(timestamp)
	return duration > rm.TTL
}
