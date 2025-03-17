// Package admin
// @Author Clover
// @Data 2025/3/10 下午4:55:00
// @Desc 管理员模块
package admin

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/msgutil"
	"strings"
)

type Admin struct {
	cache *cache.Cache
	*plugins.Plugin
}

// 同级action
var atSomeOneM = matcher.Custom{MatchFunc: func(msg *message.Message) bool {
	if msg == nil || msg.Content == "" {
		return false
	}
	return msgutil.HasPrefix(msg.Content, "@", false) // trim会把空格都排除
}}

var AA *Admin

func init() {
	admin := &Admin{
		cache:  cache.GetCache(),
		Plugin: plugins.DefaultPlugin("admin").AsLevel(plugins.VeryHighLevel).AsEnable(),
	}
	adminJudge := matcher.Custom{MatchFunc: func(msg *message.Message) bool {
		return admin.cache.HasAdminUserId(msg.WxId)
	}}
	allowText := matcher.BaseMatcher{AllowMsgType: message.MsgTypeText}
	isSelfJudge := matcher.BaseMatcher{
		Rules: matcher.IsSelf,
	}
	// op指令 基础规则
	opM := matcher.Default().And().N(allowText, matcher.PrefixMatcher{Prefix: "op", IsCaseSensitive: false, IsCut: true}, matcher.Default().Or().N(isSelfJudge, adminJudge))
	// op父操作
	opMulAction := plugins.DefaultActionHandler("op", true).AsMatcher(opM.And().N(matcher.DefaultNot(atSomeOneM)))

	help := helpFunc(isSelfJudge, adminJudge)
	// 删除管理员
	delOp := delOpFunc(isSelfJudge, adminJudge, admin)
	// op list
	opList := opListFunc(isSelfJudge, adminJudge, admin)
	// op -p 插件管理领域
	_, opPBase := opPluginFunc(isSelfJudge, adminJudge) // op -p
	// op -p list 插件列表
	PL := PLFunc()
	// op -p on 启用某插件
	onP := onPFunc()
	// op -p off
	offP := offPFunc()

	opPBase.AsChild(PL)   // 显示插件列表
	opPBase.AsChild(onP)  // 启用某插件
	opPBase.AsChild(offP) // 禁用插件

	// 子action
	opMulAction.AsChild(delOp)
	opMulAction.AsChild(opList)
	opMulAction.AsChild(opPBase) // 插件管理父行动
	opMulAction.AsChild(help.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		recvMsg.Content = admin.help()
		return *recvMsg, nil
	}))

	opAddM := opM.And().N(atSomeOneM)
	opAddByAt := plugins.DefaultActionHandler("op_add", true).AsMatcher(opAddM)
	opAddByAt.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		content, ok := admin.addOpByAt(recvMsg)
		if !ok {
			logging.Debug("艾特方式添加管理员失败了", map[string]interface{}{"msg": *recvMsg})
		}
		recvMsg.Content = content
		return *recvMsg, nil
	})

	// 绑定action
	admin.Plugin.AsAction(opAddByAt).AsAction(opMulAction)
	AA = admin                                      // test 测试使用
	plugins.GetAutoRegister().RegisterPlugin(admin) // 注册插件
}

func PLFunc() *plugins.ActionHandler {
	PLM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "list", IsCut: true, IsCaseSensitive: false})
	PL := plugins.DefaultActionHandler("op -p list", true).AsMatcher(PLM).AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		var buf bytes.Buffer
		buf.WriteString("插件列表\n")
		register := plugins.GetAutoRegister()
		for i, p := range register.Plugins() {
			buf.WriteString(fmt.Sprintf("【%d 插件名称: %s \n状态: %v\n等级: 第%d级 】", i+1, (*p).GetName(), (*p).GetPluginOpt().Enable, (*p).GetPluginOpt().Level))
		}
		recvMsg.Content = buf.String()
		return *recvMsg, nil
	})
	return PL
}

func onPFunc() *plugins.ActionHandler {
	onPM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "on", IsCut: true, IsCaseSensitive: false})
	onP := plugins.DefaultActionHandler("op -p on", true).AsMatcher(onPM).AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		name := strings.TrimSpace(recvMsg.Content)
		ar := plugins.GetAutoRegister()
		if name == "admin" {
			recvMsg.Content = "管理员插件禁止操作"
			return *recvMsg, nil
		}
		b := ar.EnableByName(name) // 启用某插件
		if b {
			recvMsg.Content = fmt.Sprintf("启用 %s 成功", name)
		} else {
			recvMsg.Content = fmt.Sprintf("启用 %s 失败", name)
		}
		return *recvMsg, nil
	})
	return onP
}

func offPFunc() *plugins.ActionHandler {
	offPM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "off", IsCut: true, IsCaseSensitive: false})
	offP := plugins.DefaultActionHandler("op -p on", true).AsMatcher(offPM).AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		name := strings.TrimSpace(recvMsg.Content)
		ar := plugins.GetAutoRegister()
		if name == "admin" {
			recvMsg.Content = "管理员插件禁止操作"
			return *recvMsg, nil
		}
		b := ar.DisableByName(name) // 启用某插件
		if b {
			recvMsg.Content = fmt.Sprintf("禁用 %s 成功", name)
		} else {
			recvMsg.Content = fmt.Sprintf("禁用 %s 失败", name)
		}
		return *recvMsg, nil
	})
	return offP
}

