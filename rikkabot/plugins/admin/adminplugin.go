// Package plugin_admin
// @Author Clover
// @Data 2024/7/15 下午4:59:00
// @Desc 管理员默认方法
package plugin_admin

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/common"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control/dialog"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/register"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/msgutil"
)

func init() {
	registAdminPlugin() // 管理管基础功能
}

const (
	defaultAdminControl = "admin_control"
)

// 注册管理员基础功能
func registAdminPlugin() {
	rules := &control.ProcessRules{IsCallMe: true, IsAdmin: true, IsAtMe: true, EnableGroup: true,
		ExecOrder: []string{"admin"}, EnableMsgType: []message.MsgType{message.MsgTypeText}}
	adminPlugin := AdminPlugin{
		onceDialog: dialog.InitOnceDialog("管理员基础功能", rules),
	}

	// 对话逻辑
	adminPlugin.onceDialog.SetOnceFunc(func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		if adminPlugin.cache == nil {
			adminPlugin.cache = adminPlugin.onceDialog.Cache // 运行时获取 缓存指针
		}
		if adminPlugin.user == nil {
			adminPlugin.user = adminPlugin.onceDialog.Self // 运行时获取  用户（自身）指针
		}
		reply := adminPlugin.handleAdminCommand(recvmsg)
		adminPlugin.onceDialog.SendText(recvmsg.MetaData, reply) // send msg
	})

	register.RegistPlugin(defaultAdminControl, adminPlugin.onceDialog, 0)
}

func isChoice(cotent string, prefix string) bool {
	return msgutil.HasPrefix(cotent, prefix, true)
}

// AdminPlugin 管理员模块
type AdminPlugin struct {
	onceDialog *dialog.OnceDialog
	cache      *cache.Cache
	user       *common.Self
}

//region admin 管理员

// 根据nickname添加管理员
func (a AdminPlugin) addAdmin(nickname string) (reply string) {
	if strings.HasPrefix(nickname, "@") {
		nickname = msgutil.GetNicknameByAt(nickname)
	}
	friendId, err := a.user.GetFriendIdByNickname(nickname)
	if err != nil {
		return "添加管理员失败，错误：" + err.Error()
	}
	a.cache.AddAdminUserId(friendId)
	return fmt.Sprintf("添加成功，用户( %s )id( %s )成为管理员", nickname, friendId)
}

// 移除管理员
func (a AdminPlugin) deleteAdmin(nickname string) (reply string) {
	friendId, err := a.user.GetFriendIdByNickname(nickname)
	if err != nil {
		return "移除管理员失败，错误：" + err.Error()
	}
	a.cache.DeleteAdminUserId(friendId)
	return fmt.Sprintf("移除了，用户( %s )id( %s )的管理员权限", nickname, friendId)
}

// 显示管理员名单
func (a AdminPlugin) showAdminList() (reply string) {
	var buf bytes.Buffer
	buf.WriteString("赋权管理员:\n")
	adminIdList := a.cache.AdminIdList()
	l := len(adminIdList)
	lose := 0
	for _, adminid := range adminIdList {
		adminNickname, err := a.user.GetFriendNicknameById(adminid)
		if err != nil {
			fmt.Println("get admin nickname err: ", err)
			l--
			lose++
			continue
		}
		buf.WriteString(fmt.Sprintf("[%s, %s]\n", adminNickname, adminid))
	}
	buf.WriteString(fmt.Sprintf("匹配到的条数为：%d，失败条数：%d", l+lose, lose))
	return buf.String()
}

// 显示目前模块状况（启用，禁用）（不能禁用默认管理员控制模块）
func (a AdminPlugin) showPluginState() (reply string) {
	var buf bytes.Buffer
	buf.WriteString("插件状态(名称-状态):\n")
	enablePluginMap := a.cache.EnablePluginMap()
	for name, state := range enablePluginMap {
		if state {
			buf.WriteString(fmt.Sprintf("%s\t%s\n", name, "启用"))
		} else {
			buf.WriteString(fmt.Sprintf("%s\t%s\n", name, "禁用"))
		}
	}
	return buf.String()
}

// 启用模块
func (a AdminPlugin) enablePlugin(pluginname string) (reply string) {
	err := a.cache.EnablePlugin(pluginname)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("启用(%s)模块成功!", pluginname)
}

