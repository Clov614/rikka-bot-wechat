// Package httpapi
// @Author Clover
// @Data 2024/7/20 下午9:37:00
// @Desc http and http webhook
package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/onebot/dto/event"
	"wechat-demo/rikkabot/onebot/oneboterr"
	"wechat-demo/rikkabot/utils/timeutil"
)

// HttpServer http 服务
type HttpServer struct {
	HttpAddr    string
	AccessToken string // 鉴权
	bot         *rikkabot.RikkaBot
}

const (
	failedStatus  = "failed"
	successStatus = "ok"
)

// Run HttpServer
func (s HttpServer) Run() {

	r := gin.Default()

	// 全局中间件
	r.Use(func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		log.Info().
			Dur("latency", duration).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Msg("Request details")
	})
	r.Use(gin.Recovery())

	// 处理
	HttpApiGroup := r.Group("/")
	{
		HttpApiGroup.GET("/*filepath", s.globalHandler())
		HttpApiGroup.POST("/*filepath", s.globalHandler())
	}

	// 启动
	parsedURL, err := url.Parse(s.HttpAddr)
	if err != nil {
		logging.Fatal("启动正向http致命错误!请检查地址是否正确", 4, map[string]interface{}{"err": err})
	}
	go func() {
		err = r.Run(parsedURL.Host)
		if err != nil {
			logging.Fatal("启动正向http致命错误!", 4, map[string]interface{}{"err": err})
		}
	}()
	logging.Info(fmt.Sprintf("正向http启动成功,监听: %s 端口", s.HttpAddr))
}

func (s HttpServer) globalHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := s.AccessToken
		if accessToken != "" {
			tokenHeader := strings.Replace(c.GetHeader("Authorization"), "Bearer ", "", 1)
			tokenQuery, _ := c.GetQuery("access_token")
			if (tokenHeader == "" || tokenHeader != accessToken) && (tokenQuery == "" || tokenQuery != accessToken) {
				c.JSON(http.StatusForbidden, gin.H{"error": "鉴权失败"})
				return
			}
		}

		logging.Debug("鉴权成功", map[string]interface{}{"path": c.Request.URL.Path})
		// 检查路径和处理对应的请求
		switch c.Request.URL.Path {
		case "/send_message": // 发送消息
			s.handleSendMsg(c)
		case "/login_callback": // 获取登录回调
			s.handleLoginUrl(c)
		}

	}
}

