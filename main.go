package main

import (
	"flag"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/skip2/go-qrcode"
	"time"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/adapter"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/onebot/httpapi"
)

func main() {
	// 是否开启调试模式
	debugflag := flag.Bool("debug", false, "debug mode")
	// 是否开启 http服务
	httpMode := flag.Bool("http", false, "http mode")
	// 是否打印 qrcode
	isPrintQr := flag.Bool("qrcode", false, "qrcode mode")
	flag.Parse()
	if *debugflag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
	// 在初始化完成后输出所有缓冲日志
	logging.Logger.Flush(zerolog.GlobalLevel())
	logging.Logger.SetActive(false) // 取消缓存，正常日志输出

	defer func() {
		if r := recover(); r != nil {
			logging.Close() // 保存日志
			logging.Fatal("Recovered from panic", 1, map[string]interface{}{"panic": r})
		}
	}()

	bot := openwechat.DefaultBot(openwechat.Desktop)
	rbot := rikkabot.GetDefaultBot()

	//// 注册登陆二维码回调
	//bot.UUIDCallback = openwechat.PrintlnQrcodeUrl
	// 注册登陆二维码回调

	var isNotUUidCallback bool
	bot.UUIDCallback = func(uuid string) {
		url := openwechat.GetQrcodeUrl(uuid)
		rbot.SetloginUrl(url)
		logging.Warn("登录地址: " + url)
		if *isPrintQr {
			consoleQrCodeand(uuid)
		}
		isNotUUidCallback = true
		// 正向http  http上报器
		if *httpMode {
			httpapi.RunHttp(rbot)
			rbot.StartHandleEvent() // 处理事件

			rbot.PushLoginNoticeEvent() // 推送登录回调通知
		}
	}

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func() {
		err := reloadStorage.Close()
		if err != nil {
			logging.Fatal("get reload storage err", 1, map[string]interface{}{"err": err.Error()})
		}
	}()
	logging.Warn("请在手机中确认登录 or 扫码登录")
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		logging.Error("bot.PushLogin() error", map[string]interface{}{"openwechat bot error": err.Error()})
		return
	}

	a := adapter.NewAdapter(bot, rbot)

	a.HandleCovert() // 消息转换
	defer a.Close()

	// 正向http  http上报器
	if *httpMode && !isNotUUidCallback {
		httpapi.RunHttp(rbot)
		rbot.StartHandleEvent() // 处理事件
	}

	if !*httpMode {
		rbot.Start()
	}

	go func() {
		for {
			if !bot.Alive() {
				rbot.PushLogOutNoticeEvent(1101, "open-wechat客户端掉线")
				time.Sleep(1 * time.Second) // 1s 延迟退出
				rbot.ExitWithErr(1101, "open-wechat客户端掉线")
				return
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	err := rbot.Block()
	if err != nil {
		logging.WarnWithErr(err, "rikka bot.Block() error")
	}
}

func consoleQrCodeand(uuid string) {
	url := openwechat.GetQrcodeUrl(uuid)
	q, _ := qrcode.New(url, qrcode.Low)
	fmt.Println(q.ToString(true))
}
