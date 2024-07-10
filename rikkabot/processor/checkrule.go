// @Author Clover
// @Data 2024/7/7 下午10:17:00
// @Desc
package processor

import (
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/utils/msgutil"
)

// 插件是否启用
func (c *Cache) isEnable(pluginname string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.EnablePlugins[pluginname]
}

// 根据处理规则校验是否执行处理 是否触发该方法
func (c *Cache) isHandle(rules *control.ProcessRules, msg *message.Message) bool {
	context := msg.RawContext

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
	atMeFlag := true // todo 待完善 （尚不明确如何判断艾特自己）
	enableMsgFlag := false
	costomTriggerFlag := false

	if rules.ExecOrder == "" {
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
	case message.MsgTypeText: // 文字的处理规则
		if msg.IsGroup {
			if rules.EnableGroup {
				enableGroupFlag = true
			} else {
				enableGroupFlag = false
			}
		}
		if rules.IsCallMe {
			calledMeFlag = msgutil.IsCallMe(context)
			msg.RawContext = msgutil.TrimCallMe(context)
		}
		// 必须先等前缀判定完 判断是否符合命令
		if rules.ExecOrder != "" {
			execOrderFlag = msgutil.IsOrder(msg.RawContext, rules.ExecOrder)
			msg.RawContext = msgutil.TrimPrefix(msg.RawContext, rules.ExecOrder, true, true)
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

	}

	firstFlags := calledMeFlag && adminFlag && whiteUserFlag && whiteGroupFlag && blackUserFlag && blackGroupFlag
	return firstFlags && atMeFlag && enableMsgFlag && costomTriggerFlag && execOrderFlag && enableGroupFlag
}

// 判断消息发送者是否为管理员
func (c *Cache) checkAdmin(msg *message.Message) bool {
	if msg.IsMySelf {
		return true
	}
	return c.HasAdminUserId(msg.SenderId)
}

// 判断消息发送者是否在白名单中
func (c *Cache) checkWhiteUser(msg *message.Message) bool {
	if msg.IsMySelf {
		return true
	}
	return c.HasWhiteUserId(msg.SenderId)
}

// 判断群组消息是否存在白名单中
func (c *Cache) checkWhiteGroup(msg *message.Message) bool {
	if !msg.IsGroup { // 不是群组消息直接返回
		return true
	}
	return c.HasWhiteGroupId(msg.GroupId)
}

// 判断是否不存在黑名单中
func (c *Cache) checkBlackUser(msg *message.Message) bool {
	if msg.IsMySelf {
		return true
	}
	return !c.HasBlackUserId(msg.SenderId)
}

// 判断是否不存在黑名单中
func (c *Cache) checkBlackGroup(msg *message.Message) bool {
	if !msg.IsGroup { // 不是群组消息直接返回
		return true
	}
	return !c.HasBlackGroupId(msg.GroupId)
}

func (c *Cache) checkEnableMsgTyp(enableMsgTypes []message.MsgType, msg *message.Message) bool {
	if enableMsgTypes == nil || len(enableMsgTypes) == 0 {
		return true
	}
	intEnableMsgTypes := make([]int, len(enableMsgTypes))
	for i := 0; i < len(enableMsgTypes); i++ {
		intEnableMsgTypes[i] = int(enableMsgTypes[i])
	}
	return msgutil.ContainsInt(intEnableMsgTypes, int(msg.Msgtype))
}
