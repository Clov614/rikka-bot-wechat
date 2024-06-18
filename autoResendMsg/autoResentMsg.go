package autoResendMsg

import (
	"github.com/eatmoreapple/openwechat"
	"time"
)

type UserNameToTime map[string]time.Time

type ResendMsg struct {
	UserNameToTime
	CustomMsg string
}

const DefaultCustomReplyMsg string = "消息被魔法给吞噬了，有什么紧急事件请拨打：13665916698"

func Init() *ResendMsg {
	return &ResendMsg{
		UserNameToTime: make(UserNameToTime, 0),
		CustomMsg:      DefaultCustomReplyMsg,
	}
}

func (rm *ResendMsg) Reply(msg *openwechat.Message) (string, bool) {
	fromUserName := msg.FromUserName
	userTimestamp, ok := rm.UserNameToTime[fromUserName]
	if !ok || isMessageExpired(userTimestamp) {
		rm.UserNameToTime[fromUserName] = time.Now()
		return rm.CustomMsg, true
	}
	return rm.CustomMsg, false
}

func (rm *ResendMsg) IsReply(msg *openwechat.Message) bool {
	if msg.IsSendByGroup() { // 忽略群聊消息
		return false
	}
	fromUserName := msg.FromUserName
	userTimestamp, ok := rm.UserNameToTime[fromUserName]
	if !ok || isMessageExpired(userTimestamp) {
		rm.UserNameToTime[fromUserName] = time.Now()
		return true
	}
	return false
}

// 消息是否过期
func isMessageExpired(timestamp time.Time) bool {
	const ttl = time.Hour
	duration := time.Since(timestamp)
	return duration > ttl
}
