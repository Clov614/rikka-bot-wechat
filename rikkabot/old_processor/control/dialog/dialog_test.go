// @Author Clover
// @Data 2024/7/8 下午11:42:00
// @Desc
package dialog

import (
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"testing"
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