// 禁用模块
func (a AdminPlugin) disablePlugin(pluginname string) (reply string) {
	if pluginname == defaultAdminControl {
		return "禁止禁用管理员基础模块"
	}
	err := a.cache.DisablePlugin(pluginname)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("禁用(%s)模块成功!", pluginname)
}

//endregion

// 群组添加白名单
func (a AdminPlugin) addWhiteGroup(groupnickname string) (reply string) {
	groupId, err := a.user.GetGroupIdByNickname(groupnickname)
	if err != nil {
		return "添加群组白名单失败，错误：" + err.Error()
	}
	a.cache.AddWhiteGroupId(groupId)
	return fmt.Sprintf("添加成功，群组( %s )id( %s )进入白名单", groupnickname, groupId)
}

// 群组添加白名单（添加发送消息的群组）
func (a AdminPlugin) addWhiteGroupByMsg(msg message.Message) (reply string) {
	if !msg.IsGroup {
		return "不是群消息，无法添加群聊白名单"
	}
	a.cache.AddWhiteGroupId(msg.RoomId)

	return fmt.Sprintf("添加成功，群组( %s )id( %s )进入白名单", msg.MetaData.GetGroupNickname(), msg.RoomId)
}

// 群组移除白名单
func (a AdminPlugin) deleteWhiteGroup(groupnickname string) (reply string) {
	groupId, err := a.user.GetGroupIdByNickname(groupnickname)
	if err != nil {
		return "移除群组白名单失败，错误：" + err.Error()
	}
	a.cache.DeleteWhiteGroupId(groupId)
	return fmt.Sprintf("移除了，群组( %s )id( %s )的白名单", groupnickname, groupId)
}

// 群组移除白名单（移除发送消息的群组）
func (a AdminPlugin) deleteWhiteGroupByMsg(msg message.Message) (reply string) {
	if !msg.IsGroup {
		return "不是群消息，无法移除群聊白名单(请携带群名参数)"
	}
	a.cache.DeleteWhiteGroupId(msg.RoomId)
	return fmt.Sprintf("移除了，群组( %s )id( %s )的白名单", msg.MetaData.GetGroupNickname(), msg.RoomId)
}

// 显示群组白名单
func (a AdminPlugin) showWhiteGroup() (reply string) {
	var buf bytes.Buffer
	buf.WriteString("群组白名单:\n")
	groupIdList := a.cache.WhiteGroupIdList()
	l := len(groupIdList)
	lose := 0
	for _, groupId := range groupIdList {
		groupNickname, err := a.user.GetGroupNicknameById(groupId)
		if err != nil {
			fmt.Println("get group nickname error:", err)
			l--
			lose++
			continue
		}
		buf.WriteString(fmt.Sprintf("[%s, %s]\n", groupNickname, groupId))
	}
	buf.WriteString(fmt.Sprintf("匹配到的条数为：%d，失败条数：%d", l+lose, lose))
	return buf.String()
}

// 黑名单
// 添加该群组黑名单,群组内使用（不携带昵称）
func (a AdminPlugin) addBlackGroupByMsg(msg message.Message) (reply string) {
	if !msg.IsGroup {
		return "不是群聊无法添加群组黑名单(请携带群名参数)"
	}
	a.cache.AddBlackGroupId(msg.RoomId)
	return fmt.Sprintf("添加了，群组( %s )id( %s )的黑名单", msg.MetaData.GetGroupNickname(), msg.RoomId)
}

// 移除该群组黑名单,群组内使用（不携带昵称）
func (a AdminPlugin) deleteBlackGroupByMsg(msg message.Message) (reply string) {
	if !msg.IsGroup {
		return "不是群聊无法移除群组黑名单(请携带群名参数)"
	}
	a.cache.DeleteBlackGroupId(msg.RoomId)
	return fmt.Sprintf("移除了，群组( %s )id( %s )的黑名单", msg.MetaData.GetGroupNickname(), msg.RoomId)
}

// 添加群组黑名单，根据群昵称
func (a AdminPlugin) addBlackGroupByNickname(nickname string) (reply string) {
	groupId, err := a.user.GetGroupIdByNickname(nickname)
	if err != nil {
		return "添加群组黑名单失败，错误：" + err.Error()
	}
	a.cache.AddBlackGroupId(groupId)
	return fmt.Sprintf("添加了，群组( %s )id( %s )的黑名单", nickname, groupId)
}