func (s HttpServer) handleLoginUrl(c *gin.Context) {
	var req event.ActionRequest[any]
	var resp event.ActionResponse
	if c.Request.Method == http.MethodGet {
		// 从URL查询解析参数

		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	logging.Debug("请求参数", map[string]interface{}{"action_request": req})
	if req.Action != "login_callback" {
		retErr(c, "/login_callback 端点只处理 action: login_callback",
			oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}
	var retData struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	retData.Type = "url"
	retData.Data = s.bot.GetloginUrl()
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = retData
	respData, err := json.Marshal(resp)
	if err != nil {
		logging.Error("marshal response failed", map[string]interface{}{"err": err})
		retErr(c, "marshal response failed", oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		return
	}

	c.Header("Content-Type", "application/json")
	logging.Info(fmt.Sprintf("发送成功回执: %+v", string(respData)))
	c.String(http.StatusOK, string(respData)) // 返回json字符串
}

func (s HttpServer) handleSendMsg(c *gin.Context) {
	var req event.ActionRequest[event.SendMsgParams]
	var resp event.ActionResponse
	if c.Request.Method == http.MethodGet {
		// 从URL查询解析参数

		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	logging.Debug("请求参数", map[string]interface{}{"action_request": req})

	// 处理发送消息

	if req.Action != "send_message" { // 判断 action
		retErr(c, "/send_message 端点只处理 action: send_message",
			oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}
	// 校验params
	if req.Params.DetailType == "" {
		retErr(c, "params.detail_type 为空 请携带其并赋予（群组/个人） group private", oneboterr.BAD_PARAME, failedStatus)
	}
	if req.Params.Message == nil {
		retErr(c, "params.message 为空 请携带发送的消息数据(string、[]byte) 图片消息支持传入string链接",
			oneboterr.BAD_PARAME, failedStatus)
	}
	var isGroup bool
	if req.Params.DetailType == "group" {
		isGroup = true
	} else if req.Params.DetailType == "private" {
		isGroup = false
	} else {
		retErr(c, "params.detail_type 只支持 group private 两种消息类型", oneboterr.BAD_PARAME, failedStatus)
	}

	err := s.bot.SendMsg(req.Params.MsgType, isGroup, req.Params.Message, req.Params.SendId)
	if err != nil {
		logging.Error("Http server 发送消息错误", map[string]interface{}{"err": err})
		retErr(c, fmt.Sprintf("发送消息错误 err: %s", err), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		return
	}
	msgRespData := event.MsgRespData{
		Time:      timeutil.GetTimeUnix(),
		MessageId: "尚未实现返回send_msg_id，后续实现(撤回消息功能需要)", // todo
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = msgRespData

	respData, err := json.Marshal(resp)
	if err != nil {
		logging.Error("marshal response failed", map[string]interface{}{"err": err})
		retErr(c, "marshal response failed", oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		return
	}

	// 返回处理结果
	c.Header("Content-Type", "application/json")
	logging.Info(fmt.Sprintf("发送成功回执: %+v", string(respData)))
	c.String(http.StatusOK, string(respData)) // 返回json字符串
}

// 处理错误并返回 json
func retErr(c *gin.Context, errMsg string, retcode int64, status string) {
	var resp event.ActionResponse
	resp.Message = errMsg
	resp.Retcode = retcode
	resp.Status = status
	c.JSON(http.StatusOK, resp)
}

// HttpClient 反向http
type HttpClient struct {
	secret     string
	postUrl    string
	timeout    int
	client     *http.Client
	MaxRetries int
	InitDelay  time.Duration // 最初的重试间隔
	MaxDelay   time.Duration // 最大重试间隔
	bot        *rikkabot.RikkaBot
}

// RunHttp 启动 http、http上报器
func RunHttp(rbot *rikkabot.RikkaBot) {
	httpserverCfg := rbot.Config.HttpServer
	// http server
	HttpServer{
		HttpAddr:    httpserverCfg.HttpAddress,
		AccessToken: httpserverCfg.AccessToken,
		bot:         rbot,
	}.Run()

	// http上报器
	for _, post := range rbot.Config.HttpPost {
		HttpClient{
			secret:     post.Secret,
			postUrl:    post.Url,
			timeout:    post.TimeOut,
			MaxRetries: post.MaxRetries,
			InitDelay:  1500 * time.Millisecond,
			MaxDelay:   8 * time.Second,
			bot:        rbot,
		}.Run()
	}
	HandlerHeartBeat(rbot) // 处理心跳 推送心跳事件
}

func (c HttpClient) Run() {
	if c.timeout < 5 {
		c.timeout = 5
	}

	c.client = &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}
	logging.Info("Http Post 上报器已启动！" + c.postUrl)
	// 注册事件处理
	c.bot.OnEventPush(c.HandlerPostEvent)
}

// HandlerHeartBeat 心跳事件
func HandlerHeartBeat(bot *rikkabot.RikkaBot) {
	cfg := bot.Config
	if !cfg.EnableHeartBeat {
		logging.Warn("警告: 心跳功能已关闭，若非预期，请检查配置文件。")
		return
	}
	go func() {
		t := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		for {
			<-t.C
			_ = bot.EventPool.AddEvent(event.HeartBeatEvent{
				Event: event.Event{
					Id:         uuid.New().String(),
					Time:       timeutil.GetTimeUnix(),
					Type:       "meta",
					DetailType: "heart_beat",
				},
				Interval: cfg.Interval,
			})
		}
	}()
	logging.Info(fmt.Sprintf("心跳事件启动！间隔：%d", cfg.Interval))
}

// HandlerPostEvent 处理 post 事件
func (c HttpClient) HandlerPostEvent(event event.IEvent) {
	var err error
	// todo 失败的请求根据 MaxRetries 重试
	eventJSON, err := json.Marshal(event)
	if err != nil {
		logging.Error("marshal event failed", map[string]interface{}{"err": err})
	}
	var req *http.Request
	var resp *http.Response
	for i := 0; i <= c.MaxRetries; i++ {
		req, err = http.NewRequest("POST", c.postUrl, bytes.NewBuffer(eventJSON))
		if err != nil {
			logHttpPostError(event, err, "request create failed")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+encrypt(c.secret))

		resp, err = c.client.Do(req) // nolint:bodyclose
		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if i < c.MaxRetries {
			logging.Warn(fmt.Sprintf("上报 Event 数据到 %v 失败， 将进行第 %d 次重试", c.postUrl, i+1),
				map[string]interface{}{"err": err})
		} else {
			logging.Warn(fmt.Sprintf("上报 Event 到 %v 失败, 停止上报：已达重试上限", c.postUrl),
				map[string]interface{}{"err": err, "event": event})
			return
		}
		delay := c.InitDelay << i
		if delay > c.MaxDelay {
			delay = c.MaxDelay
		}
		// 添加随机抖动
		jitter := time.Duration(rand.Int63n(int64(delay) / 2))
		delay += jitter
		time.Sleep(delay)
	}
	defer resp.Body.Close()

	logging.Debug("上报Event数据到 "+c.postUrl, map[string]interface{}{"event": event})
	if resp.Body == nil {
		logging.Warn("返回Body数据为空", map[string]interface{}{"err": err})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logHttpPostError(event, err, "response body read failed")
	}
	if resp.StatusCode != 200 {
		logHttpPostError(event, nil, "response status code not 200")
	}

	logging.Debug("response body: "+string(body), map[string]interface{}{"event": event})
}

func logHttpPostError(event event.IEvent, err error, msg string) {
	err = fmt.Errorf("%w %w", oneboterr.ErrHttpPost, err)
	logging.ErrorWithErr(err, msg, map[string]interface{}{"event": event})
}

// encrypt http post 加密 secret
func encrypt(secret string) string {
	key := []byte(secret)
	hash := sha256.New()
	hash.Write(key)
	secreted := hash.Sum(nil)
	return hex.EncodeToString(secreted)
}
