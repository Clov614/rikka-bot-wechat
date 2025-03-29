// Package biliDecoder
// @Author Clover
// @Data 2025/3/17 下午1:59:00
// @Desc
package biliDecoder

import (
	"bytes"
	"context"
	"encoding/xml"
	"regexp"
	"strconv"

	"github.com/Clov614/bilibili"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	"github.com/rs/zerolog/log"
)

type biliDecoder struct {
	cache *cache.Cache
	*plugins.Plugin
}

var baseM = matcher.Default().And().N(matcher.BaseMatcher{AllowMsgType: message.MsgTypeText | message.MsgTypeApp, Rules: matcher.IsGroupRule}) // 允许群聊 文本\转发xml消息

func init() {
	bili := &biliDecoder{
		cache:  cache.GetCache(),
		Plugin: plugins.DefaultPlugin("biliDecoder").AsLevel(plugins.MediumLevel).AsEnable(),
	}
	decoderM := baseM.And().N(matcher.Default().Or().N(matcher.NewRegexMatcher(false, `(BV[\w\d]+)`),
		matcher.NewRegexMatcher(false, `https:\/\/www\.biliDecoder\.com\/video\/(BV[\w\d]+)\/?`),
		matcher.NewRegexMatcher(false, `https:\/\/b23\.tv\/([\w\d]+)`)))
	decoderAction := plugins.DefaultActionHandler("url decoder", true).AsMatcher(decoderM)
	decoderActionFunc(decoderAction, bili) // 关键逻辑

	bili.AsAction(decoderAction)
	plugins.GetAutoRegister().RegisterPlugin(bili) // 注册插件
}

func decoderActionFunc(decoderAction *plugins.ActionHandler, bili *biliDecoder) *plugins.ActionHandler {
	return decoderAction.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		reply = *recvMsg
		switch recvMsg.Msgtype {
		case message.MsgTypeApp:
			var xmlMsg message.XMLMsg
			err = xml.Unmarshal([]byte(recvMsg.Content), &xmlMsg)
			if err != nil {
				log.Err(err).Msg("xml.Unmarshal fail at biliPlugin")
				return
			}
			// 解析链接
			if xmlMsg.AppInfo.AppName == "哔哩哔哩" {
				var videoInfo *bilibili.VideoInfo
				videoInfo, err = bilibili.NewUrlDecoder().Parse(xmlMsg.AppMsg.URL)
				if err != nil {
					log.Err(err).Msg("bilibili.NewUrlDecoder fail at biliPlugin")
					return
				}
				output := buildOutput(videoInfo)
				recvMsg.Content = output
				return
			}
		case message.MsgTypeText:
			regexBV := regexp.MustCompile(`(BV[\w\d]+)`)
			regexBilibili := regexp.MustCompile(`https:\/\/www\.bilibili\.com\/video\/(BV[\w\d]+)\/?`)
			regexShort := regexp.MustCompile(`https:\/\/b23\.tv\/([\w\d]+)`)
			var videoInfo *bilibili.VideoInfo
			urlParser := bilibili.NewUrlDecoder()
			if match := regexBV.FindStringSubmatch(recvMsg.Content); len(match) > 0 {
				videoInfo, err = urlParser.ParseByBvid(match[1])
				if err != nil {
					log.Err(err).Msg("bilibili.urlParser.ParseByBvid fail at biliPlugin")
					return
				}
			} else if match = regexBilibili.FindStringSubmatch(recvMsg.Content); len(match) > 0 {
				videoInfo, err = urlParser.Parse(match[1])
				if err != nil {
					log.Err(err).Msg("bilibili.urlParser.ParseByBvid fail at biliPlugin")
					return
				}
			} else if match = regexShort.FindStringSubmatch(recvMsg.Content); len(match) > 0 {
				videoInfo, err = urlParser.Parse(match[1])
				if err != nil {
					log.Err(err).Msg("bilibili.urlParser.ParseByBvid fail at biliPlugin")
					return
				}
			}
			output := buildOutput(videoInfo)
			sender := recvMsg.RoomId
			if sender == "" {
				sender = recvMsg.WxId
			}
			if nil != videoInfo {
				_ = bili.Cli.SendImage(sender, videoInfo.Pic)
			}
			_ = bili.Cli.SendText(sender, output)
		default:
			// nothing to do !!
		}
		return *recvMsg, false, nil
	})
}

func buildOutput(videoInfo *bilibili.VideoInfo) string {
	// 构建输出视频信息
	videoUrl := "https://www.bilibili.com/video/" + videoInfo.Bvid
	var buf bytes.Buffer
	buf.WriteString("🎬 标题:  " + videoInfo.Title + "\n")
	buf.WriteString("📂 分区:  " + videoInfo.Tname + "\n\n")
	buf.WriteString("📊 数据:\n")
	buf.WriteString("  - 👀 播放量:  " + strconv.Itoa(videoInfo.View) + "\n")
	buf.WriteString("  - 👍 点赞:  " + strconv.Itoa(videoInfo.Like) + "\n")
	buf.WriteString("  - 🪙 投币:  " + strconv.Itoa(videoInfo.Coin) + "\n")
	buf.WriteString("  - ⭐️ 收藏:  " + strconv.Itoa(videoInfo.Favorite) + "\n")
	buf.WriteString("  - 📤 分享:  " + strconv.Itoa(videoInfo.Share) + "\n\n")
	buf.WriteString("---\n")                // 分割线
	buf.WriteString("  " + videoUrl + "\n") // 链接前空两格
	buf.WriteString("---\n")                // 分割线
	return buf.String()
}
