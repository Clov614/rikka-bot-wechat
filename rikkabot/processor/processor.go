// Package processor
// @Author Clover
// @Data 2024/7/6 下午8:24:00
// @Desc 全局处理器
package processor

import (
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	//_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins" // 需要副作用
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	dpkg "github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control/dialog"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/register"
	"sync"
)

type Processor struct {
	*cache.Cache // 处理器缓存
	pluginPool   *register.PluginRegister

	mu           sync.RWMutex
	longConnPool map[chan message.Message]*dpkg.State // 长连接池（保存消息接收通道）
	done         chan struct{}
	closeToken   chan bool // 长连接互斥令牌
}

func NewProcessor() *Processor {
	return &Processor{
		Cache:        cache.Init(),
		pluginPool:   register.GetPluginPool(),
		longConnPool: make(map[chan message.Message]*dpkg.State),
		done:         make(chan struct{}),
		closeToken:   make(chan bool, 1),
	}
}

// Block 阻塞不退出
func (p *Processor) Block() {
	<-p.done
}

// Close 关闭阻塞
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
	logging.Info("all the long conn in pool closed")
	p.Cache.Close() // 关闭缓存
	logging.Info("processor closed")
}

// DispatchMsg 处理器分发消息，并触发方法，管理长对话
func (p *Processor) DispatchMsg(recvChan chan *message.Message, sendChan chan *message.Message) {
	for {
		select {
		case msg := <-recvChan:
			tempMsg := *msg
			p.broadcastRecv(tempMsg)                                   // 长连接分发消息
			pluginMapLevelList := p.pluginPool.GetPluginMapLevelList() // 获取等级划分的插件群
			isTrigger := false                                         // 优先级更高的方法触发标识
			for _, pluginMapLevel := range pluginMapLevelList {
				if isTrigger { // 优先级更高的插件已经触发，本次不执行优先级低的方法
					break
				}
				if pluginMapLevel == nil {
					continue
				}
				for name, plugin := range pluginMapLevel {
					if p.IsEnable(name) { // 是否启用插件
						dialog := plugin.(dpkg.IDialog)
						if checkedMsg, ok, _ := p.IsHandle(dialog.GetProcessRules(), tempMsg); ok {
							isTrigger = true
							recvConn := make(chan message.Message, 1)
							done := dpkg.NewState()
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

func (p *Processor) registLongConn(recvChan chan message.Message, done *dpkg.State) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.longConnPool[recvChan] = done
}

func (p *Processor) getLongconns() map[chan message.Message]*dpkg.State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copyconnPool := make(map[chan message.Message]*dpkg.State)
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
