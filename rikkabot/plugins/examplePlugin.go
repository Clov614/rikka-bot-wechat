// @Author Clover
// @Data 2024/7/6 下午11:21:00
// @Desc 系统自带的插件
package plugins

import (
	"fmt"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/cache"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/register"
)

func init() {
	adminPlugin := AdminPlugin{}
	adminPlugin.PluginName = "管理员模块"
	adminPlugin.ProcessRules = &control.ProcessRules{IsCallMe: true, IsAdmin: true, EnableGroup: true,
		ExecOrder: []string{"add whitelist", "加入白名单"}}
	adminPlugin.Once = func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		reply := adminPlugin.addWhiteGroup(recvmsg)
		sendMsg <- reply
	}

	// 注册插件
	register.RegisterPlugin("admin_whitelist_add", &adminPlugin.OnceDialog)

	testLongDialogPlugin()
}

// 管理员功能相关
type AdminPlugin struct {
	control.OnceDialog
}

// 添加白名单
func (ap *AdminPlugin) addWhiteGroup(msg message.Message) (reply *message.Message) {
	if msg.IsGroup {
		// 添加白名单
		c := cache.GetCache()
		c.AddWhiteGroupId(msg.GroupId)
		msg.RawContext = fmt.Sprintf("添加白名单成功！群聊: %s", msg.GroupId)
	} else {
		msg.RawContext = "仅能在群聊中添加白名单"
	}

	reply = &msg
	return
}

// 长对话测试
type LongDialogPlugin struct {
	control.LongDialog
}

func testLongDialogPlugin() {
	testLongPlugin := LongDialogPlugin{}
	testLongPlugin.PluginName = "长对话测试"
	testLongPlugin.ProcessRules = &control.ProcessRules{IsCallMe: true, IsAdmin: false, EnableGroup: true,
		ExecOrder: []string{"test long", "长对话测试"}}
	msgBuf := testLongPlugin.MsgBuf

	testLongPlugin.Long = func(firstMsg message.Message, recvMsg <-chan message.Message, sendMsg chan<- *message.Message) {
		context := firstMsg.RawContext
		if context != "" {
			msgBuf.WriteString(fmt.Sprintf("回复长对话消息 + %s,\n", context))
			testLongPlugin.SendText(firstMsg.MetaData, msgBuf.String())
			msgBuf.Reset() // 清空构建的消息
			msgBuf.WriteString("接下来请发送 42+2等于多少")
			testLongPlugin.SendText(firstMsg.MetaData, msgBuf.String())
			msgBuf.Reset()
		}

		if msg, ok := <-recvMsg; ok {
			context := msg.RawContext
			if context == "44" {
				msgBuf.WriteString("没错，答对啦")
				testLongPlugin.SendText(msg.MetaData, msgBuf.String())
				msgBuf.Reset()
			} else {
				msgBuf.WriteString(fmt.Sprintf("很遗憾，答错了, 你的回答是：%s", context))
				testLongPlugin.SendText(msg.MetaData, msgBuf.String())
				msgBuf.Reset()
			}
		} else {
			// 消息通道关闭了
			return
		}
	}

	register.RegisterPlugin("long_dialog_plugin_test", &testLongPlugin)
}
