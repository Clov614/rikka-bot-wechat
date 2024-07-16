// @Author Clover
// @Data 2024/7/15 下午4:59:00
// @Desc 管理员默认方法
package plugin_admin

import (
	"bytes"
	"fmt"
	"wechat-demo/rikkabot/common"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/cache"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/register"
	"wechat-demo/rikkabot/utils/msgutil"
)

func init() {
	registAdminPlugin() // 管理管基础功能
}

const (
	defaultAdminControl = "admin_control"
)

// 注册管理员基础功能
func registAdminPlugin() {
	adminPlugin := AdminPlugin{
		onceDialog: &control.OnceDialog{},
		user:       common.GetSelf(),
	}
	onceDialog := adminPlugin.onceDialog
	onceDialog.PluginName = "管理员基础功能"
	onceDialog.ProcessRules = &control.ProcessRules{IsCallMe: true, IsAdmin: true, IsAtMe: true, EnableGroup: true,
		ExecOrder: []string{"admin"}}

	onceDialog.Once = func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		if adminPlugin.cache == nil {
			adminPlugin.cache = cache.GetCache() // 运行时获取缓存指针
		}
		content := recvmsg.Content
		content = msgutil.TrimPrefix(content, "admin", false, true)
		reply := ""
		switch true {
		case isChoice(content, "add"):
			nickname := msgutil.TrimPrefix(content, "add", false, true)
			if nickname == "" {
				reply = "添加管理员示例: add admin <nickname>"
				break
			}
			if !recvmsg.IsMySelf {
				reply = "仅有超级管理员(自己)，才能操作"
				break
			}
			reply = adminPlugin.addAdmin(nickname)
		case isChoice(content, "del"):
			nickname := msgutil.TrimPrefix(content, "del", false, true)
			if nickname == "" {
				reply = "移除管理员示例: del admin <nickname>"
				break
			}
			if !recvmsg.IsMySelf {
				reply = "仅有超级管理员(自己)，才能操作"
				break
			}
			reply = adminPlugin.deleteAdmin(nickname)
		case isChoice(content, "show admin"):
			if !recvmsg.IsMySelf {
				reply = "仅有超级管理员(自己)，才能操作"
				break
			}
			reply = adminPlugin.showAdminList()
		case isChoice(content, "show plugin state") || isChoice(content, "show plugin status"):
			reply = adminPlugin.showPluginState()
		case content == "show":
			reply = "show what? <admin> or <plugin state>\nExample: <callbot> admin show admin"
		// 插件管理
		case isChoice(content, "plugin"):
			pluginsContent := msgutil.TrimPrefix(content, "plugin", false, true)
			switch true {
			case isChoice(pluginsContent, "enable"):
				pluginsContent = msgutil.TrimPrefix(pluginsContent, "enable", false, true)
				if pluginsContent == "" {
					reply = "enable后面请跟模块名称\n示例: admin plugin enable <plugin name> \n(如不清楚模块名称请调用: admin show plugin state)"
					break
				}
				reply = adminPlugin.enablePlugin(pluginsContent)
			case isChoice(pluginsContent, "disable"):
				pluginsContent = msgutil.TrimPrefix(pluginsContent, "disable", false, true)
				if pluginsContent == "" {
					reply = "disable后面请跟模块名称\n示例: admin plugin disable <plugin name> \n(如不清楚模块名称请调用: admin show plugin state)"
					break
				}
				reply = adminPlugin.disablePlugin(pluginsContent)
			}
		// 白名单
		case isChoice(content, "white"):
			whiteContent := msgutil.TrimPrefix(content, "white", false, true)
			switch true {
			case isChoice(whiteContent, "add"): // 添加群组白名单
				whiteContent = msgutil.TrimPrefix(whiteContent, "add", false, true)
				if whiteContent == "" {
					if !recvmsg.IsGroup {
						reply = "添加失败，群组内才能操作: admin white add"
						break
					}
					reply = adminPlugin.addWhiteGroupByMsg(recvmsg)
				} else {
					reply = adminPlugin.addWhiteGroup(whiteContent)
				}
			case isChoice(whiteContent, "del"):
				whiteContent = msgutil.TrimPrefix(whiteContent, "del", false, true)
				if whiteContent == "" {
					if !recvmsg.IsGroup {
						reply = "移除白名单失败，群组内才能操作: admin white del"
						break
					}
					reply = adminPlugin.deleteWhiteGroupByMsg(recvmsg)
				} else {
					reply = adminPlugin.deleteWhiteGroup(whiteContent)
				}
			case isChoice(whiteContent, "show"):
				reply = adminPlugin.showWhiteGroup()
			}
		// 黑名单
		case isChoice(content, "black"):
			blackcontent := msgutil.TrimPrefix(content, "black", false, true)
			switch true {
			case isChoice(blackcontent, "group"): // 群组黑名单
				groupContent := msgutil.TrimPrefix(blackcontent, "group", false, true)
				switch true {
				case isChoice(groupContent, "add"): // 添加群组白名单
					groupContent = msgutil.TrimPrefix(groupContent, "add", false, true)
					if groupContent == "" {
						if !recvmsg.IsGroup {
							reply = "添加群组黑名单失败，群组内才能操作: admin black group add"
							break
						}
						reply = adminPlugin.addBlackGroupByMsg(recvmsg)
					} else { // 否则就是按照参数添加黑名单
						reply = adminPlugin.addBlackGroupByNickname(groupContent)
					}
				case isChoice(groupContent, "del"):
					groupContent = msgutil.TrimPrefix(groupContent, "del", false, true)
					if groupContent == "" {
						if !recvmsg.IsGroup {
							reply = "移除群组黑名单失败，群组内才能操作: admin black group del\n可选使用: admin black group del <group nickname>"
							break
						}
						reply = adminPlugin.deleteBlackGroupByMsg(recvmsg)
					} else {
						reply = adminPlugin.deleteBlackGroupByNickname(groupContent)
					}
				case isChoice(groupContent, "show"):
					reply = adminPlugin.showBlackGroup()
				}
			case isChoice(blackcontent, "user"):
				userContent := msgutil.TrimPrefix(blackcontent, "user", false, true)
				switch true {
				case isChoice(userContent, "add"):
					userContent = msgutil.TrimPrefix(userContent, "add", false, true)
					reply = adminPlugin.addBlackUserByNickname(recvmsg, userContent)
				case isChoice(userContent, "del"):
					userContent = msgutil.TrimPrefix(userContent, "del", false, true)
					reply = adminPlugin.deleteBlackUserByNickname(recvmsg, userContent)
				case isChoice(userContent, "show"):
					reply = adminPlugin.showBlackUser()
				}
			}
		// help
		case content == "help":
			reply = helpContent()
		}

		onceDialog.SendText(recvmsg.MetaData, reply) // send msg
	}

	register.RegisterPlugin(defaultAdminControl, adminPlugin.onceDialog)
}

