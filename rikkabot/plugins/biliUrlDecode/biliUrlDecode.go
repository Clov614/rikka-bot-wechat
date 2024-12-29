package biliUrlDecode

import (
	"bytes"
	"encoding/xml"
	"github.com/Clov614/bilibili"
	"github.com/rs/zerolog/log"
	"regexp"
	"strconv"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/control/dialog"
	"wechat-demo/rikkabot/processor/register"
	"wechat-demo/rikkabot/utils/imgutil"
)

func init() {
	// 允许群组 白名单允许
	rules := &control.ProcessRules{EnableGroup: true, CostomTrigger: func(rikkaMsg message.Message) bool {
		if rikkaMsg.Msgtype == message.MsgTypeApp {
			var xmlMsg message.XMLMsg
			err := xml.Unmarshal([]byte(rikkaMsg.Content), &xmlMsg)
			if err != nil {
				log.Err(err).Msg("xml.Unmarshal fail at biliPlugin")
				return false
			}
			// 解析链接
			if xmlMsg.AppInfo.AppName == "哔哩哔哩" {
				return true
			}
		} else if rikkaMsg.Msgtype == message.MsgTypeText {
			regexBV := regexp.MustCompile(`(BV[\w\d]+)`)
			regexBilibili := regexp.MustCompile(`https:\/\/www\.bilibili\.com\/video\/(BV[\w\d]+)\/?`)
			regexShort := regexp.MustCompile(`https:\/\/b23\.tv\/([\w\d]+)`)
			return regexBV.MatchString(rikkaMsg.Content) || regexBilibili.MatchString(rikkaMsg.Content) || regexShort.MatchString(rikkaMsg.Content)
		}
		return false
	}}
	biliDecodePlugin := biliPlugin{
		// 设置插件名&消息规则&需要的消息类型
		OnceDialog: dialog.InitOnceDialog("bilibili链接解析", rules, message.MsgTypeList{message.MsgTypeApp}),
	}
	// 运行时逻辑
	biliDecodePlugin.SetOnceFunc(func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		switch recvmsg.Msgtype {
		case message.MsgTypeApp:
			var xmlMsg message.XMLMsg
			err := xml.Unmarshal([]byte(recvmsg.Content), &xmlMsg)
			if err != nil {
				log.Err(err).Msg("xml.Unmarshal fail at biliPlugin")
				return
			}
			// 解析链接
			if xmlMsg.AppInfo.AppName == "哔哩哔哩" {
				videoInfo, err := bilibili.NewUrlDecoder().Parse(xmlMsg.AppMsg.URL)
				if err != nil {
					log.Err(err).Msg("bilibili.NewUrlDecoder fail at biliPlugin")
				}
				output := buildOutput(videoInfo)
				biliDecodePlugin.SendText(recvmsg.MetaData, output)
				return
			}
		case message.MsgTypeText:
			regexBV := regexp.MustCompile(`(BV[\w\d]+)`)
			regexBilibili := regexp.MustCompile(`https:\/\/www\.bilibili\.com\/video\/(BV[\w\d]+)\/?`)
			regexShort := regexp.MustCompile(`https:\/\/b23\.tv\/([\w\d]+)`)
			var videoInfo *bilibili.VideoInfo
			var err error
			urlParser := bilibili.NewUrlDecoder()
			if match := regexBV.FindStringSubmatch(recvmsg.Content); len(match) > 0 {
				videoInfo, err = urlParser.ParseByBvid(match[1])
				if err != nil {
					log.Err(err).Msg("bilibili.urlParser.ParseByBvid fail at biliPlugin")
					return
				}
			} else if match = regexBilibili.FindStringSubmatch(recvmsg.Content); len(match) > 0 {
				videoInfo, err = urlParser.Parse(match[1])
				if err != nil {
					log.Err(err).Msg("bilibili.urlParser.ParseByBvid fail at biliPlugin")
					return
				}
			} else if match = regexShort.FindStringSubmatch(recvmsg.Content); len(match) > 0 {
				videoInfo, err = urlParser.Parse(match[1])
				if err != nil {
					log.Err(err).Msg("bilibili.urlParser.ParseByBvid fail at biliPlugin")
					return
				}
			}
			output := buildOutput(videoInfo)
			imgFetch, err := imgutil.ImgFetch(videoInfo.Pic)
			if err != nil {
				log.Err(err).Msg("bilibili.videoInfo.pic.fetchimg fail at biliPlugin")
			} else {
				biliDecodePlugin.SendImage(recvmsg.MetaData, imgFetch) // 发送图片封面
			}
			biliDecodePlugin.SendText(recvmsg.MetaData, output)
		}
	})
	register.RegistPlugin("bili-url-parse", biliDecodePlugin.OnceDialog, 1)
}

func buildOutput(videoInfo *bilibili.VideoInfo) string {
	// 构建输出视频信息
	videoUrl := "https://www.bilibili.com/video/" + videoInfo.Bvid + "\n"
	var buf bytes.Buffer
	buf.WriteString(videoUrl)
	buf.WriteString("标题:  " + videoInfo.Title + "\n")
	buf.WriteString("分区:  " + videoInfo.Tname + "\n")
	buf.WriteString("播放量:  " + strconv.Itoa(videoInfo.View) + "\n")
	buf.WriteString("点赞:  " + strconv.Itoa(videoInfo.Like) + "\n")
	buf.WriteString("投币:  " + strconv.Itoa(videoInfo.Coin) + "\n")
	buf.WriteString("收藏:  " + strconv.Itoa(videoInfo.Favorite) + "\n")
	buf.WriteString("分享:  " + strconv.Itoa(videoInfo.Share) + "\n")
	buf.WriteString("Bvid:  \n\n     " + videoInfo.Bvid + "\n")
	return buf.String()
}

type biliPlugin struct {
	*dialog.OnceDialog
}
