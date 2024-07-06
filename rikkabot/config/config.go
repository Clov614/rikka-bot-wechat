// @Author Clover
// @Data 2024/7/6 下午3:28:00
// @Desc
package config

import (
	"fmt"
	"wechat-demo/rikkabot/utils/configutil"
)

type CommonConfig struct {
	Symbol  string `comment:"The Robot Prefix default to ‘/’ "`
	Botname string `comment:"The Robot Name default to \"rikka\""`
	// todo 其他设置项
	PluginConfig map[string]interface{} `comment:"插件的设置" yaml:"plugin_config"`
}

// todo 动态设置项的注册以及持久化管理

var config = CommonConfig{
	Symbol:  "/",
	Botname: "rikka",
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

var defaultPath = "./cfg/"

var defaultSaveFileName = "config.yaml"

func init() {
	// todo 先判断是否已经存在持久化存储文件，存在先读取更新设置项

	err := configutil.Load(&config, defaultPath, defaultSaveFileName)
	err = configutil.Save(&config, defaultPath, defaultSaveFileName)
	if err != nil {
		fmt.Println("error saving config:", err)
	}
}
