// @Author Clover
// @Data 2024/7/6 下午3:28:00
// @Desc
package config

import (
	"fmt"
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

func (c *CommonConfig) verifycfgterm() {
	// 校验 延迟选项
	if !(c.AnswerDelayRandMin >= 1) {
		fmt.Println("Error: The Random Delay Random Min must >= 1, now using default") // 提醒
		c.AnswerDelayRandMin = defaultAnswerDelayRandMin
		c.AnswerDelayRandMax = defaultAnswerDelayRandMax
	}
	if !(c.AnswerDelayRandMax >= c.AnswerDelayRandMin) {
		fmt.Println("Error: The Random Delay Random Max must >= c.AnswerDelayRandMin, now using default") // 错误提示 todo 优化日志显示
		c.AnswerDelayRandMin = defaultAnswerDelayRandMin
		c.AnswerDelayRandMax = defaultAnswerDelayRandMax
	}

}

var defaultPath = "./cfg/"

var defaultSaveFileName = "config.yaml"

func init() {
	// todo 先判断是否已经存在持久化存储文件，存在先读取更新设置项

	err := configutil.Load(&config, defaultPath, defaultSaveFileName)
	config.verifycfgterm() // 校验设置项是否合规
	err = configutil.Save(&config, defaultPath, defaultSaveFileName)
	if err != nil {
		fmt.Println("error saving config:", err)
	}
}
