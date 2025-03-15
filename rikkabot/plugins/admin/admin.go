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
)

type Admin struct {
	cache *cache.Cache
	*plugins.Plugin
}

func init() {
	admin := &Admin{
		cache:  cache.GetCache(),
		Plugin: plugins.DefaultPlugin("admin").AsLevel(plugins.VeryHighLevel).AsEnable(),
	}
	adminJudge := &matcher.CustomMatcher{MatchFunc: func(msg *message.Message) bool {
		return admin.cache.HasAdminUserId(msg.WxId)
	}}
	actionHandler := plugins.DefaultActionHandler("op", true).AsMatcher(matcher.DefaultAnd(&matcher.PrefixMatcher{Prefix: "op", IsCaseSensitive: false, IsCut: true}, matcher.DefaultOr(&matcher.BaseMatcher{
		Rules:        matcher.IsSelf,
		AllowMsgType: message.MsgTypeText,
	}, adminJudge)))
	help := plugins.DefaultActionHandler("help", true).AsMather(matcher.DefaultAnd(&matcher.PrefixMatcher{Prefix: "help", IsCaseSensitive: false, IsCut: true}, matcher.DefaultOr(&matcher.BaseMatcher{
		Rules:        matcher.IsSelf,
		AllowMsgType: message.MsgTypeText,
	}, adminJudge)))
	// 删除管理员
	delOp := plugins.DefaultActionHandler("-d", true).AsMather(matcher.DefaultAnd(&matcher.PrefixMatcher{Prefix: "-d", IsCaseSensitive: false, IsCut: true}, matcher.DefaultOr(&matcher.BaseMatcher{
		Rules:        matcher.IsSelf,
		AllowMsgType: message.MsgTypeText,
	}, adminJudge)))
	delOp.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		content, ok := admin.delOpByAt(recvMsg)
		if !ok {
			logging.Debug("艾特方式删除管理员失败了", map[string]interface{}{"msg": *recvMsg})
		}
		recvMsg.Content = content
		return *recvMsg, nil
	})
	actionHandler.AsChild(delOp)
	// 子action
	actionHandler.AsChild(help.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		recvMsg.Content = admin.help()
		return *recvMsg, nil
	}))

	// 同级action
	opAddByAt := plugins.DefaultActionHandler("op_add", true).AsMatcher(matcher.DefaultAnd(&matcher.PrefixMatcher{Prefix: "op", IsCaseSensitive: false, IsCut: true}, matcher.DefaultOr(&matcher.BaseMatcher{
		Rules:        matcher.IsSelf,
		AllowMsgType: message.MsgTypeText,
	}, adminJudge)))
	opAddByAt.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
		content, ok := admin.addOpByAt(recvMsg)
		if !ok {
			logging.Debug("艾特方式添加管理员失败了", map[string]interface{}{"msg": *recvMsg})
		}
		recvMsg.Content = content
		return *recvMsg, nil
	})

	// 绑定action
	admin.Plugin.AsAction(opAddByAt).AsAction(actionHandler)
	plugins.GetAutoRegister().RegisterPlugin(admin) // 注册插件
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

	buf.WriteString("添加群组白名单(在群聊中使用) op -w \n")
	buf.WriteString("移除群组白名单(在群聊中使用) op -w off \n")
	//buf.WriteString("显示群组白名单 op -w list\n")

	buf.WriteString("添加群组黑名单(在群聊中使用) op -b \n")
	buf.WriteString("移除群组黑名单(在群聊中使用) op -b off\n")
	//buf.WriteString("显示群组黑名单 op -b list\n")

	buf.WriteString("添加用户黑名单 op -u kick <@someone>\n")
	buf.WriteString("移除用户黑名单 op -u save <@someone>\n")
	buf.WriteString("显示用户黑名单 op -u \n")
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
