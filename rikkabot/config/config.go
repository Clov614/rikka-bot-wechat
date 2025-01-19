// Package config
// @Author Clover
// @Data 2024/7/6 下午3:28:00
// @Desc 全局设置、并管理设置的周期持久化
package config

import (
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/configutil"
	"os"
	"sync"
)

type CommonConfig struct {
	Symbol                string `comment:"The Robot Prefix default to ‘/’ "`
	Botname               string `comment:"The Robot Name default to \"rikka\""`
	AnswerDelayRandMin    int    `comment:"The Random Delay Random Min default to 1" yaml:"answer_delay_rand_min"`
	AnswerDelayRandMax    int    `comment:"The Random Delay Random Max default to 3" yaml:"answer_delay_rand_max"`
	LogMaxSize            int    `comment:"The Max Log Size default to 10  means 10MB limit" yaml:"log_max_size,omitempty"`
	DBDirPath             string `comment:"数据库路径，默认: ./data/db" yaml:"db_dir_path,omitempty"`
	ImgDirPath            string `comment:"图片保存路径，默认: ./data/img" yaml:"img_dir_path,omitempty"`
	ImgSaveType           string `comment:"聊天图片保存方式，默认以文件保存(file: 文件保存；db: 数据桶保存)" yaml:"img_save_type,omitempty"`
	CacheSaveInterval     int    `comment:"缓存数据持久化间隔 单位为秒(默认60秒)" yaml:"cache_save_interval,omitempty"`
	ImgValidDuration      int    `comment:"聊天图片缓存有效时间 单位为天(默认7天)" yaml:"img_valid_duration,omitempty"`
	ImgCacheCheckInterval int    `comment:"聊天图片校验有效间隔 单位为小时(默认24小时)" yaml:"img_cache_check_interval,omitempty"`
	// OneBot settings
	// http 正向 HTTP API配置
	HttpServer HttpServerConfig `comment:"Http server config" yaml:"http_server"`
	// http 上报器
	HttpPost        []HttpPostConfig `comment:"Http 上报器，如不需要请注释掉" yaml:"http_post,omitempty"`
	EnableHeartBeat bool             `comment:"Enable Heart Beat" yaml:"enable_heart_beat"`
	Interval        int64            `comment:"The Heart Beat Interval" yaml:"heart_beat_interval"`
	// todo 其他设置项
	PluginConfig map[string]interface{} `comment:"插件的设置" yaml:"plugin_config"`

	mu sync.RWMutex
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
	defaultSymbol                = "/"
	defaultBotname               = "rikka"
	defaultAnswerDelayRandMin    = 1
	defaultAnswerDelayRandMax    = 3
	defaultHttpAdress            = "http://127.0.0.1:10614"
	defaultAccessToken           = "rikka-bot"
	defaultHeartBeat             = true // 默认开启心跳
	defaultInterval              = 5
	defaultLogLimit              = 10
	defaultDBDirPath             = "./data/db"
	defaultImgDirPath            = "./data/img"
	defaultImgSaveType           = "file" // 图片默认文件保存
	defaultCacheSaveInterval     = 60
	defaultImgValidDuration      = 7  // day
	defaultImgCacheCheckInterval = 24 // hour
)

var config = CommonConfig{
	Symbol:                defaultSymbol,
	Botname:               defaultBotname,
	AnswerDelayRandMin:    defaultAnswerDelayRandMin,
	AnswerDelayRandMax:    defaultAnswerDelayRandMax,
	LogMaxSize:            defaultLogLimit,
	DBDirPath:             defaultDBDirPath,
	ImgDirPath:            defaultImgDirPath,
	ImgSaveType:           defaultImgSaveType,
	CacheSaveInterval:     defaultCacheSaveInterval,
	ImgValidDuration:      defaultImgValidDuration,
	ImgCacheCheckInterval: defaultImgCacheCheckInterval,
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
	// 校验 http post 列表是否为空视为不开启主动post
	if len(c.HttpPost) == 1 && c.HttpPost[0].Url == "" && c.HttpPost[0].Secret == "" && c.HttpPost[0].MaxRetries == 0 {
		c.HttpPost = []HttpPostConfig{}
	}

}

// SetCustomPluginCfg 保存自定义插件设置
func (c *CommonConfig) SetCustomPluginCfg(pluginName string, pluginCfg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.PluginConfig[pluginName] = pluginCfg
}

func (c *CommonConfig) GetCustomPluginCfg(pluginName string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	plg, ok := c.PluginConfig[pluginName]
	return plg, ok
}

var defaultPath = "./cfg/"

var defaultSaveFileName = "config.yaml"

func init() {
	err := configutil.Load(&config, defaultPath, defaultSaveFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = configutil.Save(&config, defaultPath, defaultSaveFileName)
		}
		logging.ErrorWithErr(err, "error load config")
	}
	err = configutil.Save(&config, defaultPath, defaultSaveFileName)
	config.verifiability() // 校验设置项是否合规
	if err != nil {
		logging.ErrorWithErr(err, "error saving config")
	}
}
