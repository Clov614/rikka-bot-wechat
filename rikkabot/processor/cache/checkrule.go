// Package cache
// @Author Clover
// @Data 2024/7/7 下午10:17:00
// @Desc
package cache

import (
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/common"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/msgutil"
)

// IsEnable 插件是否启用
func (c *Cache) IsEnable(pluginname string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.EnablePlugins[pluginname]
}

// IsHandle 根据处理规则校验是否执行处理 是否触发该方法
func (c *Cache) IsHandle(rules *control.ProcessRules, msg message.Message) (message.Message, bool, string) {
	var matchOrder string
	if rules == nil {
		rules = &control.ProcessRules{}
	}

	enableGroupFlag := true
	calledMeFlag := !rules.IsCallMe
	adminFlag := !rules.IsAdmin
	whiteUserFlag := !rules.CheckWhiteUser
	blackUserFlag := !rules.CheckBlackUser
	whiteGroupFlag := !rules.CheckWhiteGroup
	blackGroupFlag := !rules.CheckBlackGroup
	execOrderFlag := false
	enableMsgFlag := false
	costomTriggerFlag := false

	if rules.ExecOrder == nil || len(rules.ExecOrder) == 0 {
		execOrderFlag = true
	}

	if rules.CostomTrigger == nil {
		costomTriggerFlag = true
	}
	enableMsgFlag = c.checkEnableMsgTyp(rules.EnableMsgType, msg) // 校验是否为允许的类型
	// 自定义处理规则
	if rules.CostomTrigger != nil {
		costomTriggerFlag = rules.CostomTrigger(msg)
	}

	switch msg.Msgtype {
	case message.MsgTypeApp:
		fallthrough
	case message.MsgTypeText: // 文字的处理规则
		if msg.IsGroup {
			if rules.EnableGroup {
				enableGroupFlag = true
			} else {
				enableGroupFlag = false
			}
		}
		if rules.IsCallMe {
			if msgutil.HasPrefix(msg.Content, "@", true) { // 处理艾特
				nickname := msgutil.GetNicknameByAt(msg.Content)
				if nickname == common.GetSelf().GetNickName() {
					calledMeFlag = true
					msg.Content = msgutil.TrimPrefix(msg.Content, "@"+nickname+" ", false, true)
				}
			} else { // 处理机器人名方式
				me := c.config.Symbol + c.config.Botname
				calledMeFlag = msgutil.IsCallMe(me, msg.Content)
				msg.Content = msgutil.TrimCallMe(me, msg.Content)
			}
		}
		// 必须先等前缀判定完 判断是否符合命令
		if rules.ExecOrder != nil && len(rules.ExecOrder) > 0 {
			order := ""
			execOrderFlag, order = msgutil.IsOrder(rules.ExecOrder, msg.Content)
			msg.Content = msgutil.TrimPrefix(msg.Content, order, true, true)
			matchOrder = order
		}
		if rules.IsAdmin {
			adminFlag = c.checkAdmin(msg)
		}
		if rules.CheckWhiteUser {
			whiteUserFlag = c.checkWhiteUser(msg)
		}
		if rules.CheckBlackUser {
			blackUserFlag = c.checkBlackUser(msg)
		}
		if rules.CheckWhiteGroup {
			whiteGroupFlag = c.checkWhiteGroup(msg)
		}
		if rules.CheckBlackGroup {
			blackGroupFlag = c.checkBlackGroup(msg)
		}

	case message.MsgTypeImage:
		// 暂不处理
	default:
		logging.Warn("unhandled default case, unsupported message type for now")

	}

	firstFlags := calledMeFlag && adminFlag && whiteUserFlag && whiteGroupFlag && blackUserFlag && blackGroupFlag
	return msg, firstFlags && enableMsgFlag && costomTriggerFlag && execOrderFlag && enableGroupFlag, matchOrder
}

// 判断消息发送者是否为管理员
func (c *Cache) checkAdmin(msg message.Message) bool {
	if msg.IsMySelf {
		return true
	}
	return c.HasAdminUserId(msg.WxId)
}

// 判断消息发送者是否在白名单中
func (c *Cache) checkWhiteUser(msg message.Message) bool {
	if msg.IsMySelf {
		return true
	}
	return c.HasWhiteUserId(msg.WxId)
}

// 判断群组消息是否存在白名单中
func (c *Cache) checkWhiteGroup(msg message.Message) bool {
	if !msg.IsGroup { // 不是群组消息直接返回
		return true
	}
	return c.HasWhiteGroupId(msg.RoomId)
}

// 判断是否不存在黑名单中
func (c *Cache) checkBlackUser(msg message.Message) bool {
	if msg.IsMySelf {
		return true
	}
	return !c.HasBlackUserId(msg.WxId)
}

// 判断是否不存在黑名单中
func (c *Cache) checkBlackGroup(msg message.Message) bool {
	if !msg.IsGroup { // 不是群组消息直接返回
		return true
	}
	return !c.HasBlackGroupId(msg.RoomId)
}

func (c *Cache) checkEnableMsgTyp(enableMsgTypes []message.MsgType, msg message.Message) bool {
	if enableMsgTypes == nil || len(enableMsgTypes) == 0 {
		return true
	}
	intEnableMsgTypes := make([]int, len(enableMsgTypes))
	for i := 0; i < len(enableMsgTypes); i++ {
		intEnableMsgTypes[i] = int(enableMsgTypes[i])
	}
	return msgutil.ContainsInt(intEnableMsgTypes, int(msg.Msgtype))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
