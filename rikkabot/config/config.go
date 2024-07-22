// Package config
// @Author Clover
// @Data 2024/7/6 下午3:28:00
// @Desc 全局设置、并管理设置的周期持久化
package config

import (
	"fmt"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/utils/configutil"
)

type CommonConfig struct {
	Symbol             string `comment:"The Robot Prefix default to ‘/’ "`
	Botname            string `comment:"The Robot Name default to \"rikka\""`
	AnswerDelayRandMin int    `comment:"The Random Delay Random Min default to 1"`
	AnswerDelayRandMax int    `comment:"The Random Delay Random Max default to 3"`
	// OneBot settings
	// http 正向 HTTP API配置
	HttpServer HttpServerConfig `comment:"Http server config" yaml:"http_server"`
	// http 上报器
	HttpPost        []HttpPostConfig `comment:"Http 上报器，如不需要请注释掉" yaml:"http_post,omitempty"`
	EnableHeartBeat bool             `comment:"Enable Heart Beat" yaml:"enable_heart_beat"`
	Interval        int64            `comment:"The Heart Beat Interval" yaml:"heart_beat_interval"`
	// todo 其他设置项
	PluginConfig map[string]interface{} `comment:"插件的设置" yaml:"plugin_config"`
}

// HttpServerConfig http 正向 HTTP API配置
type HttpServerConfig struct {
	HttpAddress     string `comment:"The Robot HTTP Address default to http://127.0.0.1:10614" yaml:"http_address"`
	AccessToken     string `comment:"The Robot Access Token" yaml:"access_token"`
	EventEnabled    bool   `comment:"是否启用 get_latest_events 元动作" yaml:"event_enabled"`
	EventBufferSize int64  `comment:"事件缓冲区大小，超过该大小将会丢弃最旧的事件，0 表示不限大小" yaml:"event_buffer_size"`
}

// HttpPostConfig http 上报器配置
type HttpPostConfig struct {
	Url        string `comment:"The httpapi post URL" yaml:"url"`
	Secret     string `comment:"The httpapi post Access Token" yaml:"secret"`
	MaxRetries int    `comment:"The maximum number of retries" yaml:"max_retries"`
	TimeOut    int    `comment:"上报请求超时时间" yaml:"time_out"`
}

// todo 动态设置项的注册以及持久化管理

const (
	defaultSymbol             = "/"
	defaultBotname            = "rikka"
	defaultAnswerDelayRandMin = 1
	defaultAnswerDelayRandMax = 3
	defaultHttpAdress         = "http://127.0.0.1:10614"
	defaultAccessToken        = "rikka-bot"
	defaultHeartBeat          = true // 默认开启心跳
	defaultInterval           = 5
)

var config = CommonConfig{
	Symbol:             defaultSymbol,
	Botname:            defaultBotname,
	AnswerDelayRandMin: defaultAnswerDelayRandMin,
	AnswerDelayRandMax: defaultAnswerDelayRandMax,
	// OneBot
	HttpServer: HttpServerConfig{
		HttpAddress: defaultHttpAdress,
		AccessToken: defaultAccessToken,
	},
	HttpPost:        make([]HttpPostConfig, 1),
	EnableHeartBeat: defaultHeartBeat,
	Interval:        defaultInterval,
	// 其他设置项
	PluginConfig: make(map[string]interface{}),
}

func GetConfig() *CommonConfig {
	return &config
}

func (c *CommonConfig) RegistConfig(pulginname string, v interface{}) {
	c.PluginConfig[pulginname] = v
}

func (c *CommonConfig) Update() error {
	err := configutil.Save(c, defaultPath, defaultSaveFileName)
	if err != nil {
		return fmt.Errorf("update config failed. %w", err)
	}
	return nil
}

func (c *CommonConfig) verifiability() {
	// 校验 延迟选项
	if !(c.AnswerDelayRandMin >= 1) {
		logging.Warn("Error: The Random Delay Random Min must >= 1, now using default") // 提醒
		c.AnswerDelayRandMin = defaultAnswerDelayRandMin
		c.AnswerDelayRandMax = defaultAnswerDelayRandMax
	}
	if !(c.AnswerDelayRandMax >= c.AnswerDelayRandMin) {
		logging.Warn("Error: The Random Delay Random Max must >= c.AnswerDelayRandMin, now using default")
		c.AnswerDelayRandMin = defaultAnswerDelayRandMin
		c.AnswerDelayRandMax = defaultAnswerDelayRandMax
	}

}

var defaultPath = "./cfg/"

var defaultSaveFileName = "config.yaml"

func init() {
	err := configutil.Load(&config, defaultPath, defaultSaveFileName)
	if err != nil {
		logging.ErrorWithErr(err, "error load config")
	}
	config.verifiability() // 校验设置项是否合规
	err = configutil.Save(&config, defaultPath, defaultSaveFileName)
	if err != nil {
		logging.ErrorWithErr(err, "error saving config")
	}
}
