package rikkabot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/onebot/dto/event"
	"wechat-demo/rikkabot/processor"
)

type RikkaBot struct {
	ctx     context.Context
	cancel  func()
	self    *common.Self
	sendMsg chan *message.Message
	recvMsg chan *message.Message
	Config  *config.CommonConfig

	EnableProcess bool // 是否处理消息
	Processor     *processor.Processor

	enableEventHandle bool // 是否开启事件处理
	EventPool         *event.EventPool
	EventFuncs        []func(event event.IEvent)
	mu                sync.Mutex

	err error
}

var (
	ErrInvalidCall = errors.New("invalid bot call")
)

var DefaultBot *RikkaBot

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.GetConfig()
	DefaultBot = &RikkaBot{
		ctx:        ctx,
		cancel:     cancel,
		sendMsg:    make(chan *message.Message),
		recvMsg:    make(chan *message.Message),
		Processor:  processor.NewProcessor(),
		Config:     cfg,
		EventPool:  event.NewEventPool(cfg.HttpServer.EventBufferSize),
		EventFuncs: make([]func(event event.IEvent), 0),
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
		return nil, fmt.Errorf("error update bot config: %w", err)
	}
	return DefaultBot, nil
}

func (r *RikkaBot) OnEventPush(f func(event event.IEvent)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.EventFuncs = append(r.EventFuncs, f)
}

func (r *RikkaBot) GetEventFuncs() []func(event event.IEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.EventFuncs
}

func (r *RikkaBot) StartHandleEvent() {
	r.enableEventHandle = true
	logging.Info("开始处理事件")
	r.EventPool.StartProcessing(r.GetEventFuncs()...)
}

func (r *RikkaBot) DispatchMsgEvent(rikkaMsg message.Message) {
	if !r.enableEventHandle {
		return // 不处理分发消息
	}
	var detailType string
	if rikkaMsg.IsGroup {
		detailType = "group"
	} else if rikkaMsg.IsFriend {
		detailType = "private"
	}
	msgEvent := event.MsgEvent{}
	msgEvent.InitEvent("message", detailType, "")
	msgEvent.InitMsgEvent(rikkaMsg)
	err := r.EventPool.AddEvent(msgEvent) // ignore err
	if err != nil {
		logging.Warn(fmt.Sprintf("事件池警告 %w", err))
	}
}

func (r *RikkaBot) SetSelf(self *common.Self) {
	r.self = self
}

// Start 启动 rikkabot 进行消息处理
func (r *RikkaBot) Start() {
	logging.Info("rikka bot start")
	go r.Processor.DispatchMsg(r.recvMsg, r.sendMsg)
	r.EnableProcess = true // 防止生产者阻塞
}

// Exit 主动退出 rikkabot
func (r *RikkaBot) Exit() {
	r.Processor.Close()
	r.cancel()
	logging.Info("rikka bot exited")
}

// Block 当发生错误，该方法会立即返回，否则会一直阻塞
func (r *RikkaBot) Block() error {
	if r.self == nil {
		return fmt.Errorf("`Block` must be called after adapter.HandleCovert(): %w", ErrInvalidCall)
	}
	<-r.ctx.Done()
	logging.Close() // 关闭日志文件
	return r.err
}

// region Msg Chan Operation

// GetReqMsgSendChan 获取消息接收通道 写通道
func (r *RikkaBot) GetReqMsgSendChan() chan<- *message.Message {
	return r.recvMsg
}

// GetReqMsgRecvChan 获取消息接收通道 读通道
func (r *RikkaBot) GetReqMsgRecvChan() <-chan *message.Message {
	return r.recvMsg
}

// GetRespMsgSendChan 获取消息发送通道 写通道
func (r *RikkaBot) GetRespMsgSendChan() chan<- *message.Message {
	return r.sendMsg
}

// GetRespMsgRecvChan 获取消息发送通道 读通道
func (r *RikkaBot) GetRespMsgRecvChan() <-chan *message.Message {
	return r.sendMsg
}

//endregion
