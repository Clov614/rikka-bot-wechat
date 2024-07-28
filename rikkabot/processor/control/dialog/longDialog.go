// Package dialog
// @Author Clover
// @Data 2024/7/9 上午12:08:00
// @Desc
package dialog

import (
	"time"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/cache"
)

type LongDialog struct {
	Dialog
	id         string                                                                                          // 长会话的标识
	Long       func(firstMsg message.Message, recvMsg <-chan message.Message, sendMsg chan<- *message.Message) // 处理对话实现
	TimeLimit  time.Duration                                                                                   // 对话超时时间
	resetTimer chan struct{}
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
		ld.id = firstMsg.GroupId
	} else {
		ld.id = firstMsg.SenderId
	}

	// 初始化通道
	if ld.resetTimer == nil {
		ld.resetTimer = make(chan struct{})
	}

	go ld.timeoutDitecter() // 超时监测

	go func() {
		defer ld.done.SafeClose()
		ld.Long(firstMsg, ld.RecvMsgFilter(), sendChan)
	}()
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
					if ld.id == msg.GroupId {
						filtedRecv <- msg
						ld.resetTimer <- struct{}{} // 重置超时
					}
				} else {
					if ld.id == msg.SenderId {
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