func helpFunc(isSelfJudge matcher.BaseMatcher, adminJudge matcher.Custom) *plugins.ActionHandler {
	helpM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "help", IsCaseSensitive: false, IsCut: true}, matcher.Default().Or().N(isSelfJudge, adminJudge))
	help := plugins.DefaultActionHandler("op help", true).AsMatcher(helpM)
	return help
}

func delOpFunc(isSelfJudge matcher.BaseMatcher, adminJudge matcher.Custom, admin *Admin) *plugins.ActionHandler {
	delM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "-d", IsCaseSensitive: false, IsCut: true}, matcher.Default().Or().N(isSelfJudge, adminJudge))
	delOp := plugins.DefaultActionHandler("op -d", true).AsMatcher(delM)
	delOp.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		content, ok := admin.delOpByAt(recvMsg)
		if !ok {
			logging.Debug("艾特方式删除管理员失败了", map[string]interface{}{"msg": *recvMsg})
		}
		recvMsg.Content = content
		return *recvMsg, nil
	})
	return delOp
}

func opPluginFunc(isSelfJudge matcher.BaseMatcher, adminJudge matcher.Custom) (matcher.DefaultMatcher, *plugins.ActionHandler) {
	pluginM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "-p", IsCaseSensitive: false, IsCut: true}, matcher.Default().Or().N(isSelfJudge, adminJudge))
	pluginOp := plugins.DefaultActionHandler("op -p", true).AsMatcher(pluginM)
	return pluginM, pluginOp
}

func opListFunc(isSelfJudge matcher.BaseMatcher, adminJudge matcher.Custom, admin *Admin) *plugins.ActionHandler {
	opListM := matcher.Default().And().N(matcher.PrefixMatcher{Prefix: "list", IsCaseSensitive: false, IsCut: true}, matcher.Default().Or().N(isSelfJudge, adminJudge))
	opList := plugins.DefaultActionHandler("op list", true).AsMatcher(opListM)
	opList.Action = func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		var buf bytes.Buffer
		buf.WriteString("管理员列表\n")
		idList := admin.cache.AdminIdList()
		if idList == nil || len(idList) == 0 {
			recvMsg.Content = "暂无管理员"
			return *recvMsg, nil
		}
		for _, wxid := range idList {
			m := admin.Cli.GetMember(wxid, true)
			if m != nil {
				buf.WriteString(fmt.Sprintf("name: %s, id: %s", m.NickName, m.Wxid))
			}
		}
		recvMsg.Content = buf.String()
		return *recvMsg, nil
	}
	return opList
}

func (a *Admin) help() string {
	var buf bytes.Buffer
	buf.WriteString("管理模块手册: \n")

	buf.WriteString("添加管理员 op <@someone>\n")
	buf.WriteString("移除管理员 op -d <@someone>\n")

	buf.WriteString("显示管理员 op list\n")
	buf.WriteString("显示模块状态 op -p list\n")

	buf.WriteString("启用模块 op -p on <plugin name>\n")
	buf.WriteString("禁用模块 op -p off <plugin name>\n")

	//buf.WriteString("添加群组白名单(在群聊中使用) op -w \n")
	//buf.WriteString("移除群组白名单(在群聊中使用) op -w off \n")
	////buf.WriteString("显示群组白名单 op -w list\n")
	//
	//buf.WriteString("添加群组黑名单(在群聊中使用) op -b \n")
	//buf.WriteString("移除群组黑名单(在群聊中使用) op -b off\n")
	////buf.WriteString("显示群组黑名单 op -b list\n")
	//
	//buf.WriteString("添加用户黑名单 op -u kick <@someone>\n")
	//buf.WriteString("移除用户黑名单 op -u save <@someone>\n")
	//buf.WriteString("显示用户黑名单 op -u \n")
	return buf.String()
}

func (a *Admin) addOpByAt(msg *message.Message) (string, bool) {
	if !msg.IsGroup {
		return "", false
	}
	if msg.RoomAts != nil && len(msg.RoomAts) > 0 && msg.RoomAts[0] != nil {
		a.cache.AddAdminUserId(msg.RoomAts[0].Wxid) // 添加管理
		return fmt.Sprintf("添加管理员%s成功 id: %s", msg.RoomAts[0].NickName, msg.RoomAts[0].Wxid), true
	}
	return "failed", true
}

func (a *Admin) delOpByAt(msg *message.Message) (string, bool) {
	if !msg.IsGroup {
		return "", false
	}
	if msg.RoomAts != nil && len(msg.RoomAts) > 0 && msg.RoomAts[0] != nil {
		a.cache.DeleteAdminUserId(msg.RoomAts[0].Wxid)
		return fmt.Sprintf("删除管理员%s成功 id: %s", msg.RoomAts[0].NickName, msg.RoomAts[0].Wxid), true
	}
	return "failed", true
}