func isChoice(cotent string, prefix string) bool {
	return msgutil.HasPrefix(cotent, prefix, true)
}

// 管理员模块
type AdminPlugin struct {
	onceDialog *control.OnceDialog
	cache      *cache.Cache
	user       *common.Self
}

//region admin 管理员

// 根据nickname添加管理员
func (a AdminPlugin) addAdmin(nickname string) (reply string) {
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
		return fmt.Sprintf("不是群消息，无法添加群聊白名单")
	}
	a.cache.AddWhiteGroupId(msg.GroupId)

	return fmt.Sprintf("添加成功，群组( %s )id( %s )进入白名单", msg.MetaData.GetGroupNickname(), msg.GroupId)
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
	a.cache.DeleteWhiteGroupId(msg.GroupId)
	return fmt.Sprintf("移除了，群组( %s )id( %s )的白名单", msg.MetaData.GetGroupNickname(), msg.GroupId)
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
	a.cache.AddBlackGroupId(msg.GroupId)
	return fmt.Sprintf("添加了，群组( %s )id( %s )的黑名单", msg.MetaData.GetGroupNickname(), msg.GroupId)
}

// 移除该群组黑名单,群组内使用（不携带昵称）
func (a AdminPlugin) deleteBlackGroupByMsg(msg message.Message) (reply string) {
	if !msg.IsGroup {
		return "不是群聊无法移除群组黑名单(请携带群名参数)"
	}
	a.cache.DeleteBlackGroupId(msg.GroupId)
	return fmt.Sprintf("移除了，群组( %s )id( %s )的黑名单", msg.MetaData.GetGroupNickname(), msg.GroupId)
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
		friendId, err = msg.MetaData.GetGroupMemberIdByNickname(nickname)
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
		friendId, err = msg.MetaData.GetGroupMemberIdByNickname(nickname)
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
