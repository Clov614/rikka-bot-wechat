// Package plugins
// @Author Clover
// @Data 2025/3/6 下午9:46:00
// @Desc
package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	wcf "github.com/Clov614/wcf-rpc-sdk"
	"sync"
	"time"

	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
)

var (
	ErrRecvMsgNull = errors.New("receive message is null")
	ErrUnCheck     = errors.New("un check rules")
)

type PluginHandler interface {
	HandleRecv(ctx context.Context)
	Close()
}

type ActionHandler struct {
	Name    string          // Action 的名称，方便识别和管理
	Matcher matcher.Matcher // 核心：用于匹配消息的 Matcher 接口
	Action  ActionFunc      // 核心：实际执行的动作函数
	//priority int        // 可选：优先级，用于控制执行顺序（如果需要）
	Children []*ActionHandler // 子操作，递归执行
	// ... 其他元数据，例如描述、启用状态等
	IsEnable bool // 是否启用
	mu       sync.RWMutex
}

func DefaultActionHandler(name string, isEnable bool) *ActionHandler {
	return &ActionHandler{
		Name:     name,
		IsEnable: isEnable,
	}
}

func (ah *ActionHandler) AsMatcher(matcher matcher.Matcher) *ActionHandler {
	ah.Matcher = matcher
	return ah
}

func (ah *ActionHandler) AsActionFunc(action ActionFunc) *ActionHandler {
	ah.Action = action
	return ah
}

func (ah *ActionHandler) AsChild(child *ActionHandler) *ActionHandler {
	if ah.Children == nil {
		ah.Children = make([]*ActionHandler, 0)
	}
	ah.Children = append(ah.Children, child)
	return ah
}

func (ah *ActionHandler) IsEnabled() bool {
	ah.mu.RLock()
	defer ah.mu.RUnlock()
	return ah.IsEnable
}

func (ah *ActionHandler) Enable() {
	ah.mu.Lock()
	defer ah.mu.Unlock()
	ah.IsEnable = true
}

func (ah *ActionHandler) Disable() {
	ah.mu.Lock()
	defer ah.mu.Unlock()
	ah.IsEnable = false
}

// ActionFunc 动作函数类型
type ActionFunc func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error)

func (ah *ActionHandler) doAction(ctx context.Context, recvMsg *message.Message) (replies []message.Message, ok bool, err error) {
	if recvMsg == nil {
		return nil, false, ErrRecvMsgNull
	}
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	replies = make([]message.Message, 0)
	// 1. 执行当前 Action 的 action
	if ah.Action != nil {
		rMsg, ok, err := ah.Action(ctx, recvMsg)
		if err != nil {
			return nil, false, err // 如果 action 执行出错，立即返回错误
		}
		if ok { // ok判断是否回复
			replies = append(replies, rMsg)
		}
	}

	// 2. 递归处理子 Action
	if ah.Children != nil {
		for _, child := range ah.Children {
			if child.Matcher.Match(ctx, recvMsg) { // **重要: 子 Action 仍然需要 Matcher 匹配**
				childReplies, okChild, childErr := child.doAction(ctx, recvMsg) // 递归调用 doAction
				if childErr != nil {
					return nil, false, childErr // 如果子 Action 出错，立即返回错误
				}
				if okChild {
					replies = append(replies, childReplies...) // 合并子 Action 的 replies
				}

			}
		}
	}

	return replies, true, nil // 成功执行，返回 replies 和 nil error
}

type IPlugin interface {
	HandleRecv(ctx context.Context, recv *message.Message, sendChan chan<- *message.Message) (execute bool)
	GetName() string
	GetLevel() PluginLevel
	GetPluginOpt() PluginOpt
	SetCli(cli *wcf.Client)
	GetCli() *wcf.Client
	EnableP()  // 启用插件
	DisableP() // 禁用插件
	Close()
}

type Plugin struct {
	// 外部控制
	sendChan chan<- *message.Message // 消息发送通道
	// end
	Name              string      // 插件名称
	PluginOpt                     // 插件设置
	Cli               *wcf.Client // wcf客户端
	ActionHandlerList []*ActionHandler
	pluginCancel      context.CancelFunc
	wg                sync.WaitGroup // 用于等待所有 ActionHandler 完成
}

