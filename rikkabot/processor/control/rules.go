// @Author Clover
// @Data 2024/7/8 下午8:35:00
// @Desc
package control

import "wechat-demo/rikkabot/message"

// 处理规则
type ProcessRules struct {
	CheckWhiteUser  bool // 启用 用户白名单 校验
	CheckBlackUser  bool // 启用 用户黑名单 校验
	CheckWhiteGroup bool // 启用 群组白名单校验
	CheckBlackGroup bool // 启用 群组黑名单校验

	IsAtMe        bool                                 // 是否需要艾特
	IsCallMe      bool                                 // 是否呼唤机器人 （满足 symbol + botname）
	IsAdmin       bool                                 // 是否只有管理员能操作
	EnableGroup   bool                                 // 是否处理群消息
	ExecOrder     string                               // 匹配的指定
	CostomTrigger func(rikkaMsg *message.Message) bool // 自定义触发器 （是否触发）
	EnableMsgType []message.MsgType                    // 允许的消息类型 （nil 或空默认 全允许）
}
