// Package hentai 提供涩图功能
package hentai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
)

// Hentai 插件结构体
type Hentai struct {
	cache *cache.Cache
	*plugins.Plugin
}

const (
	apiURL   = "https://api.lolicon.app/setu/v2" // Lolicon API V2 地址
	proxyADR = "i.pixiv.re"
)

// --- API 请求/响应结构体 ---

// RequestBodyTag API 请求体
type RequestBodyTag struct {
	Proxy string     `json:"proxy"` // 反代
	Tag   [][]string `json:"tag"`   // 标签组，用于 AND/OR 逻辑
	Size  []string   `json:"size"`
	// R18 int    `json:"r18,omitempty"` // 0: non-r18, 1: r18, 2: mix
	// Num int    `json:"num,omitempty"` // 数量, default 1, max 20
}

// ApiResponse API 响应结构
type ApiResponse struct {
	Error string      `json:"error"` // 错误信息
	Data  []ImageData `json:"data"`  // 图片数据
}

// ImageData 单个图片数据
type ImageData struct {
	Pid        int       `json:"pid"`
	P          int       `json:"p"`
	Uid        int       `json:"uid"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	R18        bool      `json:"r18"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Tags       []string  `json:"tags"`
	Ext        string    `json:"ext"`
	AiType     int       `json:"aiType"`
	UploadDate int64     `json:"uploadDate"`
	Urls       ImageUrls `json:"urls"`
}

// ImageUrls 图片 URL
type ImageUrls struct {
	Original string `json:"original,omitempty"` // 原始图片 URL
	Regular  string `json:"regular,omitempty"`  // 原始图片 URL
}

// --- 插件初始化与注册 ---

// baseM 基础匹配器：群聊中的文本或AppMsg
var baseM = matcher.Default().And().N(matcher.BaseMatcher{AllowMsgType: message.MsgTypeText, Rules: matcher.IsGroupRule})

func init() {
	hentai := &Hentai{
		cache:  cache.GetCache(),
		Plugin: plugins.DefaultPlugin("hentai").AsLevel(plugins.HighLevel).AsEnable(),
	}

	hentai.registerHelpHandler()
	hentai.registerMainHandler()

	// 注意: 插件注册通常在主程序或插件加载逻辑中完成
	plugins.GetAutoRegister().RegisterPlugin(hentai) // 注册插件
}

// registerHelpHandler 注册 help 命令
func (ht *Hentai) registerHelpHandler() {
	helpM := matcher.PrefixMatcher{IsCut: false, IsCaseSensitive: false, Prefixes: []string{"hentai help", "车来 help", "setu help"}}
	helpAc := plugins.DefaultActionHandler("hentai help", true).AsMatcher(helpM)

	helpAc.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		reply = *recvMsg
		reply.Content = getHelpText()
		reply.Msgtype = message.MsgTypeText
		return reply, true, nil
	})
	ht.AsAction(helpAc)
}