func DefaultPlugin(name string) *Plugin {
	return &Plugin{
		Name: name,
		PluginOpt: PluginOpt{
			IsExclusion: false,           // 默认不排斥
			Level:       MediumLevel,     // 默认中等优先级
			LifeTime:    time.Minute * 2, // 默认存活 2 分钟
		},
		ActionHandlerList: make([]*ActionHandler, 0), // 初始化为空切片
	}
}

func (p *Plugin) AsEnable() *Plugin {
	p.PluginOpt.Enable = true
	return p
}

func (p *Plugin) AsDisable() *Plugin {
	p.PluginOpt.Enable = false
	return p
}

func (p *Plugin) AsAction(ah *ActionHandler) *Plugin {
	if p.ActionHandlerList == nil {
		p.ActionHandlerList = make([]*ActionHandler, 0)
	}
	p.ActionHandlerList = append(p.ActionHandlerList, ah)
	return p
}

func (p *Plugin) AsLevel(l PluginLevel) *Plugin {
	p.PluginOpt.Level = l
	return p
}

func (p *Plugin) AsPluginOpt(opt PluginOpt) *Plugin {
	p.PluginOpt = opt
	return p
}

func (p *Plugin) AsLifeTime(life time.Duration) *Plugin {
	p.LifeTime = life
	return p
}

func (p *Plugin) GetName() string {
	return p.Name
}
func (p *Plugin) GetLevel() PluginLevel {
	return p.PluginOpt.Level
}

func (p *Plugin) GetPluginOpt() PluginOpt {
	return p.PluginOpt
}

func (p *Plugin) SetCli(cli *wcf.Client) {
	p.Cli = cli
}

func (p *Plugin) GetCli() *wcf.Client {
	return p.Cli
}

func (p *Plugin) EnableP() {
	p.PluginOpt.Enable = true
}

func (p *Plugin) DisableP() {
	p.PluginOpt.Enable = false
}

func (p *Plugin) HandleRecv(ctx context.Context, recv *message.Message, sendChan chan<- *message.Message) (execute bool) {
	deadlineCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(p.PluginOpt.LifeTime)) // 使用 Plugin 的上下文作为基础
	p.pluginCancel = cancelFunc
	defer cancelFunc()
	if !p.Enable { // 插件被禁用了
		return false
	}
	p.sendChan = sendChan
	select {
	case <-ctx.Done(): // 外部上下文取消
		cancelFunc() // 通知内层消息处理退出
		return false
	case <-deadlineCtx.Done(): // Plugin 超时
		return false
	default:
		return p.handleMessage(deadlineCtx, recv) // 处理每条接收到的消息
	}
}

func (p *Plugin) Close() {
	if p.pluginCancel != nil {
		p.pluginCancel() // 取消 Plugin 的上下文，停止所有相关的 goroutine (如果 Action 中使用了上下文)
	}
	p.wg.Wait() // 等待所有 ActionHandler 的 goroutine 完成
}

func (p *Plugin) handleMessage(ctx context.Context, recvMsg *message.Message) (isMatch bool) {
	if recvMsg == nil {
		logging.Error("received nil message")
		return
	}

	ahFunc := func(ah *ActionHandler, msg *message.Message) {
		defer p.wg.Done()
		if !ah.Matcher.Match(ctx, msg) { // 不匹配直接退出
			return
		}
		isMatch = true                           // 匹配
		replies, _, err := ah.doAction(ctx, msg) // 获取 reply 和 childActions
		if err != nil {
			logging.ErrorWithErr(err, "doAction err", map[string]interface{}{"actionName": ah.Name})
			logging.Debug("doAction err", map[string]interface{}{"actionName": ah.Name, "msg": msg})
		}
		if replies != nil && len(replies) > 0 {
			for _, reply := range replies {
				select {
				case p.sendChan <- &reply:
				case <-ctx.Done():
					return
				}
			}
		}
	}
	var msg *message.Message
	for _, actionHandler := range p.ActionHandlerList {
		msg = recvMsg.DeepCopy()
		actionHandler := actionHandler
		p.wg.Add(1)
		if p.PluginOpt.ActionAsync {
			go ahFunc(actionHandler, msg)
		} else {
			ahFunc(actionHandler, msg) // 串行
		}
	}
	p.wg.Wait()
	return
}

