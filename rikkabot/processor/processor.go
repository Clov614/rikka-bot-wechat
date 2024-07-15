// @Author Clover
// @Data 2024/7/6 下午8:24:00
// @Desc 全局处理器
package processor

import (
	"sync"
	"wechat-demo/rikkabot/message"
	_ "wechat-demo/rikkabot/plugins"
	"wechat-demo/rikkabot/processor/cache"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/register"
)

type Processor struct {
	*cache.Cache // 处理器缓存
	pluginPool   *register.PluginRegister

	mu           sync.RWMutex
	longConnPool map[chan message.Message]*control.State // 长连接池（保存消息接收通道）
	// todo 方法 消息/群号 锁 保证对话发起者只能拥有一个插件的对话存活
	// todo 加入done 处理关闭 临时的
	done       chan struct{}
	closeToken chan bool
}

func NewProcessor() *Processor {
	return &Processor{
		Cache:        cache.Init(),
		pluginPool:   register.GetPluginPool(),
		longConnPool: make(map[chan message.Message]*control.State),
		done:         make(chan struct{}),
		closeToken:   make(chan bool, 1),
	}
}

// 阻塞不退出
func (p *Processor) Block() {
	<-p.done
}

// 关闭阻塞
func (p *Processor) Close() {
	p.closeToken <- true
	// 关闭 长连接的所有连接
	for msgChan := range p.getLongconns() {
		select {
		case <-msgChan:
		default:
		}
		p.unregistLongconn(msgChan)
	}
	select {
	case <-p.done:
	default:
		close(p.done)
	}
	<-p.closeToken
	p.Cache.Close() // 关闭缓存
}

// 处理器分发消息，并触发方法，管理长对话
func (p *Processor) DispatchMsg(recvChan chan *message.Message, sendChan chan *message.Message) {
	for {
		select {
		case msg := <-recvChan:
			tempMsg := *msg
			p.broadcastRecv(tempMsg) // 长连接分发消息
			pluginMap := p.pluginPool.GetPluginMap()
			for name, plugin := range pluginMap {
				dialog := plugin.(control.IDialog)
				if p.IsEnable(name) { // 是否启用插件
					if checkedMsg, ok := p.IsHandle(dialog.GetProcessRules(), tempMsg); ok {
						recvConn := make(chan message.Message, 1)
						done := control.NewState()
						p.registLongConn(recvConn, done)
						select { // 发送二手消息
						case recvConn <- checkedMsg:
						default:
							// skip send
						}
						go func() {
							dialog.RunPlugin(sendChan, recvConn, done)
						}()
					}
				}
			}
		default:
			select {
			case <-p.done:
				return
			default:
			}
		}
	}
}

func (p *Processor) broadcastRecv(recvMsg message.Message) {
	p.closeToken <- true
	for c, state := range p.getLongconns() {
		conn := c
		done := state.Done
		select {
		case <-done: // 对话已关闭
			p.unregistLongconn(conn)
		default:
			select {
			case conn <- recvMsg:
			default:
				// 阻塞就跳过发送
			}

		}
	}
	<-p.closeToken
}

func (p *Processor) registLongConn(recvChan chan message.Message, done *control.State) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.longConnPool[recvChan] = done
}

func (p *Processor) getLongconns() map[chan message.Message]*control.State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copyconnPool := make(map[chan message.Message]*control.State)
	for k, v := range p.longConnPool {
		copyconnPool[k] = v
	}
	return copyconnPool
}

func (p *Processor) unregistLongconn(recvChan chan message.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.longConnPool, recvChan)
	close(recvChan)
}
