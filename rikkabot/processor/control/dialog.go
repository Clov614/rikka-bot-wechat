// Package control
// @Author Clover
// @Data 2024/7/8 下午9:41:00
// @Desc 对话控制-普通对话-长对话
package control

import (
	"bytes"
	"sync"
	"wechat-demo/rikkabot/message"
)

type IDialog interface {
	GetPluginName() string
	GetProcessRules() *ProcessRules
	RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State)
}

type Dialog struct {
	PluginName   string                  // 插件注册名-对应对话对象
	ProcessRules *ProcessRules           // 触发规则
	sendMsg      chan<- *message.Message // 发送消息通道
	recvMsg      chan message.Message    // 接收消息通道
	// HandleFunc is unrecommended: using OnceDialog or LongDialog corresponding method
	HandleFunc func() // 对话逻辑方法

	MsgBuf bytes.Buffer // 消息构建缓冲
	done   *State       // 控制存活
}

func (d *Dialog) GetPluginName() string {
	return d.PluginName
}
func (d *Dialog) GetProcessRules() *ProcessRules {
	return d.ProcessRules
}

func (d *Dialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State) {
	d.done = done
	defer done.SafeClose()
	d.sendMsg = sendChan
	d.recvMsg = receiveChan
	d.HandleFunc()
}

func (d *Dialog) SendText(meta message.IMeta, sendtext string) {
	sendMsg := message.Message{Msgtype: message.MsgTypeText, MetaData: meta, Content: sendtext} // todo test 暂时修改 去掉raw
	d.sendMessage(&sendMsg)
}

func (d *Dialog) SendImage(meta message.IMeta, imgData []byte) {
	sendMsg := message.Message{Msgtype: message.MsgTypeImage, MetaData: meta, Raw: imgData}
	d.sendMessage(&sendMsg)
}

func (d *Dialog) sendMessage(msg *message.Message) {
	d.sendMsg <- msg
}

func (d *Dialog) recvMessage() message.Message {
	return <-d.recvMsg
}

type OnceDialog struct {
	Dialog
	Once func(recvmsg message.Message, sendMsg chan<- *message.Message) // 一次对话
}

func (cd *OnceDialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State) {
	cd.done = done
	defer done.SafeClose()
	cd.sendMsg = sendChan
	cd.recvMsg = receiveChan
	cd.Once(<-cd.recvMsg, cd.sendMsg)
}

type State struct {
	once sync.Once
	Done chan struct{}
}

func NewState() *State {
	return &State{Done: make(chan struct{})}
}

// SafeClose 安全关闭 Done
func (s *State) SafeClose() {
	s.once.Do(func() {
		close(s.Done)
	})
}
