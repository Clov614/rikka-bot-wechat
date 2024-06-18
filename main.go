package main

import (
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"time"
	"wechat-demo/autoResendMsg"
)

func main() {
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式

	resendMsg := autoResendMsg.Init()

	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		//if msg.IsText() && msg.Content == "ping" {
		//	msg.ReplyText()
		//}
		if resendMsg.IsReply(msg) { // 自动回复消息设置
			gapTime()
			msg.ReplyText(resendMsg.CustomMsg)
		}
	}
	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err := bot.Login(); err != nil {
		fmt.Println(err)
		return
	}

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取所有的好友
	friends, err := self.Friends()
	fmt.Println(friends, err)

	// 获取所有的群组
	groups, err := self.Groups()
	fmt.Println(groups, err)

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

func gapTime() {
	time.Sleep(2000 * time.Millisecond) // 暂停两秒发送消息
}

func is2MeMsg(bot *openwechat.Bot, msg *openwechat.Message) bool {
	self, _ := bot.GetCurrentUser()

	if msg.ToUserName == self.UserName {
		return true
	}
	return false
}