// 移除群组黑名单，根据群昵称
func (a AdminPlugin) deleteBlackGroupByNickname(nickname string) (reply string) {
	groupId, err := a.user.GetGroupIdByNickname(nickname)
	if err != nil {
		return "移除群组黑名单失败，错误：" + err.Error()
	}
	a.cache.DeleteBlackGroupId(groupId)
	return fmt.Sprintf("移除了，群组( %s )id( %s )的黑名单", nickname, groupId)
}

// 显示群组黑名单
func (a AdminPlugin) showBlackGroup() (reply string) {
	var buf bytes.Buffer
	buf.WriteString("群组黑名单:\n")
	groupIdList := a.cache.BlackGroupIdList()
	l := len(groupIdList)
	lose := 0
	for _, groupId := range groupIdList {
		groupNickname, err := a.user.GetGroupNicknameById(groupId)
		if err != nil {
			fmt.Println("get group nickname error:", err)
			l--
			lose++
			continue
		}
		buf.WriteString(fmt.Sprintf("[%s, %s]\n", groupNickname, groupId))
	}
	buf.WriteString(fmt.Sprintf("匹配到的条数为：%d，失败条数：%d", l+lose, lose))
	return buf.String()
}

// 个人
// 添加用户黑名单，根据昵称
func (a AdminPlugin) addBlackUserByNickname(msg message.Message, nickname string) (reply string) {
	friendId, err := a.user.GetFriendIdByNickname(nickname)
	if friendId == "" { // 不是好友，尝试通过msg获取member id
		friendId, err = msg.MetaData.GetRoomNameByRoomId(nickname)
	}
	if friendId == "" && err != nil {
		return "添加用户黑名单失败，错误：" + err.Error()
	}
	a.cache.AddBlackUserId(friendId)
	return fmt.Sprintf("添加了，用户( %s )id( %s )的黑名单", nickname, friendId)
}

// 移除用户黑名单，根据昵称
func (a AdminPlugin) deleteBlackUserByNickname(msg message.Message, nickname string) (reply string) {
	friendId, err := a.user.GetFriendIdByNickname(nickname)
	if friendId == "" { // 不是好友，尝试通过msg获取member id
		friendId, err = msg.MetaData.GetRoomNameByRoomId(nickname)
	}
	if friendId == "" && err != nil {
		return "移除用户黑名单失败，错误：" + err.Error()
	}
	a.cache.DeleteBlackUserId(friendId)
	return fmt.Sprintf("移除了，用户( %s )id( %s )的黑名单", nickname, friendId)
}

// 显示用户黑名单
func (a AdminPlugin) showBlackUser() (reply string) {
	var buf bytes.Buffer
	buf.WriteString("用户黑名单:\n")
	userIdList := a.cache.BlackUserIdList()
	l := len(userIdList)
	lose := 0
	for _, userId := range userIdList {
		userNickname, err := a.user.GetFriendNicknameById(userId)
		if err != nil {
			fmt.Println("get user nickname error:", err)
			l--
			lose++
			buf.WriteString(fmt.Sprintf("[%s, %s]\n", userId, userId)) // 非好友群成员封禁信息
			continue
		}
		buf.WriteString(fmt.Sprintf("[%s, %s]\n", userNickname, userId))
	}
	buf.WriteString(fmt.Sprintf("匹配到的条数为：%d，群成员非好友条数：%d", l+lose, lose))
	return buf.String()
}

// 更新handleAtContentMapper方法
func (a AdminPlugin) handleAtContentMapper(fn interface{}, params ...interface{}) string {
	switch f := fn.(type) {
	case func(nickname string) string:
		nickname := params[1].(string)
		if strings.HasPrefix(nickname, "@") {
			nickname = msgutil.GetNicknameByAt(nickname) // 支持获取at后的用户名
		}
		return f(nickname)
	case func(msg message.Message, nickname string) string:
		nickname := params[2].(string)
		if strings.HasPrefix(nickname, "@") {
			nickname = msgutil.GetNicknameByAt(nickname) // 支持获取at后的用户名
		}
		return f(params[1].(message.Message), nickname)
	}
	return ""
}

