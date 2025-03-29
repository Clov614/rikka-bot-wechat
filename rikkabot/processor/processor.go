// Package processor
// @Author Clover
// @Data 2025/3/7 下午8:17:00
// @Desc 模块处理器
package processor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"

	"github.com/Clov614/logging"
	wcf "github.com/Clov614/wcf-rpc-sdk"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"

	/* 下方为插件的导入 */
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/admin"       // 管理员模块
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/biliDecoder" // bilibili链接解析
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/hentai"      // 车来
	/* 从上到下对应优先级由高到低 */
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/ai" // AI对话
	/* 从上到下对应优先级由高到低 */)

type Processor struct {
	ctx        context.Context
	cancel     context.CancelFunc
	LevelLayer []*Layer
	cli        *wcf.Client
	inputChan  chan *message.Message // 接收外部消息的入口 channel
	sendChan   chan *message.Message // 发送消息的 channel
	busyMu     sync.RWMutex
	closeOnce  sync.Once
}

func NewProcessor(ctx context.Context, cli *wcf.Client) *Processor {
	ctx, cancel := context.WithCancel(ctx)
	p := &Processor{
		ctx:        ctx,
		cli:        cli,
		cancel:     cancel,
		LevelLayer: make([]*Layer, plugins.LevelSize), // 初始化 Layer 切片
	}
	return p
}

func (p *Processor) initLayer() {
	// 初始化 Layer
	var nextLayer *Layer = nil                    // 最后一层的 NextLayer 为 nil
	for i := plugins.LevelSize - 1; i >= 0; i-- { // 倒序初始化，方便设置 NextLayer
		layer := &Layer{
			Level:        plugins.PluginLevel(i),
			RecvChan:     make(chan *message.Message), // 每层 Layer 创建自己的 RecvChan
			SendChan:     p.sendChan,
			Plugins:      make([]*plugins.IPlugin, 0), // 初始化插件列表
			NextLayer:    nextLayer,                   // 设置 NextLayer
			processorCtx: p.ctx,                       // 传递 Processor 的 Context
		}
		layer.active.Store(true)
		p.LevelLayer[i] = layer
		nextLayer = layer // 当前层的 RecvChan 成为下一层的 NextLayer
	}
}

// startLayerHandle 处理器启动层处理
func (p *Processor) startLayerHandle() {
	p.busyMu.Lock()
	for _, layer := range p.LevelLayer {
		go layer.StartHandle()
	}
	p.busyMu.Unlock()
}

type Layer struct {
	Level        plugins.PluginLevel
	RecvChan     chan *message.Message
	SendChan     chan *message.Message
	Plugins      []*plugins.IPlugin
	mu           sync.RWMutex
	active       atomic.Bool
	NextLayer    *Layer          // 下一层级
	processorCtx context.Context // Processor 的 Context，用于传递取消信号
}

func (l *Layer) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.active.Store(false)
	// 排空 RecvChan
	for len(l.RecvChan) > 0 {
		<-l.RecvChan
	}
	close(l.RecvChan) // 关闭通道读取
}

func (l *Layer) StartHandle() { // todo 恐慌恢复
	for {
		select {
		case msg, ok := <-l.RecvChan:
			if msg == nil { // 通道关闭
				continue
			}
			if !ok {
				logging.Debug("处理器层接收通道关闭")
				return
			}
			handled := l.handleMessage(msg) // 处理消息，并获取是否被处理的结果
			if !handled && l.NextLayer != nil && l.NextLayer.active.Load() {
				l.NextLayer.mu.RLock()
				l.NextLayer.RecvChan <- msg // 如果当前层级没有处理，且有下一层级，则传递到下一层级
				l.NextLayer.mu.RUnlock()
			}
		case <-l.processorCtx.Done(): // 监听 Processor 的取消信号
			return
		}
	}
}

func (l *Layer) handleMessage(msg *message.Message) bool {
	handled := false // 标记消息是否被处理
	var wg sync.WaitGroup
	for _, plugin := range l.Plugins {
		plugin := *plugin
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !handled { // 只有消息未被处理时才处理
				pluginHandled := plugin.HandleRecv(l.processorCtx, msg, l.SendChan) // 调用 Plugin 的消息处理方法
				if pluginHandled {
					handled = true // 只要有一个插件处理成功，就标记为已处理
				}
			}
		}()
	}
	wg.Wait()
	return handled // 返回消息是否被处理的结果
}

func (l *Layer) GetPlugins() []*plugins.IPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Plugins
}

// RegisterPlugin 注册插件到指定层级
func (p *Processor) RegisterPlugin(level plugins.PluginLevel, plugin *plugins.IPlugin) {
	// 给插件设置cli
	(*plugin).SetCli(p.cli)
	// 注册到每层
	if level >= 0 && level < plugins.LevelSize {
		layer := p.LevelLayer[level]
		layer.mu.Lock()
		layer.Plugins = append(layer.Plugins, plugin)
		layer.mu.Unlock()
	}
}

// Start 启动 Processor 的消息接收和分发
func (p *Processor) Start(recvChan chan *message.Message, sendChan chan *message.Message) {
	// 自动注册插件
	autoRegister := plugins.GetAutoRegister()
	p.sendChan = sendChan
	p.initLayer() // 初始化层
	for _, plugin := range autoRegister.Plugins() {
		p.RegisterPlugin((*plugin).GetLevel(), plugin)
	}
	p.inputChan = recvChan
	p.startLayerHandle() // 启动各层消息处理

	go func() {
		for {
			select {
			case msg, ok := <-p.inputChan: // 接收外部消息
				if msg == nil {
					if !ok {
						return // inputChan 关闭
					}
					continue // 空消息不处理
				}
				logging.Debug("处理器接收到外部消息", map[string]interface{}{"msg": msg})
				if len(p.LevelLayer) > 0 && p.LevelLayer[0].RecvChan != nil {
					p.LevelLayer[0].RecvChan <- msg // 将消息发送到第一层级的 RecvChan
				}
			case <-p.ctx.Done(): // 监听 Processor 的取消信号
				return
			}
		}
	}()
}

// Close 关闭 Processor，停止所有 Layer 和 Plugin 的处理
func (p *Processor) Close() {
	p.busyMu.Lock()
	defer p.busyMu.Unlock()
	p.closeOnce.Do(func() {
		// 先停止接收新的消息
		if p.inputChan != nil {
			close(p.inputChan)
			p.inputChan = nil // 将 p.inputChan 设置为 nil
		}
		// 等待所有消息处理完成, 或者超时
		done := make(chan struct{})
		go func() {
			for _, layer := range p.LevelLayer {
				for _, plugin := range layer.Plugins {
					(*plugin).Close() // 关闭每个 Plugin
				}
			}
			close(done)
			logging.Debug("所有插件关闭完成")
		}()
		// 缓存插件设置信息
		ag := plugins.GetAutoRegister()
		ag.CachePlugins()
		logging.Debug("保存插件信息完毕")

		c := cache.GetCache()
		c.Close() // 保存并关闭缓存
		select {
		case <-done:
		case <-time.After(time.Second * 5): // 设置一个超时时间
			logging.Error("关闭 Processor 超时")
		}
		p.cancel()

		for _, layer := range p.LevelLayer {
			layer.Close() // 关闭对应层
		}
		// 关闭消息发送通道
		if p.sendChan != nil {
			close(p.sendChan)
			p.sendChan = nil // 将p.sendChan设置为nil
		}
	})
}
