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
	"io"
	"net/http"
	"time"
	"wechat-demo/rikkabot"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/onebot/dto/event"
	"wechat-demo/rikkabot/onebot/oneboterr"
)

// HttpServer http 服务
type HttpServer struct {
	HttpAddr    string
	AccessToken string // 鉴权
}

func (s *HttpServer) Run() {

}

// HttpClient 反向http
type HttpClient struct {
	secret     string
	postUrl    string
	timeout    int
	client     *http.Client
	MaxRetries int
	bot        *rikkabot.RikkaBot
}

// RunHttp 启动 http、http上报器
func RunHttp(rbot *rikkabot.RikkaBot) {

	// http server

	// http上报器
	for _, post := range rbot.Config.HttpPost {
		HttpClient{
			secret:     post.Secret,
			postUrl:    post.Url,
			timeout:    post.TimeOut,
			MaxRetries: post.MaxRetries,
			bot:        rbot,
		}.Run()
	}
}

func (c HttpClient) Run() {
	if c.timeout < 5 {
		c.timeout = 5
	}

	c.client = &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}
	logging.Info(fmt.Sprintf("Http Post 上报器已启动！%s", c.postUrl))
	// 注册事件处理
	c.bot.OnEventPush(c.HandlerPostEvent)
}

// HandlerPostEvent 处理 post 事件
func (c *HttpClient) HandlerPostEvent(event event.IEvent) {

	eventJSON, _ := json.Marshal(event)

	req, err := http.NewRequest("POST", c.postUrl, bytes.NewBuffer(eventJSON))
	if err != nil {
		logHttpPostError(event, err, "request create failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", encrypt(c.secret)))

	resp, err := c.client.Do(req)
	if err != nil {
		logHttpPostError(event, err, "request post failed")
	}
	if resp == nil || resp.Body == nil {
		logging.Fatal("Http上报器连接错误", 3, map[string]interface{}{"err": err})
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logging.ErrorWithErr(err, "close body failed")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logHttpPostError(event, err, "response body read failed")
	}
	if resp.StatusCode != 200 {
		logHttpPostError(event, nil, "response status code not 200")
	}

	logging.Debug(fmt.Sprintf("response body: %s", string(body)), map[string]interface{}{"event": event})
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
	bytes := hash.Sum(nil)
	return hex.EncodeToString(bytes)
}