// registerMainHandler 注册主命令 ("车来", "setu", "hentai")
func (ht *Hentai) registerMainHandler() {
	mainM := matcher.PrefixMatcher{IsCut: true, IsCaseSensitive: false, Prefixes: []string{"车来", "setu", "hentai"}}
	combinedM := baseM.And().N(mainM, matcher.DefaultNot(matcher.PrefixMatcher{IsCut: false, IsCaseSensitive: false, Prefixes: []string{"help"}}))
	mainAc := plugins.DefaultActionHandler("hentai main", true).AsMatcher(combinedM)

	mainAc.AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
		keywords := strings.Fields(strings.TrimSpace(recvMsg.Content))
		log.Printf("Hentai request received. Keywords: %v", keywords)

		// 构造 API 请求的 tag 结构
		var tags [][]string
		for _, kw := range keywords {
			tags = append(tags, []string{kw}) // 每个关键词作为独立的 AND 条件组
		}

		// 调用 API
		apiRespStr, apiErr := SendRequestWithTag(tags)
		if apiErr != nil {
			log.Printf("Error calling Lolicon API: %v", apiErr)
			reply = *recvMsg
			reply.Content = fmt.Sprintf("请求涩图接口失败：%v", apiErr)
			reply.Msgtype = message.MsgTypeText
			return reply, true, nil // 告知用户接口错误，但处理流程成功
		}

		// 解析 API 响应
		var apiResponse ApiResponse
		if err := json.Unmarshal([]byte(apiRespStr), &apiResponse); err != nil {
			log.Printf("Error parsing Lolicon API response: %v. Response: %s", err, apiRespStr)
			reply = *recvMsg
			reply.Content = "解析 API 响应失败。"
			reply.Msgtype = message.MsgTypeText
			return reply, true, nil
		}

		// 处理 API 返回的错误
		if apiResponse.Error != "" {
			log.Printf("Lolicon API returned error: %s", apiResponse.Error)
			reply = *recvMsg
			reply.Content = fmt.Sprintf("API 返回错误：%s", apiResponse.Error)
			reply.Msgtype = message.MsgTypeText
			return reply, true, nil
		}

		// 处理未找到图片的情况
		if len(apiResponse.Data) == 0 {
			log.Printf("Lolicon API returned no data for keywords: %v", keywords)
			reply = *recvMsg
			keywordStr := strings.Join(keywords, ", ")
			if keywordStr == "" {
				reply.Content = "找不到涩图 QAQ"
			} else {
				reply.Content = fmt.Sprintf("找不到与 [%s] 相关的涩图 QAQ", keywordStr)
			}
			reply.Msgtype = message.MsgTypeText
			return reply, true, nil
		}

		// 成功获取图片信息
		imageData := apiResponse.Data[0]
		imageUrl := imageData.Urls.Regular // 常规大小 （原图太大了）
		recvMsg.Content = imageData.Title
		log.Printf("Found image: PID=%d, Title=%s, URL=%s", imageData.Pid, imageData.Title, imageUrl)

		// 发送图片
		sender := recvMsg.RoomId
		if sender == "" {
			sender = recvMsg.WxId
		}
		err = ht.Cli.SendImage(sender, imageUrl)
		if err != nil {
			log.Printf("Error sending image to %s: %v", sender, err)
			// 尝试回复错误文本消息
			reply = *recvMsg
			reply.Content = fmt.Sprintf("发送图片失败: %v", err)
			reply.Msgtype = message.MsgTypeText
			return reply, true, nil // 认为处理成功，但告知用户发送失败
		}
		reply = *recvMsg
		// 图片发送成功，不需要回复额外消息
		return reply, true, nil // ok=false 表示不需要框架发送 reply 消息
	})
	ht.AsAction(mainAc)
}

// getHelpText 生成帮助文本
func getHelpText() string {
	return `--- 涩图插件帮助 ---
命令格式: 车来 [关键词1] [关键词2] ...
或使用: setu [关键词1] [关键词2] ...
或使用: hentai [关键词1] [关键词2] ...

说明:
  - 发送命令以获取一张涩图。
  - 无关键词时随机获取。
  - 支持多个关键词，用空格分隔，为 "与" 关系搜索。
  - 例如: "车来 萝莉 白丝"

示例:
  车来
  车来 碧蓝航线
  setu 原神 刻晴
  hentai help  显示此帮助信息`
}

// --- API 请求核心函数 ---

// SendRequestWithTag 发送带 tag 参数的请求
func SendRequestWithTag(tags [][]string) (string, error) {
	payload := RequestBodyTag{
		Proxy: proxyADR,
		Tag:   tags,
		Size:  []string{"original", "regular"},
		// Num: 1, // 可按需添加其他参数
	}
	// log.Printf("Sending request to %s with payload: %+v", apiURL, payload) // 调试日志
	return sendPostRequest(apiURL, payload)
}

// sendPostRequest 通用的 POST 请求发送函数
func sendPostRequest(url string, payload interface{}) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RikkaBotWechat/1.0 (HentaiPlugin)")

	// 建议使用可配置的超时时间
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("API request failed with status code %d. Response body: %s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("请求失败, 状态码: %d", resp.StatusCode)
	}

	return string(bodyBytes), nil
}
