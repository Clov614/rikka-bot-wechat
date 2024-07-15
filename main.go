package main

import (
	"bufio"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"os"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/adapter"
)

func main() {
	bot := openwechat.DefaultBot(openwechat.Desktop)

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer reloadStorage.Close()
	println("请在手机中确认登录")
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		println(fmt.Println(err))
		return
	}

	rbot := rikkabot.GetDefaultBot()

	a := adapter.NewAdapter(bot, rbot)
	a.HandleCovert() // 消息转换
	defer a.Close()

	rbot.Start()

	go func() {
		println("退出请输入q 或者 exit")
		input := bufio.NewScanner(os.Stdin)
		for input.Scan() {
			if input.Text() == "exit" || input.Text() == "quit" || input.Text() == "q" {
				rbot.Exit()
				bot.Exit()
			}
		}
	}()

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	rbot.Block()
	bot.Block()

}
