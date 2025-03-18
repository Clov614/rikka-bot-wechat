// Package ai
// @Author Clover
// @Data 2024/8/13 下午4:35:00
// @Desc 对话插件
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	aisdk "github.com/Clov614/go-ai-sdk"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/rs/zerolog/log"
)

type AiCfg struct {
	SystemSet      string `comment:"预设语句" yaml:"system_set" json:"system_set"`
	SessionTimeOut int    `comment:"会话超时时间" yaml:"session_time_out" json:"session_time_out"`
}

type Talk2AI struct {
	*plugins.Plugin
	*aisdk.Session
}

var defaultAiCfg = AiCfg{"从现在开始，我需要你扮演小鸟游六花这个动漫角色，语气尽力模仿(逻辑符合正常可用逻辑进行模仿)，要完美融入这个角色的设定中。我会称呼你为rikka或六花，届时你明白是在称呼你。", 30}

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
	aiM := matcher.Default().And().N(matcher.BaseMatcher{Rules: matcher.IsGroupRule | matcher.IsAtMeRule, AllowMsgType: message.MsgTypeText}).
		Or().N(matcher.BaseMatcher{Rules: matcher.IsFriendRule, AllowMsgType: message.MsgTypeText})

	talk2AI := Talk2AI{Plugin: plugins.DefaultPlugin("aiTalk"),
		Session: aisdk.NewSession(aiCfg.SystemSet, aiCfg.SessionTimeOut),
	}
	talk2AI.AsLevel(plugins.VeryLowLevel) // 最低优先级
	actionHandler := plugins.DefaultActionHandler("talk", true).AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		answer, err := DefaultFilter.filter(recvMsg.Content, func(content string) (string, error) {
			if recvMsg.IsGroup { // 群组消息 sessionid 为 groupid
				answer, err := talk2AI.Session.TalkByIdEx(recvMsg.RoomId, recvMsg.Content,
					func() string {
						return fmt.Sprintf("(该次对话隐藏信息 uuid:%s isGroup:%v)", recvMsg.RoomId, recvMsg.IsGroup)
					})
				if err != nil {
					log.Error().Err(err).Msg("talk2AI.Session.TalkById")
					return "", fmt.Errorf("failed to talk2AI.Session.TalkById %w", err)
				}
				return answer, nil
			}
			answer, err := talk2AI.Session.TalkByIdEx(recvMsg.WxId, recvMsg.Content,
				func() string {
					return fmt.Sprintf("(该次对话隐藏信息 uuid:%s isGroup:%v)", recvMsg.WxId, recvMsg.IsGroup)
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

		recver := recvMsg.RoomId
		if recver == "" {
			recver = recvMsg.WxId
		}
		_ = talk2AI.Cli.SendText(recver, answer) // 回复消息
		return reply, false, nil                 // 向外传递不回复消息
	})
	actionHandler.AsMatcher(aiM) // 绑定消息匹配器
	talk2AI.AsEnable().AsAction(actionHandler)

	ag := plugins.GetAutoRegister()
	ag.RegisterPlugin(talk2AI) // 注册模块
}
