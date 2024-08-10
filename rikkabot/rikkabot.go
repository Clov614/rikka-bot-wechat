package rikkabot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/onebot/dto/event"
	"wechat-demo/rikkabot/processor"
	"wechat-demo/rikkabot/utils/imgutil"
	"wechat-demo/rikkabot/utils/timeutil"
)

type RikkaBot struct {
	ctx      context.Context
	cancel   func()
	self     *common.Self
	sendMsg  chan *message.Message
	recvMsg  chan *message.Message
	Config   *config.CommonConfig
	loginUrl string // 登录链接

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
	ErrSendMsg     = errors.New("send message error")
	ErrFetchImg    = errors.New("fetch image error")
)

var DefaultBot *RikkaBot

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.GetConfig()
	// 启动日志检测模块
	go logging.MonitorLogSize(int64(cfg.LogMaxSize) * 1024 * 1024)
	// 初始化
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

func (r *RikkaBot) GetloginUrl() string {
	return r.loginUrl
}

func (r *RikkaBot) SetloginUrl(url string) {
	r.loginUrl = url
}

// PushLoginNoticeEvent 推送登录事件
func (r *RikkaBot) PushLoginNoticeEvent() {
	loginUrl := r.GetloginUrl()
	if loginUrl == "" { // 没有回调不需要处理
		return
	}
	// 构造 notice—event
	type LoginType struct {
		LoginUrl string `json:"login_url"`
	}
	var loginData LoginType
	loginData.LoginUrl = loginUrl
	e := event.Event{
		Id:         uuid.New().String(),
		Time:       timeutil.GetTimeUnix(),
		Type:       "notice",
		DetailType: "login_callback",
		SubType:    "",
	}
	noticeEvent := event.NoticeEvent[LoginType]{}
	initNoticeEvent := noticeEvent.InitNoticeEvent(e, loginData)
	err := r.EventPool.AddEvent(*initNoticeEvent)
	if err != nil {
		logging.WarnWithErr(err, "推送登录回调事件至事件池错误")
	}
}

// PushLogOutNoticeEvent 推送机器人掉线事件
func (r *RikkaBot) PushLogOutNoticeEvent(code int, msg string) {
	// 构造 notice—event
	type LogOutType struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	var logoutData LogOutType
	logoutData.Code = code
	logoutData.Msg = msg
	e := event.Event{
		Id:         uuid.New().String(),
		Time:       timeutil.GetTimeUnix(),
		Type:       "notice",
		DetailType: "logout",
		SubType:    "",
	}
	noticeEvent := event.NoticeEvent[LogOutType]{}
	initNoticeEvent := noticeEvent.InitNoticeEvent(e, logoutData)
	err := r.EventPool.AddEvent(*initNoticeEvent)
	if err != nil {
		logging.WarnWithErr(err, "推送登出事件至事件池错误")
	}
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
		logging.Warn(fmt.Sprintf("事件池警告 %s", err))
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
	logging.Info("rikka bot exited")
	r.Processor.Close()
	r.cancel()
}

// ExitWithErr 异常退出 rikkabot
func (r *RikkaBot) ExitWithErr(code int, msg string) {
	logging.Info("rikka bot exited")
	logging.Error("异常退出")
	logging.Error(msg, map[string]interface{}{"exit code": code})
	r.Processor.Close()
	r.cancel()
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

// SendMsg 统一发送消息接口 消息类型 是否群组 发送数据 群/好友 id
// nolint
func (r *RikkaBot) SendMsg(msgType message.MsgType, isGroup bool, data any, sendId string) error {
	// todo 发送消息回调消息id 并保存sendmsg，提供过期控制、根据id查询发送的消息
	var err error
	switch msgType {
	case message.MsgTypeText:
		s, ok := data.(string)
		if !ok {
			return fmt.Errorf("`SendMsg of text` must be a string: %w", ErrSendMsg)
		}
		if common.IsUuidValid(sendId) { // 支持 uuid为sendId 发送消息
			err = r.self.SendTextByUuid(sendId, s, isGroup)
			return err
		}
		err = r.self.SendTextById(sendId, s, isGroup)
	case message.MsgTypeImage:
		d, ok := data.([]byte)
		if !ok { // 断言失败，传入的是路径，请求即可
			path := data.(string)
			d, err = imgutil.ImgFetch(path)
			if err != nil {
				err = fmt.Errorf("%w 从路径 %s 中获取路径发送错误 %w", ErrFetchImg, path, err)
				return err
			}
		}
		var buf bytes.Buffer
		buf.Write(d)
		if common.IsUuidValid(sendId) { // 支持 uuid为sendId 发送消息
			err = r.self.SendImgByUuid(sendId, &buf, isGroup)
			return err
		}
		err = r.self.SendImgById(sendId, &buf, isGroup)
	default:
		err = fmt.Errorf("`SendMsg of type` must be either text or image: %w", ErrSendMsg)
	}
	return err
}
