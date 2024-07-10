// @Author Clover
// @Data 2024/7/6 下午8:24:00
// @Desc 全局处理器
package processor

import (
	"sync"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/cache"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/register"
)

type Processor struct {
	*cache.Cache // 处理器缓存
	pluginPool   *register.PluginRegister

	mu           sync.RWMutex
	longConnPool map[chan message.Message]chan struct{} // 长连接池（保存消息接收通道）
	// todo 方法 消息/群号 锁 保证对话发起者只能拥有一个插件的对话存活
}

func NewProcessor() *Processor {
	return &Processor{
		Cache:        cache.GetCache(),
		pluginPool:   register.GetPluginPool(),
		longConnPool: make(map[chan message.Message]chan struct{}),
	}
}

func (p *Processor) DispatchMsg(recvChan chan *message.Message, sendChan chan *message.Message) {
	for msg := range recvChan {
		go p.broadcastRecv(*msg) // 长连接分发消息
		pluginMap := p.pluginPool.GetPluginMap()
		for name, plugin := range pluginMap {
			dialog := plugin.(control.IDialog)
			if p.IsEnable(name) && p.IsHandle(dialog.GetProcessRules(), msg) { // 是否满足规则
				recvConn := make(chan message.Message)
				done := make(chan struct{})
				p.registLongConn(recvConn, done)
				go func() { recvConn <- *msg }()
				go dialog.RunPlugin(sendChan, recvConn, done)
			}
		}
	}
}

func (p *Processor) broadcastRecv(recvMsg message.Message) {
	for c, d := range p.getLongconns() {
		conn := c
		done := d
		go func() {
			select {
			case conn <- recvMsg:
				// do send msg
			case <-done: // 对话已关闭
				p.unregistLongconn(conn)
			default:
				// skip send
			}
		}()
	}
}

func (p *Processor) registLongConn(recvChan chan message.Message, done chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.longConnPool[recvChan] = done
}

func (p *Processor) getLongconns() map[chan message.Message]chan struct{} {
	copyconnPool := make(map[chan message.Message]chan struct{})
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.longConnPool {
		copyconnPool[k] = v
	}
	return copyconnPool
}

func (p *Processor) unregistLongconn(recvChan chan message.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.longConnPool, recvChan)
}
