// Package ai
// @Author Clover
// @Data 2024/8/15 下午10:38:00
// @Desc 天气模块
package ai

import (
	"encoding/json"
	ai_sdk "github.com/Clov614/go-ai-sdk"
	"github.com/Clov614/go-ai-sdk/example_func_call/weather"
	"github.com/Clov614/go-ai-sdk/global"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/logging"
)

type weatherCfg struct {
	Key string `json:"key"`
}

func init() {
	// 注册对话插件
	cfg := config.GetConfig()
	wCfgInterface, ok := cfg.GetCustomPluginCfg("weather_ai")
	if !ok {
		cfg.SetCustomPluginCfg("weather_ai", weatherCfg{Key: ""})
		_ = cfg.Update() // 更新设置
		logging.Fatal("weather_ai plugin config loaded empty please write the key about weather api in config.yaml", 12)
	}
	bytes, _ := json.Marshal(wCfgInterface)
	var wcfg weatherCfg
	err := json.Unmarshal(bytes, &wcfg)
	if err != nil {
		logging.Fatal("weather_ai plugin config unmarshal fail", 12)
	}
	w := weather.NewWeather(wcfg.Key)
	funcCallInfo := ai_sdk.FuncCallInfo{
		Function: ai_sdk.Function{
			Name:        "get_weather_by_city",
			Description: "根据地址获取城市代码 cityAddress: 城市地址，如: 泉州市永春县 isMultiDay: 是否获取多日天气",
			Parameters: ai_sdk.FunctionParameter{
				Type: global.ObjType,
				Properties: ai_sdk.Properties{
					"city_addr": ai_sdk.Property{
						Type:        global.StringType,
						Description: "地址，如：国家，城市，县、区地址",
					},
					"is_multi": ai_sdk.Property{
						Type:        global.BoolType,
						Description: "是否获取多日天气",
					},
				},
				Required: []string{"city_addr", "is_multi"},
			},
			Strict: false,
		},
		CallFunc: w,
		//CustomTrigger: nil, // 暂时不测试
	}
	ai_sdk.FuncRegister.Register(&funcCallInfo, []string{"天气", "weather"})
}
