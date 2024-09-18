// Package cronplugins
// @Author Clover
// @Data 2024/9/17 下午8:03:00
// @Desc
package cronplugins

import (
	"encoding/json"
	"fmt"
	aisdk "github.com/Clov614/go-ai-sdk"
	"github.com/Clov614/go-ai-sdk/global"
	"github.com/rs/zerolog/log"
	"strconv"
	"wechat-demo/rikkabot/plugins/ai/cron"
	"wechat-demo/rikkabot/plugins/ai/wechatSports/zepplife"
)

type StepSetPlugin struct {
	*cron.CronJob
	Account  string `json:"account"`
	Password string `json:"password"`
	Steps    int    `json:"steps"`
}

func NewStepSetPlugin(account string, password string, steps int, cronJob *cron.CronJob) *StepSetPlugin {
	return &StepSetPlugin{
		CronJob:  cronJob,
		Account:  account,
		Password: password,
		Steps:    steps,
	}
}

func (p *StepSetPlugin) Call(params string) (jsonStr string, err error) {
	type stepProperties struct {
		Uuid     string `json:"uuid"`
		IsGroup  bool   `json:"is_group"`
		CronSpec string `json:"cron_spec"`
		Account  string `json:"account"`
		Secret   string `json:"password"`
		Steps    int    `json:"steps"`
	}
	var properties stepProperties
	err = json.Unmarshal([]byte(params), &properties)
	if err != nil {
		return "", fmt.Errorf("call_err: json.Unmarshal([]byte(params)): %w", err)
	}
	p.Uuid = properties.Uuid
	p.IsGroup = properties.IsGroup
	p.Spec = properties.CronSpec
	p.JobName = "set_wechat_sport_steps"

	p.Account = properties.Account
	p.Password = properties.Secret
	p.Steps = properties.Steps

	p.CreateSchedule(p)
	res := "[系统] 设置微信步数" + strconv.Itoa(p.Steps) + "步定时任务成功 cron: " + p.Spec
	p.SendText(res)
	return res, nil
}

func (p *StepSetPlugin) Run() {
	zeppLife := zepplife.NewZeppLife(p.Account, p.Password)
	err := zeppLife.SetSteps(p.Steps)
	if err != nil {
		log.Err(err).Msg("定时任务: 设置微信步数失败")
		p.SendText("[系统] 设置微信步数失败")
		return
	} else {
		p.SendText(fmt.Sprintf("[系统] 定时刷取微信步数:%d 成功", p.Steps))
	}

}

func init() {
	// 注册定时任务触发的ai插件
	funcCallInfo := aisdk.FuncCallInfo{
		Function: aisdk.Function{
			Name:        "set_wechat_sport_steps_schedule",
			Description: "定时设置微信运动步数(通过zeppLife api刷微信步数)！如未提供账号密码还请提醒用户给出！",
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
						Description: "定时任务的cron表达式,6字段的cron表达式，按照该格式给出(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)",
					},
					"account": aisdk.Property{
						Type:        global.StringType,
						Description: "账号",
					},
					"password": aisdk.Property{
						Type:        global.StringType,
						Description: "密码",
					},
					"steps": aisdk.Property{
						Type:        global.IntType,
						Description: "微信步数",
					},
				},
				Required: []string{"uuid", "is_group", "cron_spec", "account", "password", "steps"},
			},
			Strict: false,
		},
		CallFunc: &StepSetPlugin{
			CronJob: &cron.CronJob{},
		},
	}

	aisdk.FuncRegister.Register(&funcCallInfo, []string{"定时任务", "定时", "刷步数", "步数"})
}