type PluginLevel uint8

const (
	VeryHighLevel PluginLevel = 0
	HighLevel                 = iota
	UpperLevel
	DownLevel
	MediumLevel
	LowLevel
	VeryLowLevel
)

var Level2Str = map[uint8]string{
	uint8(VeryHighLevel): "VeryHighLevel",
	HighLevel:            "HighLevel",
	UpperLevel:           "UpperLevel",
	DownLevel:            "DownLevel",
	MediumLevel:          "MediumLevel",
	LowLevel:             "LowLevel",
	VeryLowLevel:         "VeryLowLevel",
}

const LevelSize = 7

type PluginOpt struct {
	Enable      bool          // 是否启用
	IsExclusion bool          // todo 是否排斥其他模块
	ActionAsync bool          // 同级action是否异步
	Level       PluginLevel   // 模块等级
	LifeTime    time.Duration // 存活时间
}

type AutoRegister struct {
	pluginLevelList []map[string]IPlugin // 分级模块列表
	mu              sync.Mutex
}

var autoRegister AutoRegister

func (ag *AutoRegister) RegisterPlugin(p IPlugin) {
	c := cache.GetCache()
	pluginInfo, b := c.GetPluginInfo(p.GetName())

	if b { // 检查缓存信息中插件是否开启
		pB, _ := json.Marshal(pluginInfo)
		opt := PluginOpt{}
		err := json.Unmarshal(pB, &opt)
		if err == nil {
			if opt.Enable {
				p.EnableP()
			} else {
				p.DisableP()
			}
		}
	}
	if ag.pluginLevelList == nil {
		ag.pluginLevelList = make([]map[string]IPlugin, LevelSize)
	}
	if ag.pluginLevelList[p.GetLevel()] == nil {
		ag.pluginLevelList[p.GetLevel()] = make(map[string]IPlugin)
	}
	// 打印注册模块信息
	logging.Info(fmt.Sprintf("加载模块: %s 状态: %v 注册等级: %v 存活时间: %v", p.GetName(), p.GetPluginOpt().Enable, p.GetPluginOpt().Level, p.GetPluginOpt().LifeTime))
	ag.mu.Lock()
	defer ag.mu.Unlock()

	ag.pluginLevelList[p.GetLevel()][p.GetName()] = p
}

func (ag *AutoRegister) Plugins() []*IPlugin {
	ag.mu.Lock()
	plugins := make([]*IPlugin, 0)
	for _, m := range ag.pluginLevelList {
		for _, plugin := range m {
			plugins = append(plugins, &plugin)
		}
	}
	ag.mu.Unlock()

	return plugins
}

func (ag *AutoRegister) DisableByName(name string) bool {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	for _, pluginMap := range ag.pluginLevelList {
		p, ok := pluginMap[name]
		if ok {
			p.DisableP()
			return true
		}
	}
	return false
}

func (ag *AutoRegister) EnableByName(name string) bool {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	for _, pluginMap := range ag.pluginLevelList {
		p, ok := pluginMap[name]
		if ok {
			p.EnableP()
			return true
		}
	}
	return false
}

func (ag *AutoRegister) CachePlugins() {
	c := cache.GetCache()
	for _, p := range ag.Plugins() { // fixme: 退出时无法持久化至rikkadb 可能退出顺序相关
		logging.Debug("缓存插件信息", map[string]interface{}{"plugin_name": (*p).GetName()})
		c.CachePluginInfo((*p).GetName(), (*p).GetPluginOpt()) // 缓存插件信息
	}
}

func GetAutoRegister() *AutoRegister {
	return &autoRegister
}
