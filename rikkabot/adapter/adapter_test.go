// @Author Clover
// @Data 2024/7/5 下午6:09:00
// @Desc
package adapter

import (
	"fmt"
	"github.com/eatmoreapple/openwechat"
	_ "image/png"
	"strings"
	"testing"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/utils/serializer"
)

// 测试是否转发接收到的消息，二次封装是否正常
func TestReceiveRawMsg(t *testing.T) {

}

// 测试rikkaBot发送的消息是否能转发给 openwechat
func TestSendRikkaMsg(t *testing.T) {

}

// 获取 第三方平台发送来的经转换后的消息json，保存于同级目录下
func TestGetMsgJson(t *testing.T) {
	bot := openwechat.DefaultBot(openwechat.Desktop)

	a := NewAdapter(bot, rikkabot.GetDefaultBot())

	HandleCovert(a)
	defer a.Close()

	t.Logf("Start test\n")

	go func() {
		// 保存 json
		cnt := 7
		for rikkaMsg := range a.selfBot.GetReqMsgRecvChan() {
			fmt.Printf("save json recv rikkaMsg: %#v\n", *rikkaMsg)
			err := serializer.Save("./test/cacheMsg", fmt.Sprintf("rikkaMsg%d", cnt), rikkaMsg)
			senderId := rikkaMsg.RawMsg.GetSenderId()
			receiverId := rikkaMsg.RawMsg.GetReceiverId()
			t.Logf("sender id: %s, receiver id: %s\n", senderId, receiverId)
			if err != nil {
				t.Logf("err: %v", err)
			}
			cnt++
		}
	}()

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer reloadStorage.Close()
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		t.Error(err)
		return
	}

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		t.Logf("err: %v", err)
		return
	}

	if self == nil {
		t.Errorf("GetCurrentUser err")
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
	fmt.Println("hello")
	// Output: hello
}

// 测试发送文本消息
func TestSendMsg(t *testing.T) {
	bot := openwechat.DefaultBot(openwechat.Desktop)

	a := NewAdapter(bot, rikkabot.GetDefaultBot())

	HandleCovert(a)
	defer a.Close()

	t.Logf("Start test\n")

	// test 实际回复功能测试
	doneEcho := make(chan struct{})
	go func() {
		defer close(doneEcho)
		err2 := imgEcho(a, doneEcho)
		if err2 != nil {
			t.Logf("echo err: %v", err2)
		}

	}()

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer reloadStorage.Close()
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		t.Error(err)
		return
	}

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		t.Logf("err: %v", err)
		return
	}

	if self == nil {
		t.Errorf("GetCurrentUser err")
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
	fmt.Println("hello")
	// Output: hello
}

func echo(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			if rikkaMsg.MetaType != message.MsgRequest {
				return fmt.Errorf("metaType err")
			}
			rikkaMsg.MetaType = message.MsgResponse
			context := rikkaMsg.RawContext
			if strings.HasPrefix(context, "echo ") {
				trimed := strings.TrimPrefix(context, "echo ")
				rikkaMsg.RawContext = trimed
				rikkaMsg.Raw = []byte(rikkaMsg.RawContext)
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg
			}
		}
	}
	return nil
}

func doubleEcho(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			if rikkaMsg.MetaType != message.MsgRequest {
				return fmt.Errorf("metaType err")
			}
			rikkaMsg.MetaType = message.MsgResponse
			context := rikkaMsg.RawContext
			if strings.HasPrefix(context, "echo ") {
				trimed := strings.TrimPrefix(context, "echo ")
				rikkaMsg.RawContext = trimed
				rikkaMsg.Raw = []byte(rikkaMsg.RawContext)
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg
			}
		}
	}
	return nil
}

func doubleEchoActive(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			if rikkaMsg.MetaType != message.MsgRequest {
				return fmt.Errorf("metaType err")
			}
			rikkaMsg.MetaType = message.MsgResponse
			context := rikkaMsg.RawContext
			if strings.HasPrefix(context, "echo ") {
				trimed := strings.TrimPrefix(context, "echo ")
				rikkaMsg.RawContext = trimed
				rikkaMsg.Raw = []byte(rikkaMsg.RawContext)
				rikkaMsg.RawMsg.GetRawMsg() // todo 主动发送消息的 rawMsg增强
			}
		}
	}
	return nil
}

func imgEcho(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			if rikkaMsg.MetaType != message.MsgRequest {
				return fmt.Errorf("metaType err")
			}
			rikkaMsg.MetaType = message.MsgResponse
			if rikkaMsg.Msgtype == message.MsgTypeImage {
				msg := rikkaMsg.RawMsg.GetRawMsg().(*openwechat.Message)
				msg.SaveFileToLocal("./test/testImg.jpg")
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg

			}

		}
	}
	return nil
}
