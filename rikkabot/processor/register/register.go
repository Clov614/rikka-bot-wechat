// @Author Clover
// @Data 2024/7/6 下午8:28:00
// @Desc 注册 插件、模块，供处理器使用
package register

import (
	"sync"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/control"
)

// todo 通用 状态缓存，信息缓存结构的定义（定义标准供插件使用） （总结出功能的共性）（通用的开发功能方法的提取）

type IPlugin interface {
	HandleMsgFunc(msg *message.Message)
	GetHandleRules() *control.ProcessRules
}

type PluginRegister struct {
	mu      sync.RWMutex       // guard
	Plugins map[string]IPlugin // 插件 名称：插件主体

}

func (p *PluginRegister) Regist(name string, plugin IPlugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Plugins[name] = plugin
}

func (p *PluginRegister) Unregist(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Plugins, name)
}

func (p *PluginRegister) GetPlugin(name string) IPlugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Plugins[name]
}

func (p *PluginRegister) GetPluginMap() map[string]IPlugin {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// create a copy of the plugins map
	pluginsCopy := make(map[string]IPlugin, len(p.Plugins))
	for name, plugin := range p.Plugins {
		pluginsCopy[name] = plugin
	}
	return pluginsCopy
}

var pluginPool *PluginRegister

func GetPluginPool() *PluginRegister {
	return pluginPool
}

func init() {
	pluginPool = &PluginRegister{
		Plugins: make(map[string]IPlugin),
	}
}
