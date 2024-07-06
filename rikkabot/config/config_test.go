// @Author Clover
// @Data 2024/7/6 下午3:49:00
// @Desc
package config

import (
	"testing"
)

func TestConfigInit(t *testing.T) {
	// init not need to call code
}

func TestPluginInit(t *testing.T) {
	type EConfig struct {
		Name     string `comment:"插件名称"`
		Loop     bool   `comment:"是否循环"`
		interval int    `comment:"间隔时间"`
	}

	var testConfig = EConfig{
		Name:     "测试插件01",
		Loop:     false,
		interval: 2,
	}

	commonConfig := GetConfig()
	commonConfig.RegistConfig("test01", &testConfig)
	commonConfig.RegistConfig("test02", &testConfig)
	err := commonConfig.Update()
	if err != nil {
		t.Error(err)
	}
}
