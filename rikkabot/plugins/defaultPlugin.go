// Package plugins
// @Author Clover
// @Data 2024/7/6 下午11:21:00
// @Desc 系统自带的插件
package plugins

import (
	"fmt"
	"wechat-demo/rikkabot/message"
	_ "wechat-demo/rikkabot/plugins/admin" // 需要副作用 init注册方法
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/register"
)

func init() {
	testPlugin := TestPlugin{}
	testPlugin.PluginName = "管理员模块"
	testPlugin.ProcessRules = &control.ProcessRules{IsCallMe: true, IsAdmin: true, EnableGroup: true,
		ExecOrder: []string{"add whitelist", "加入白名单"}}

	// 注册插件
	register.RegistPlugin("admin_whitelist_add", &testPlugin.OnceDialog)

	testLongDialogPlugin()
}

// TestPlugin 管理员功能相关
type TestPlugin struct {
	control.OnceDialog
}

// LongDialogPlugin 长对话测试
type LongDialogPlugin struct {
	control.LongDialog
}

func testLongDialogPlugin() {
	testLongPlugin := LongDialogPlugin{}
	testLongPlugin.PluginName = "长对话测试"
	testLongPlugin.ProcessRules = &control.ProcessRules{IsCallMe: true, IsAdmin: false, EnableGroup: true,
		ExecOrder: []string{"test long", "长对话测试"}}
	msgBuf := testLongPlugin.MsgBuf // 获取 msg buffer

	testLongPlugin.Long = func(firstMsg message.Message, recvMsg <-chan message.Message, sendMsg chan<- *message.Message) {
		context := firstMsg.Content
		if context != "" {
			msgBuf.WriteString(fmt.Sprintf("回复长对话消息 + %s,\n", context))
			testLongPlugin.SendText(firstMsg.MetaData, msgBuf.String())
			msgBuf.Reset() // 清空构建的消息
			msgBuf.WriteString("接下来请发送 42+2等于多少")
			testLongPlugin.SendText(firstMsg.MetaData, msgBuf.String())
			msgBuf.Reset()
		} else {
			msgBuf.WriteString("长对话测试开始")
			testLongPlugin.SendText(firstMsg.MetaData, msgBuf.String())
			msgBuf.Reset() // 清空构建的消息
		}

		if msg, ok := <-recvMsg; ok {
			context := msg.Content
			if context == "44" {
				msgBuf.WriteString("没错，答对啦")
				testLongPlugin.SendText(msg.MetaData, msgBuf.String())
				msgBuf.Reset()
			} else {
				msgBuf.WriteString("很遗憾，答错了, 你的回答是: " + context)
				testLongPlugin.SendText(msg.MetaData, msgBuf.String())
				msgBuf.Reset()
			}
		} else {
			// 消息通道关闭了
			return
		}
	}

	register.RegistPlugin("long_dialog_plugin_test", &testLongPlugin)
}
