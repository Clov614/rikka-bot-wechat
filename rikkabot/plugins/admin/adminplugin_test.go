// @Author Clover
// @Data 2024/7/15 下午9:18:00
// @Desc deprecated
package plugin_admin_test

import (
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/common"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor"
	"github.com/eatmoreapple/openwechat"
	"testing"
)

// deprecated 测试管理员模块
func TestAdminPlugin(t *testing.T) {
	bot := getOpenBot()

	common.InitSelf(bot)
	_ = common.GetSelf()

	recvChan := make(chan *message.Message)
	sendChan := make(chan *message.Message)
	p := processor.NewProcessor()
	go p.DispatchMsg(recvChan, sendChan)

	// 测试小号：2788092443
	// 测试大号：813466966
	// 测试群组：813467281

	const (
		nickname = "Rikka"
	)

	tests := []struct {
		recvMsg     *message.Message
		wantContent string
	}{{
		// 管理员基础功能
		// 00 测试 添加管理员 成功
		recvMsg:     &message.Message{Content: "/rikka admin add " + nickname, SenderId: "2788092443", GroupId: "813467281", ReceiverId: "813467281", IsGroup: true},
		wantContent: fmt.Sprintf("添加成功，用户( %s )id( %d )成为管理员", nickname, 813466966),
	}}

	go func() {
		for _, test := range tests {
			recvChan <- test.recvMsg
		}
	}()

	for {
		for i, test := range tests {
			sendMsg := <-sendChan
			if sendMsg.Content != test.wantContent {
				t.Errorf("i = %d, want = %s, got = %s", i, test.wantContent, sendMsg.Content)
			}
		}
	}
}

func getOpenBot() *openwechat.Bot {
	bot := openwechat.DefaultBot(openwechat.Desktop)

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func() {
		err := reloadStorage.Close()
		if err != nil {
			logging.Fatal("reload storage err", 1)
		}
	}()
	println("请在手机中确认登录")
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		println(fmt.Println(err))
		return nil
	}
	return bot
}
