package main

import (
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"os"
	"strconv"
	"time"
	"wechat-demo/autoResendMsg"
)

func main() {
	args := os.Args

	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式

	resendMsg := autoResendMsg.Init()

	if args != nil && len(args) == 1 {
		customMsg := args[0]
		resendMsg.CustomMsg = customMsg // 自定义消息 命令行设置
	}

	if args != nil && len(args) >= 2 {
		customMsg := args[0]
		if customMsg != "" {
			resendMsg.CustomMsg = customMsg
		}
		ttl := args[1]
		ttlNum, err := strconv.Atoi(ttl)
		if err != nil {
			fmt.Errorf("convErr: ttl (arg[1]) not a integer, %s", err)
			os.Exit(1)
		}
		resendMsg.TTL = time.Hour * time.Duration(ttlNum)
	}

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
