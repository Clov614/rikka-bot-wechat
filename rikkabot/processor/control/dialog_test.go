// @Author Clover
// @Data 2024/7/8 下午11:42:00
// @Desc
package control

import (
	"testing"
	"wechat-demo/rikkabot/message"
)

func TestDialog_RunPlugin(t *testing.T) {
	sendChan := make(chan *message.Message)
	recvChan := make(chan message.Message)
	dialog := &Dialog{
		PluginName:   "test_plugin",
		ProcessRules: nil,
		sendMsg:      sendChan,
		recvMsg:      recvChan,
		HandleFunc:   nil,
	}
	dialog.HandleFunc = func() {
	}
}
