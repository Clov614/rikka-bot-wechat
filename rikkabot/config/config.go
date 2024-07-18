// Package config
// @Author Clover
// @Data 2024/7/6 下午3:28:00
// @Desc
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
	// todo 其他设置项
	PluginConfig map[string]interface{} `comment:"插件的设置" yaml:"plugin_config"`
}

// todo 动态设置项的注册以及持久化管理

const (
	defaultSymbol             = "/"
	defaultBotname            = "rikka"
	defaultAnswerDelayRandMin = 1
	defaultAnswerDelayRandMax = 3
)

var config = CommonConfig{
	Symbol:             defaultSymbol,
	Botname:            defaultBotname,
	AnswerDelayRandMin: defaultAnswerDelayRandMin,
	AnswerDelayRandMax: defaultAnswerDelayRandMax,
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
		return fmt.Errorf("update config failed. %v", err)
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
	config.verifiability() // 校验设置项是否合规
	err = configutil.Save(&config, defaultPath, defaultSaveFileName)
	if err != nil {
		logging.ErrorWithErr(err, "error saving config")
	}
}
