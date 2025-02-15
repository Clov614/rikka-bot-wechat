package main

import (
	"context"
	"flag"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/adapter"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/onebot/httpapi"
	wcf "github.com/Clov614/wcf-rpc-sdk"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"time"
)

func main() {

	// todo 支持自动获取sdk.dll 自动完成注入
	// 是否自动注入
	autoInject := flag.Bool("autoInject", false, "注入sdk.dll")
	// 是否开启调试模式
	debugflag := flag.Bool("debug", false, "debug mode")
	// 是否开启wcf调试模式
	wcfdebugflag := flag.Bool("wcfdebug", false, "wcf debug mode")
	// 是否开启 http服务
	httpMode := flag.Bool("http", false, "http mode")
	// 是否开启 rikkabot
	botMode := flag.Bool("bot", false, "bot mode(using to start rikkabot and also http can run)")
	flag.Parse()
	if *debugflag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
	//// 在初始化完成后输出所有缓冲日志
	//logging.Logger.Flush(zerolog.GlobalLevel())
	//logging.Logger.SetActive(false) // 取消缓存，正常日志输出

	defer func() {
		if r := recover(); r != nil {
			logging.Close() // 保存日志
			logging.Fatal("Recovered from panic", 1, map[string]interface{}{"panic": r})
		}
	}()
	ctx := context.Background()
	cli := wcf.NewClient(30, *autoInject, false)
	cli.Run(*wcfdebugflag) // 运行wcf客户端

	rbot := rikkabot.NewRikkaBot(ctx, cli, *debugflag)
	a := adapter.NewAdapter(ctx, cli, rbot)
	a.HandleCovert() // 消息转换e11

	// 正向http  http上报器
	if *httpMode {
		httpapi.RunHttp(rbot)
		rbot.StartHandleEvent() // 处理事件
	}

	if !*httpMode || *botMode { // http 不启动情况 或者 bot模式启动 情况下 启动bot
		rbot.Start()
	}

	go func() {
		maxRetries := 7
		retryInterval := 1 * time.Second
		for i := 0; i < maxRetries; i++ {
			if cli.IsLogin() {
				logging.Info("微信登录成功")
				return // 登录成功，退出循环
			}
			logging.Warn("微信未登录或掉线，正在重试...", map[string]interface{}{
				"retry_count": i + 1,
			})
			time.Sleep(retryInterval)
			retryInterval *= 2 // 指数退避
		}
		// 重试7次后仍然失败
		rbot.PushLogOutNoticeEvent(1101, "微信未登录或掉线")
		time.Sleep(1 * time.Second) // 1s 延迟退出
		rbot.ExitWithErr(1101, "微信未登录或掉线")
	}()

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	err := rbot.Block()
	if err != nil {
		logging.WarnWithErr(err, "rikka bot.Block() error")
	}
}
