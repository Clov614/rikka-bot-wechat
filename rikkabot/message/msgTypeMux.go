// Package message
// @Author Clover
// @Data 2024/12/29 下午5:53:00
// @Desc 消息类型选择器
package message

import "sync"

//type IMsgTypeMutex interface {
//	RegistByPluginName(pluginName string, list MsgTypeList)
//}

type MsgTypeList []MsgType

type MsgTypeMux struct {
	mu                        sync.RWMutex
	necessaryMsgTypeByPlugins map[string]MsgTypeList // 插件要求的消息类型 pluginName2MsgType
}

var msgTypeMux *MsgTypeMux

// GetMsgTypeMux 获取消息类型选择器
func GetMsgTypeMux() *MsgTypeMux {
	if msgTypeMux == nil {
		msgTypeMux = &MsgTypeMux{
			necessaryMsgTypeByPlugins: make(map[string]MsgTypeList),
		}
	}
	return msgTypeMux
}

// RegistByPluginName 根据插件名称注册过滤规则
func (m *MsgTypeMux) RegistByPluginName(pluginName string, list MsgTypeList) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.necessaryMsgTypeByPlugins[pluginName] = list
}

// Mux 消息类型选择器，该条消息是否为该模块使用
func (m *MsgTypeMux) Mux(pluginName string, msg Message) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msgType := range m.necessaryMsgTypeByPlugins[pluginName] {
		if msgType == msg.Msgtype {
			return true
		}
	}
	return false
}
