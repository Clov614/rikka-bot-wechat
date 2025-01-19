// Package httpapi
// @Author Clover
// @Data 2024/7/21 下午6:05:00
// @Desc 测试 http server and http poster
package httpapi

import (
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/adapter"
	"github.com/eatmoreapple/openwechat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"testing"
)

func TestHttpPost(t *testing.T) {
	// 测试开启debug模式
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	bot := openwechat.DefaultBot(openwechat.Desktop)

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func() {
		err := reloadStorage.Close()
		if err != nil {
			log.Debug().Err(err).Msg("get reload storage err")
			logging.Fatal("get reload storage err", 1)
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
	RunHttp(rbot)

	rbot.StartHandleEvent()

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	err := rbot.Block()
	if err != nil {
		logging.WarnWithErr(err, "rikka bot.Block() error")
	}
}
