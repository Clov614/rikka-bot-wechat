// Package dialog
// @Author Clover
// @Data 2024/7/9 上午12:08:00
// @Desc
package dialog

import (
	"github.com/Clov614/rikka-bot-wechat/rikkabot/common"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control"
	"time"
)

// LongFunc 长对话作用方法
type LongFunc func(firstMsg message.Message, recvMsg <-chan message.Message, sendMsg chan<- *message.Message)

type LongDialog struct {
	*Dialog
	filtedRecv <-chan message.Message
	id         string        // 长会话的标识
	Long       LongFunc      // 处理对话实现
	TimeLimit  time.Duration // 对话超时时间
	resetTimer chan struct{}
}

// InitLongDialog 初始化长对话
func InitLongDialog(pluginName string, processRules *control.ProcessRules) *LongDialog {
	return &LongDialog{
		Dialog: initDialog(pluginName, processRules),
	}
}

// SetLongFunc 设置长对话执行时逻辑
func (ld *LongDialog) SetLongFunc(long LongFunc) {
	ld.Long = long
}

func (ld *LongDialog) SendMessage(msg *message.Message) {
	ld.sendMsg <- msg
}

func (ld *LongDialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State) {
	ld.done = done
	ld.Dialog.sendMsg = sendChan
	ld.Dialog.recvMsg = receiveChan
	ld.Cache = cache.GetCache()
	ld.Self = common.GetSelf()
	firstMsg := <-receiveChan
	// 记录 id
	if firstMsg.IsGroup {
		ld.id = firstMsg.RoomId
	} else {
		ld.id = firstMsg.WxId
	}

	// 初始化通道
	if ld.resetTimer == nil {
		ld.resetTimer = make(chan struct{})
	}

	go ld.timeoutDitecter() // 超时监测

	ld.filtedRecv = ld.RecvMsgFilter()

	go func() {
		defer ld.done.SafeClose()
		ld.Long(firstMsg, ld.filtedRecv, sendChan)
	}()
}

func (ld *LongDialog) RecvMessage(checkRules *control.ProcessRules, done chan struct{}) (message.Message, bool, string) {
	if done == nil {
		done = make(chan struct{})
		defer close(done)
	}
	for {
		select {
		case msg := <-ld.filtedRecv:
			msg, isHandle, order := ld.Cache.IsHandle(checkRules, msg)
			if isHandle {
				return msg, true, order
			}
		case <-done:
			return message.Message{}, false, ""
		default:
			time.Sleep(time.Millisecond * 100) // 适当延迟
			continue
		}
	}
}

func (ld *LongDialog) RecvMsgFilter() (filtedRecv chan message.Message) {
	filtedRecv = make(chan message.Message)
	go func() {
		defer close(filtedRecv)
		for {
			select {
			case msg, ok := <-ld.recvMsg:
				if !ok {
					return
				}
				if msg.IsGroup {
					if ld.id == msg.RoomId {
						filtedRecv <- msg
						ld.resetTimer <- struct{}{} // 重置超时
					}
				} else {
					if ld.id == msg.WxId {
						filtedRecv <- msg
						ld.resetTimer <- struct{}{} // 重置超时
					}
				}
			case <-ld.done.Done:
				return
			}
		}
	}()
	return
}

func (ld *LongDialog) timeoutDitecter() {
	if ld.TimeLimit == 0 { // 判断是否未初始化 time.Duration 是 int64
		ld.TimeLimit = 2 * time.Minute // 默认 2分钟超时
	}
	go func() {
		timer := time.NewTimer(ld.TimeLimit)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				// 关闭对话连接
				ld.done.SafeClose()
				return
			case <-ld.resetTimer:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(ld.TimeLimit)
			case <-ld.done.Done:
				return
			}
		}
	}()
}
