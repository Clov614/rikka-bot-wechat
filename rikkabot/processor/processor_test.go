// @Author Clover
// @Data 2024/7/10 下午5:59:00
// @Desc
package processor

import (
	"fmt"
	"testing"
	"time"
	"wechat-demo/rikkabot/message"
	_ "wechat-demo/rikkabot/plugins"
	"wechat-demo/rikkabot/processor/cache"
)

func TestMsgDispatch(t *testing.T) {
	recvChan := make(chan *message.Message)
	sendChan := make(chan *message.Message)
	processor := NewProcessor()
	go processor.DispatchMsg(recvChan, sendChan)

	// todo test 设计模拟消息交互
	// handle send recvmsg
	testmessages := []message.Message{
		{IsMySelf: true, IsGroup: true,
			GroupId: "813467281 ", SenderId: "2788092443", ReceiverId: "2788092443",
			Content: "/rikka add whitelist 1"},
		{IsMySelf: true, IsGroup: true,
			GroupId: "813467281 ", SenderId: "2788092443", ReceiverId: "2788092443",
			Content: "/123123125432"},
		{IsMySelf: true, IsGroup: false,
			GroupId: "", SenderId: "2788092443", ReceiverId: "2788092443",
			Content: "/rikka add whitelist 2"},
		{IsMySelf: false, IsGroup: false,
			GroupId: "", SenderId: "813466966", ReceiverId: "2788092443",
			Content: "/rikka add whitelist"},
		// 长对话测试
		{IsMySelf: false, IsGroup: true,
			GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443",
			Content: "/rikka 长对话测试"},
		{IsMySelf: true, IsGroup: true,
			GroupId: "813467281", SenderId: "2788092443", ReceiverId: "2788092443",
			Content: "44"},
		{IsMySelf: false, IsGroup: true,
			GroupId: "777777777777", SenderId: "813466966", ReceiverId: "2788092443",
			Content: "46"},
		{IsMySelf: false, IsGroup: false,
			GroupId: "777777777777", SenderId: "813466966", ReceiverId: "2788092443",
			Content: "46"},
		{IsMySelf: false, IsGroup: true,
			GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443",
			Content: "44"},
	}
	go func() {
		for _, msg := range testmessages {
			smsg := msg
			gapTime(500 * time.Millisecond)
			recvChan <- &smsg
			//gapOneSecond()
		}
		time.Sleep(3 * time.Second)
		processor.Close()
	}()
	// handle recv send chan
	go func() {
		for sendMsg := range sendChan {
			fmt.Println("发送消息: ")
			fmt.Printf("msg: %s  rawstruct: %#v\n\n", sendMsg.Content, sendMsg)
		}
	}()

	processor.Block() // 阻塞
	cache.GetCache().Close()
}

func gapOneSecond() {
	//fmt.Println("gap one second")
	time.Sleep(1 * time.Second)
}

func gapOneMinute() {
	time.Sleep(1 * time.Minute)
}

func gapTime(timedura time.Duration) {
	time.Sleep(timedura) // 暂停两秒发送消息
}
