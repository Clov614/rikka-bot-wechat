// Package anime
// @Author Clover
// @Data 2025/3/31 下午7:58:00
// @Desc 动漫相关插件
package anime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
	wcf "github.com/Clov614/wcf-rpc-sdk"
)

type Anime struct {
	cache *cache.Cache
	*plugins.Plugin
}

var baseTextM = matcher.Default().And().N(matcher.BaseMatcher{AllowMsgType: message.MsgTypeText, Rules: matcher.IsGroupRule}) // 允许群聊 文本
var baseImgM = matcher.Default().And().N(matcher.BaseMatcher{AllowMsgType: message.MsgTypeImage, Rules: matcher.IsGroupRule}) // 允许群聊 图片

func init() {
	cfg := config.GetConfig()
	var apiKey string
	pluginCfg, b := cfg.GetCustomPluginCfg("saucenao_apikey")
	if !b {
		logging.Warn("saucenao apikey尚未设置，请在配置文件中配置该key")
		cfg.SetCustomPluginCfg("saucenao_apikey", apiKey)
		cfg.Update()
	} else {
		bytes, _ := json.Marshal(pluginCfg)
		json.Unmarshal(bytes, &apiKey)
	}
	anime := &Anime{
		cache:  cache.GetCache(),
		Plugin: plugins.DefaultPlugin("anime").AsLevel(plugins.MediumLevel).AsEnable(),
	}
	traceMoeImgAH := plugins.DefaultActionHandler("traceMoeImg", true).AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		var sendMsg = *recvMsg
		var id = recvMsg.WxId
		if recvMsg.IsGroup {
			id = recvMsg.RoomId
		}
		anime.Cli.SendText(id, "@识别中...", recvMsg.WxId)
		res, err := SearchAnimeByBytes(recvMsg.MetaData.GetImgData(), "image/"+recvMsg.FileInfo.FileExt)
		if err != nil {
			sendMsg.Msgtype = message.MsgTypeText // 确保发送消息以文本回复
			sendMsg.Content = err.Error()
			return sendMsg, true, nil
		}
		if res != nil && len(res.Result) != 0 {
			r := res.Result[0]

			anime.Cli.SendCardMessage(id, wcf.CardMessage{ // 发送卡片消息
				Title:    r.Filename,
				ThumbURL: r.Image,
				URL:      r.Video,
				Digest:   fmt.Sprintf("集: %v 相似度: %.2f \n开始: %.2f 结束: %.2f", r.Episode, r.Similarity, r.From, r.To),
			})
		}
		return sendMsg, false, nil
	}).AsMatcher(baseImgM)

	traceMoeM := baseTextM.And().N(matcher.NewPrefixMatcher(false, false, "以图搜番", "anime search", "tracemoe", "trace moe", "trace_moe", "trace-moe"))
	traceMoeHandler := plugins.DefaultActionHandler("traceMoeText", true).AsMatcher(traceMoeM)
	traceMoeHandler.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		var sendMsg = *recvMsg
		conn := plugins.NewAHConn(ctx, "traceMoeImg", 30*time.Second).AddAC(traceMoeImgAH)
		anime.ACPool.AddNewConn(recvMsg, conn)
		sendMsg.Content = "请在30s内发送图片"
		return sendMsg, true, nil
	})

	sauceNaoImgAH := plugins.DefaultActionHandler("saucenaoImg", true).AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		var sendMsg = *recvMsg
		var id = recvMsg.WxId
		if recvMsg.IsGroup {
			id = recvMsg.RoomId
		}
		anime.Cli.SendText(id, "@识别中...", recvMsg.WxId)
		res, err := SearchSauceNaoByBytes(apiKey, recvMsg.MetaData.GetImgData(), "image/"+recvMsg.FileInfo.FileExt, nil)
		if err != nil {
			sendMsg.Msgtype = message.MsgTypeText // 确保发送消息以文本回复
			sendMsg.Content = err.Error()
			return sendMsg, true, nil
		}
		if res != nil && len(res.Results) != 0 {
			r := res.Results[0]
			for _, newR := range res.Results {
				if newR.Header.Similarity > newR.Header.Similarity { // 选取最大相似度 item
					r = newR
				}
			}
			var url string
			if len(r.Data.ExtURLs) > 0 {
				url = r.Data.ExtURLs[0]
			}
			if strings.HasPrefix(url, "https://www.pixiv.net/") {
				url = strings.Replace(r.Data.ExtURLs[0], "https://www.pixiv.net/", "https://i.pixiv.cat/", 1)
			}
			var title = r.Data.Title
			if r.Data.Title == "" {
				title = r.Data.Source
			}
			var account = fmt.Sprintf("%v", r.Data.MemberID)
			if account == "" {
				account = strconv.Itoa(r.Data.AnidbAid)
			}
			anime.Cli.SendCardMessage(id, wcf.CardMessage{ // 发送卡片消息
				Title:    title,
				Account:  account,
				ThumbURL: r.Header.Thumbnail,
				URL:      url,
				Digest:   fmt.Sprintf("相似度: %v EstTime: %s pixivId: %v", r.Header.Similarity, r.Data.EstTime, r.Data.PixivID),
			})
		}
		return sendMsg, false, nil
	}).AsMatcher(baseImgM)

	saucenaoM := baseTextM.And().N(matcher.NewPrefixMatcher(false, false, "saucenao", "搜图", "pixiv_search", "pixiv search", "search image"))
	saucenaoHandler := plugins.DefaultActionHandler("saucenaoText", true).AsMatcher(saucenaoM)
	saucenaoHandler.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		var sendMsg = *recvMsg
		conn := plugins.NewAHConn(ctx, "saucenaoImg", 30*time.Second).AddAC(sauceNaoImgAH)
		anime.ACPool.AddNewConn(recvMsg, conn)
		sendMsg.Content = "请在30s内发送图片"
		return sendMsg, true, nil
	})

	anime.AsAction(traceMoeHandler) // 注册 traceMoe
	anime.AsAction(saucenaoHandler) // 注册 saucenao

	// --- 添加帮助功能 ---
	helpM := baseTextM.And().N(matcher.NewPrefixMatcher(false, false, "搜番 help", "anime help"))
	helpHandler := plugins.DefaultActionHandler("animeHelp", true).AsMatcher(helpM)
	helpHandler.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		reply = *recvMsg
		reply.Content = anime.help()
		return reply, true, nil
	})
	anime.AsAction(helpHandler) // 注册 help
	// --- 结束帮助功能添加 ---

	plugins.GetAutoRegister().RegisterPlugin(anime)
}

// help 生成帮助信息
func (a *Anime) help() string {
	var buf bytes.Buffer
	buf.WriteString("动漫插件使用帮助:\n")
	buf.WriteString("---------------\n")
	buf.WriteString("1. 以图搜番 (Trace Moe):\n")
	buf.WriteString("   - 发送 \"以图搜番\" 或 \"tracemoe\"\n")
	buf.WriteString("   - 然后在30秒内发送需要识别的动漫截图。\n")
	buf.WriteString("   - 功能: 根据图片查找对应的动漫及出处。\n")
	buf.WriteString("---------------\n")
	buf.WriteString("2. 搜图 (SauceNAO):\n")
	buf.WriteString("   - 发送 \"搜图\" 或 \"saucenao\"\n")
	buf.WriteString("   - 然后在30秒内发送需要识别的图片。\n")
	buf.WriteString("   - 功能: 查找图片的来源，支持P站等。\n")
	buf.WriteString("---------------\n")
	buf.WriteString("发送 \"搜番 help\" 查看此帮助。")
	return buf.String()
}
