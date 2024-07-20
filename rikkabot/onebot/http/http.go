// Package http
// @Author Clover
// @Data 2024/7/20 下午9:37:00
// @Desc http and http webhook
package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
	secret          string
	postUrl         string
	timeout         int32
	client          *http.Client
	MaxRetries      uint64
	RetriesInterval uint64
}

func (c *HttpClient) Run() {
	if c.timeout < 5 {
		c.timeout = 5
	}

	c.client = &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}
	logging.Info(fmt.Sprintf("Http Post 上报器已启动！%s", c.postUrl))
	// todo 添加 handler
}

// HandlerPostEvent 处理 post 事件
func (c *HttpClient) HandlerPostEvent(event event.Event) {

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

func logHttpPostError(event event.Event, err error, msg string) {
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
