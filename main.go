package main

import (
	"bufio"
	"github.com/eatmoreapple/openwechat"
	"os"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/adapter"
	"wechat-demo/rikkabot/logging"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			logging.Close() // 保存日志
			logging.Fatal("Recovered from panic", 1, map[string]interface{}{"panic": r})
		}
	}()

	bot := openwechat.DefaultBot(openwechat.Desktop)

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func() {
		err := reloadStorage.Close()
		if err != nil {
			logging.Fatal("get reload storage err", 1, map[string]interface{}{"err": err})
		}
	}()
	println("请在手机中确认登录")
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		logging.Error("bot.PushLogin() error", map[string]interface{}{"openwechat bot error": err.Error()})
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
	err := rbot.Block()
	if err != nil {
		logging.Warn("rikka bot.Block() error", map[string]interface{}{"err": err.Error()})
	}
}
