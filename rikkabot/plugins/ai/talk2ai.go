// Package ai
// @Author Clover
// @Data 2024/8/13 下午4:35:00
// @Desc 对话插件
package ai

import (
	aisdk "github.com/Clov614/go-ai-sdk"
	"github.com/rs/zerolog/log"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/control/dialog"
	"wechat-demo/rikkabot/processor/register"
)

type Talk2AI struct {
	OnceDialog dialog.OnceDialog
	*aisdk.Session
}

func init() {
	talk2AI := Talk2AI{OnceDialog: dialog.OnceDialog{
		Dialog: dialog.Dialog{
			PluginName: "AI实时对话",
			ProcessRules: &control.ProcessRules{EnableGroup: true, CheckBlackUser: true, CheckBlackGroup: true,
				CostomTrigger: func(rikkaMsg message.Message) bool {
					if rikkaMsg.IsGroup {
						// 群聊消息需要艾特
						return rikkaMsg.IsAtMe
					} else if rikkaMsg.IsFriend {
						// 好友消息，直接回复
						return true
					}
					return false
				}},
		},
	},
		Session: aisdk.DefaultSession,
	}
	talk2AI.OnceDialog.Once = func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		if recvmsg.IsGroup { // 群组消息 sessionid 为 groupid
			answer, err := talk2AI.Session.TalkById(recvmsg.GroupId, recvmsg.Content)
			if err != nil {
				log.Error().Err(err).Msg("talk2AI.Session.TalkById")
				return
			}
			talk2AI.OnceDialog.SendText(recvmsg.MetaData, answer)
			return
		}
		answer, err := talk2AI.Session.TalkById(recvmsg.SenderId, recvmsg.Content)
		if err != nil {
			log.Error().Err(err).Msg("talk2AI.Session.TalkById")
			return
		}
		talk2AI.OnceDialog.SendText(recvmsg.MetaData, answer)
	}
	register.RegistPlugin("talk2ai", &talk2AI.OnceDialog, 5)
}
