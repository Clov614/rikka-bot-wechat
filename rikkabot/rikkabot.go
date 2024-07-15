package rikkabot

import (
	"context"
	"errors"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor"
)

type RikkaBot struct {
	ctx       context.Context
	cancel    func()
	self      *common.Self
	sendMsg   chan *message.Message
	recvMsg   chan *message.Message
	Config    *config.CommonConfig
	Processor *processor.Processor // todo 待完善 待确认

	err error
}

var DefaultBot *RikkaBot

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	DefaultBot = &RikkaBot{
		ctx:       ctx,
		cancel:    cancel,
		sendMsg:   make(chan *message.Message),
		recvMsg:   make(chan *message.Message),
		Processor: processor.NewProcessor(),
		Config:    config.GetConfig(),
	}
}

func GetDefaultBot() *RikkaBot {
	return DefaultBot
}

func Bot() *RikkaBot {
	return DefaultBot
}

func GetBot(botname string) (*RikkaBot, error) {
	DefaultBot.Config.Botname = botname
	err := DefaultBot.Config.Update()
	if err != nil {
		return nil, err
	}
	return DefaultBot, nil
}

// 启动 rikkabot 进行消息处理
func (r *RikkaBot) Start() {
	println("rikkabot start")
	go r.Processor.DispatchMsg(r.recvMsg, r.sendMsg)
}

// 主动退出 rikkabot
func (r *RikkaBot) Exit() {
	r.Processor.Close()
	r.cancel()
}

// Block 当发生错误，该方法会立即返回，否则会一直阻塞
func (r *RikkaBot) Block() error {
	if r.self == nil {
		return errors.New("`Block` must be called after adapter.HandleCovert()")
	}
	<-r.ctx.Done()
	return r.err
}

// region Msg Chan Operation

// 获取消息接收通道 写通道
func (rb *RikkaBot) GetReqMsgSendChan() chan<- *message.Message {
	return rb.recvMsg
}

// 获取消息接收通道 读通道
func (rb *RikkaBot) GetReqMsgRecvChan() <-chan *message.Message {
	return rb.recvMsg
}

// 获取消息发送通道 写通道
func (rb *RikkaBot) GetRespMsgSendChan() chan<- *message.Message {
	return rb.sendMsg
}

// 获取消息发送通道 读通道
func (rb *RikkaBot) GetRespMsgRecvChan() <-chan *message.Message {
	return rb.sendMsg
}

//endregion