func helpContent() string {
	var buf bytes.Buffer
	buf.WriteString("管理模块手册: (<call>: 机器人呼唤名)\n")
	buf.WriteString("统一前缀: <call> admin\n")

	buf.WriteString("添加管理员 add admin <user nickname>\n")
	buf.WriteString("移除管理员 del admin <user nickname>\n")

	buf.WriteString("显示管理员 show admin\n")
	buf.WriteString("显示模块状态 show plugin state\n")

	buf.WriteString("启用模块 plugin enable <plugin name>\n")
	buf.WriteString("禁用模块 plugin disable <plugin name>\n")

	buf.WriteString("添加群组白名单 white add <group name 可选>\n")
	buf.WriteString("移除群组白名单 white del <group name 可选>\n")
	buf.WriteString("显示群组白名单 white show\n")

	buf.WriteString("添加群组黑名单 black group add <group name 可选>\n")
	buf.WriteString("移除群组黑名单 black group del <group name 可选>\n")
	buf.WriteString("显示群组黑名单 black group show\n")

	buf.WriteString("添加用户黑名单 black user add <group name>\n")
	buf.WriteString("移除用户黑名单 black user del <group name>\n")
	buf.WriteString("显示用户黑名单 black user show\n")
	return buf.String()
}

// 将闭包捕获的函数提取为方法，减少闭包的使用
func (a AdminPlugin) handleAdminCommand(recvmsg message.Message) string {
	content := recvmsg.Content
	content = msgutil.TrimPrefix(content, "admin", false, true)
	switch {
	case isChoice(content, "add"):
		nickname := msgutil.TrimPrefix(content, "add", false, true)
		if nickname == "" {
			return "添加管理员示例: add admin <nickname>"
		}
		if !recvmsg.IsMySelf {
			return "仅有超级管理员(自己)，才能操作"
		}
		return a.addAdmin(nickname)
	case isChoice(content, "del"):
		nickname := msgutil.TrimPrefix(content, "del", false, true)
		if nickname == "" {
			return "移除管理员示例: del admin <nickname>"
		}
		if !recvmsg.IsMySelf {
			return "仅有超级管理员(自己)，才能操作"
		}
		return a.deleteAdmin(nickname)
	case isChoice(content, "show admin"):
		if !recvmsg.IsMySelf {
			return "仅有超级管理员(自己)，才能操作"
		}
		return a.showAdminList()
	case isChoice(content, "show plugin state"), isChoice(content, "show plugin status"):
		return a.showPluginState()
	case content == "show":
		return "show what? <admin> or <plugin state>\nExample: <callbot> admin show admin"
	case isChoice(content, "plugin"):
		pluginsContent := msgutil.TrimPrefix(content, "plugin", false, true)
		switch {
		case isChoice(pluginsContent, "enable"):
			return a.enablePlugin(msgutil.TrimPrefix(pluginsContent, "enable", false, true))
		case isChoice(pluginsContent, "disable"):
			return a.disablePlugin(msgutil.TrimPrefix(pluginsContent, "disable", false, true))
		default:
			return "未知插件管理命令"
		}
	case isChoice(content, "white"):
		whiteContent := msgutil.TrimPrefix(content, "white", false, true)
		switch {
		case isChoice(whiteContent, "add"):
			return a.addWhiteGroupByMsg(recvmsg)
		case isChoice(whiteContent, "del"):
			return a.deleteWhiteGroupByMsg(recvmsg)
		case isChoice(whiteContent, "show"):
			return a.showWhiteGroup()
		default:
			return "未知白名单命令"
		}
	case isChoice(content, "black"):
		blackContent := msgutil.TrimPrefix(content, "black", false, true)
		switch {
		case isChoice(blackContent, "group"):
			groupContent := msgutil.TrimPrefix(blackContent, "group", false, true)
			switch {
			case isChoice(groupContent, "add"):
				return a.handleAtContentMapper(a.addBlackGroupByNickname, groupContent)
			case isChoice(groupContent, "del"):
				return a.handleAtContentMapper(a.deleteBlackGroupByNickname, groupContent)
			case isChoice(groupContent, "show"):
				return a.showBlackGroup()
			default:
				return "未知黑名单群组命令"
			}
		case isChoice(blackContent, "user"):
			userContent := msgutil.TrimPrefix(blackContent, "user", false, true)
			switch {
			case isChoice(userContent, "add"):
				return a.handleAtContentMapper(a.addBlackUserByNickname, recvmsg, userContent)
			case isChoice(userContent, "del"):
				return a.handleAtContentMapper(a.deleteBlackUserByNickname, recvmsg, userContent)
			case isChoice(userContent, "show"):
				return a.showBlackUser()
			default:
				return "未知黑名单用户命令"
			}
		default:
			return "未知黑名单命令"
		}
	case content == "help":
		return helpContent()
	default:
		return "未知命令"
	}
}
