// Package register @Author Clover
// @Data 2024/7/6 下午8:28:00
// @Desc 注册 插件、模块，供处理器使用
package register

import (
	"sync"
	"wechat-demo/rikkabot/processor/control"
)

// todo 通用 状态缓存，信息缓存结构的定义（定义标准供插件使用） （总结出功能的共性）（通用的开发功能方法的提取）

type IPlugin interface {
	GetPluginName() string
	GetProcessRules() *control.ProcessRules
	GetLevel() int
	SetLevel(int)
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

var cachePluginMapLevelList []map[string]IPlugin // 等级划分模块列表缓存

// GetPluginMapLevelList 根据等级划分模块（等级==优先级  等级为切片下标）
func (p *PluginRegister) GetPluginMapLevelList() []map[string]IPlugin {
	if cachePluginMapLevelList != nil {
		return cachePluginMapLevelList
	}
	pluginMap := p.GetPluginMap()
	// 初始5个等级
	pluginLevelList := make([]map[string]IPlugin, 5)
	for name, plugin := range pluginMap {
		level := plugin.GetLevel()
		if level >= len(pluginLevelList) {
			// 等级超过大小扩容
			copyList := make([]map[string]IPlugin, level+1)
			copy(copyList, pluginLevelList)
			pluginLevelList = copyList
		}
		if pluginLevelList[level] == nil {
			pluginLevelList[level] = map[string]IPlugin{name: plugin}
		} else {
			pluginLevelList[level][name] = plugin
		}
	}
	cachePluginMapLevelList = pluginLevelList // cache
	return pluginLevelList
}

var pluginPool *PluginRegister

// RegistPlugin 注册对话插件
func RegistPlugin(name string, plugin IPlugin, pluginLevel int) {
	plugin.SetLevel(pluginLevel)
	pluginPool.Regist(name, plugin)
}

func GetPluginPool() *PluginRegister {
	return pluginPool
}

func init() {
	pluginPool = &PluginRegister{
		Plugins: make(map[string]IPlugin),
	}
}
