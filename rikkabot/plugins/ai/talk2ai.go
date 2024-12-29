// Package ai
// @Author Clover
// @Data 2024/8/13 下午4:35:00
// @Desc 对话插件
package ai

import (
	"encoding/json"
	"fmt"
	aisdk "github.com/Clov614/go-ai-sdk"
	"github.com/rs/zerolog/log"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/message"
	_ "wechat-demo/rikkabot/plugins/ai/cron"
	_ "wechat-demo/rikkabot/plugins/ai/cron/cronplugins"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/control/dialog"
	"wechat-demo/rikkabot/processor/register"
)

type AiCfg struct {
	SystemSet      string `comment:"预设语句" yaml:"system_set" json:"system_set"`
	SessionTimeOut int    `comment:"会话超时时间" yaml:"session_time_out" json:"session_time_out"`
}

type Talk2AI struct {
	OnceDialog *dialog.OnceDialog
	*aisdk.Session
}

var defaultAiCfg = AiCfg{"从现在开始，我需要你扮演小鸟游六花这个动漫角色，语气跟说话逻辑都要尽力模仿，要完美融入这个角色的设定中。我会称呼你为rikka或六花，届时你明白是在称呼你。", 30}

func init() {
	cfg := config.GetConfig()
	var aiCfg AiCfg
	iaiCfg, ok := cfg.GetCustomPluginCfg("ai")
	if !ok {
		aiCfg = defaultAiCfg
		cfg.SetCustomPluginCfg("ai", aiCfg)
		_ = cfg.Update()
		log.Warn().Msg("AI模块设置尚未初始化，初始化为默认设置")
	} else {
		bytes, err := json.Marshal(iaiCfg)
		if err != nil {
			log.Err(err).Msg("解析ai设置信息错误1")
			aiCfg = defaultAiCfg
		}
		if err == nil {
			err = json.Unmarshal(bytes, &aiCfg)
			if err != nil {
				log.Err(err).Msg("解析ai设置信息错误2")
				aiCfg = defaultAiCfg
			}
		}
	}
	// 消息规则
	rules := &control.ProcessRules{EnableGroup: true, CheckBlackUser: true, CheckBlackGroup: true,
		CostomTrigger: func(rikkaMsg message.Message) bool {
			if rikkaMsg.Msgtype != message.MsgTypeText || !(rikkaMsg.IsFriend || rikkaMsg.IsGroup) {
				return false
			}
			if rikkaMsg.IsGroup {
				// 群聊消息需要艾特
				return rikkaMsg.IsAtMe
			} else if rikkaMsg.IsFriend {
				// 好友消息，直接回复
				return true
			}
			return false
		}}

	talk2AI := Talk2AI{OnceDialog: dialog.InitOnceDialog("AI实时对话", rules, message.MsgTypeList{message.MsgTypeText}),
		Session: aisdk.NewSession(aiCfg.SystemSet, aiCfg.SessionTimeOut),
	}
	talk2AI.OnceDialog.Once = func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		answer, err := DefaultFilter.filter(recvmsg.Content, func(content string) (string, error) {
			if recvmsg.IsGroup { // 群组消息 sessionid 为 groupid
				answer, err := talk2AI.Session.TalkByIdEx(recvmsg.GroupId, recvmsg.Content,
					func() string {
						return fmt.Sprintf("(该次对话隐藏信息 uuid:%s isGroup:%v)", recvmsg.Uuid, recvmsg.IsGroup)
					})
				if err != nil {
					log.Error().Err(err).Msg("talk2AI.Session.TalkById")
					return "", fmt.Errorf("failed to talk2AI.Session.TalkById %w", err)
				}
				return answer, nil
			}
			answer, err := talk2AI.Session.TalkByIdEx(recvmsg.SenderId, recvmsg.Content,
				func() string {
					return fmt.Sprintf("(该次对话隐藏信息 uuid:%s isGroup:%v)", recvmsg.Uuid, recvmsg.IsGroup)
				})
			if err != nil {
				log.Error().Err(err).Msg("talk2AI.Session.TalkById")
				return "", fmt.Errorf("failed to talk2AI.Session.TalkById %w", err)
			}
			return answer, nil
		})
		if err != nil {
			log.Error().Err(err).Msg("talk2AI.Session.TalkById")
			return
		}
		talk2AI.OnceDialog.SendText(recvmsg.MetaData, answer) // 回复消息
	}
	register.RegistPlugin("talk2ai", talk2AI.OnceDialog, 5)
}
