// @Author Clover
// @Data 2024/7/8 下午9:41:00
// @Desc 对话控制-普通对话-长对话
package control

import "wechat-demo/rikkabot/message"

type IDialog interface {
	GetPluginName()
	GetProcessRules() *ProcessRules
	RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done chan struct{})
}

type Dialog struct {
	PluginName   string                  // 插件注册名-对应对话对象
	ProcessRules *ProcessRules           // 触发规则
	sendMsg      chan<- *message.Message // 发送消息通道
	recvMsg      chan message.Message    // 接收消息通道
	HandleFunc   func()                  // 对话逻辑方法
	done         chan struct{}
}

func (d *Dialog) GetPluginName() string {
	return d.PluginName
}
func (d *Dialog) GetProcessRules() *ProcessRules {
	return d.ProcessRules
}

func (d *Dialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done chan struct{}) {
	d.done = done
	defer close(d.done)
	defer close(d.recvMsg)
	d.sendMsg = sendChan
	d.recvMsg = receiveChan
	d.HandleFunc()
}

func (d *Dialog) sendMessage(msg *message.Message) {
	d.sendMsg <- msg
}

func (d *Dialog) recvMessage() message.Message {
	return <-d.recvMsg
}

type OnceDialog struct {
	Dialog
	Once func(msg message.Message, sendMsg chan<- *message.Message) // 一次对话
}

func (cd *OnceDialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done chan struct{}) {
	cd.done = done
	defer close(cd.done)
	defer close(cd.recvMsg)
	cd.sendMsg = sendChan
	cd.recvMsg = receiveChan
	cd.Once(<-cd.recvMsg, cd.sendMsg)
}
