// Package dialog
// @Author Clover
// @Data 2024/7/8 下午9:41:00
// @Desc 对话控制-普通对话-长对话
package dialog

import (
	"bytes"
	"errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"runtime"
	"sync"
	"time"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/cache"
	"wechat-demo/rikkabot/processor/control"
)

var (
	errPluginRunTime = errors.New("plugin run time default err")
)

type IDialog interface {
	GetPluginName() string
	GetProcessRules() *control.ProcessRules
	RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State)
}

type Dialog struct {
	PluginName   string                  // 插件注册名-对应对话对象
	ProcessRules *control.ProcessRules   // 触发规则
	sendMsg      chan<- *message.Message // 发送消息通道
	recvMsg      chan message.Message    // 接收消息通道
	Cache        *cache.Cache
	Self         *common.Self
	PluginLevel  int // 会话等级
	// HandleFunc is unrecommended: using OnceDialog or LongDialog corresponding method
	HandleFunc func() // 对话逻辑方法

	MsgBuf bytes.Buffer // 消息构建缓冲
	done   *State       // 控制存活
}

func initDialog(pluginName string, processRules *control.ProcessRules) *Dialog {
	return &Dialog{
		PluginName:   pluginName,
		ProcessRules: processRules,
	}
}

func (d *Dialog) GetPluginName() string {
	return d.PluginName
}
func (d *Dialog) GetProcessRules() *control.ProcessRules {
	return d.ProcessRules
}
func (d *Dialog) GetLevel() int {
	return d.PluginLevel
}
func (d *Dialog) SetLevel(level int) {
	d.PluginLevel = level
}

func (d *Dialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State) {
	d.done = done
	defer done.SafeClose()
	d.sendMsg = sendChan
	d.recvMsg = receiveChan
	d.Cache = cache.GetCache()
	d.Self = common.GetSelf()
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

func (d *Dialog) RecvMessage(checkRules *control.ProcessRules, done chan struct{}) (message.Message, bool, string) {
	if done == nil {
		done = make(chan struct{})
		defer close(done)
	}
	for {
		select {
		case msg := <-d.recvMsg:
			msg, isHandle, order := d.Cache.IsHandle(checkRules, msg)
			if isHandle {
				msg.MetaData.AsReadMsg() // 确认处理标为已读消息
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

type OnceFunc func(recvmsg message.Message, sendMsg chan<- *message.Message) // 单次对话插件实现

type OnceDialog struct {
	*Dialog
	Once OnceFunc // 一次对话
}

// InitOnceDialog 初始化单次对话
func InitOnceDialog(pluginName string, processRules *control.ProcessRules) *OnceDialog {
	return &OnceDialog{
		Dialog: initDialog(pluginName, processRules),
	}
}

// SetOnceFunc 设置具体对话逻辑
func (cd *OnceDialog) SetOnceFunc(once OnceFunc) {
	cd.Once = once
}

func (cd *OnceDialog) RunPlugin(sendChan chan<- *message.Message, receiveChan chan message.Message, done *State) {
	defer func() {
		if r := recover(); r != nil {
			log.Err(errPluginRunTime).Msg("panic!!! 插件运行错误！")
			if zerolog.GlobalLevel() == zerolog.DebugLevel { // Debug 模式中
				// 打印详细错误堆栈
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, false)
				logging.Debug("plugin run time default err: "+string(buf), nil)
			}
			// todo 向机器人发送错误提示消息
		}
	}()
	cd.done = done
	defer done.SafeClose()
	cd.sendMsg = sendChan
	cd.recvMsg = receiveChan
	cd.Cache = cache.GetCache()
	cd.Self = common.GetSelf()
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
