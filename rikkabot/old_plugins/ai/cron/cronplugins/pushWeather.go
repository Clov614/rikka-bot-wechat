// Package cronplugins
// @Author Clover
// @Data 2024/9/17 下午8:27:00
// @Desc 定时获取天气模块
package cronplugins

import (
	"encoding/json"
	"fmt"
	aisdk "github.com/Clov614/go-ai-sdk"
	"github.com/Clov614/go-ai-sdk/example_func_call/weather"
	"github.com/Clov614/go-ai-sdk/global"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/old_plugins/ai/cron"
	"strings"
)

type weatherCfg struct {
	Key string `json:"key"`
}

var (
	w *weather.Weather
)

type WeatherCronPlugin struct {
	*cron.CronJob
	*weather.Weather
	city string
}

func NewWeatherPlugin(city string, cronJob *cron.CronJob) *WeatherCronPlugin {
	return &WeatherCronPlugin{
		Weather: w,
		CronJob: cronJob,
		city:    city,
	}
}

func (p *WeatherCronPlugin) Call(params string) (jsonStr string, err error) {
	type weatherScheduleTaskProperties struct {
		Uuid     string `json:"uuid"`
		IsGroup  bool   `json:"is_group"`
		CronSpec string `json:"cron_spec"`
		CityAddr string `json:"city_addr"`
	}
	var properties weatherScheduleTaskProperties
	err = json.Unmarshal([]byte(params), &properties)
	if err != nil {
		return "", fmt.Errorf("call_err: json.Unmarshal([]byte(params)): %w", err)
	}
	p.Uuid = properties.Uuid
	p.IsGroup = properties.IsGroup
	p.Spec = properties.CronSpec
	p.JobName = "push_weather"

	p.Weather = w
	p.city = properties.CityAddr
	p.CreateSchedule(p)
	res := "[系统] 添加" + p.city + "天气推送成功 cron: " + p.Spec
	p.SendText(res)
	return res, nil
}

func (p *WeatherCronPlugin) Run() {
	weatherResp := p.Weather.GetWeatherByCityAddr(p.city, true)
	if len(weatherResp.Forecasts) > 0 {
		p.SendText(getMultiDayWeatherString(weatherResp.Forecasts))
	}
}

// 美观输出多日天气信息
func getMultiDayWeatherString(forecasts []weather.Forecast) string {
	var result strings.Builder

	for _, forecast := range forecasts {
		result.WriteString(fmt.Sprintf("🏙️ 城市：%s\n", forecast.City))
		result.WriteString(fmt.Sprintf("📅 报告时间：%s\n", forecast.ReportTime))
		for _, cast := range forecast.Casts {
			result.WriteString(fmt.Sprintf("📅 日期：%s\n", cast.Date))
			result.WriteString(fmt.Sprintf("🌞 白天天气：%s 🌙 夜晚天气：%s\n", cast.DayWeather, cast.NightWeather))
			result.WriteString(fmt.Sprintf("🌡️ 白天温度：%s°C 🌙 夜晚温度：%s°C\n", cast.DayTemp, cast.NightTemp))
			result.WriteString(fmt.Sprintf("🌬️ 白天风向：%s 💨 风力：%s\n", cast.DayWind, cast.DayPower))
			result.WriteString(fmt.Sprintf("🌬️ 夜晚风向：%s 💨 风力：%s\n", cast.NightWind, cast.NightPower))
		}
		result.WriteString("--------------------------------------------------\n")
	}

	return result.String()
}

func init() {
	// 获取天气相关配置
	cfg := config.GetConfig()
	wCfgInterface, ok := cfg.GetCustomPluginCfg("weather_ai")
	if !ok {
		cfg.SetCustomPluginCfg("weather_ai", weatherCfg{Key: ""})
		_ = cfg.Update() // 更新设置
		logging.Fatal("weather_ai plugin config loaded empty please write the key about weather api in config.yaml", 12)
	}
	bytes, err := json.Marshal(wCfgInterface)
	if err != nil {
		logging.Fatal("weather_ai plugin config marshalling failed", 12)
	}
	var wcfg weatherCfg
	err = json.Unmarshal(bytes, &wcfg)
	if err != nil {
		logging.Fatal("weather_ai plugin config unmarshal fail", 12)
	}
	w = weather.NewWeather(wcfg.Key)

	// 注册定时任务触发的ai插件
	funcCallInfo := aisdk.FuncCallInfo{
		Function: aisdk.Function{
			Name:        "push_weather_schedule",
			Description: "定期推送天气信息",
			Parameters: aisdk.FunctionParameter{
				Type: global.ObjType,
				Properties: aisdk.Properties{
					"uuid": aisdk.Property{
						Type:        global.StringType,
						Description: "标记定时任务的标识",
					},
					"is_group": aisdk.Property{
						Type:        global.BoolType,
						Description: "是否为群聊",
					},
					"cron_spec": aisdk.Property{
						Type:        global.StringType,
						Description: "定时任务cron表达式，请根据我的描述生成6字段的cron表达式，只返回表达式字符串:(每天早上八点半执行: 0 30 8 1/1 * ?)",
					},
					"city_addr": aisdk.Property{
						Type:        global.StringType,
						Description: "地址，如：国家，城市，县、区地址",
					},
				},
				Required: []string{"uuid", "is_group", "cron_spec", "city_addr"},
			},
			Strict: false,
		},
		CallFunc: &WeatherCronPlugin{
			Weather: w,
			CronJob: &cron.CronJob{},
		},
	}

	aisdk.FuncRegister.Register(&funcCallInfo, []string{"定时任务", "定时", "推送"})
}
