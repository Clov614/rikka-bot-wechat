package rikkabot

import (
	"context"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/message"
)

type RikkaBot struct {
	ctx     context.Context
	sendMsg chan *message.Message
	recvMsg chan *message.Message
	Config  *config.CommonConfig
	//Processor *processor.Processor // todo 待完善 待确认

	err error
}

var DefaultBot *RikkaBot

func init() {
	DefaultBot = &RikkaBot{
		ctx:     context.Background(),
		sendMsg: make(chan *message.Message),
		recvMsg: make(chan *message.Message),
		Config:  config.GetConfig(),
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
