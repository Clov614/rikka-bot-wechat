// @Author Clover
// @Data 2024/7/10 下午5:59:00
// @Desc
package processor

import (
	"testing"
	"wechat-demo/rikkabot/message"
)

func TestMsgDispatch(t *testing.T) {
	recvChan := make(chan *message.Message)
	sendChan := make(chan *message.Message)
	processor := NewProcessor()
	processor.DispatchMsg(recvChan, sendChan)
	// todo test 设计模拟消息交互
}
