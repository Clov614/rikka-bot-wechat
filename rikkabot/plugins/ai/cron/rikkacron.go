// Package cron
// @Author Clover
// @Data 2024/9/17 下午7:36:00
// @Desc 定时器
package cron

import (
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/common"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

var cronServer *MyCronServer

func init() {
	cronServer = NewCronServer()
}

type MyCronServer struct {
	*cron.Cron
	jobId2cron map[string]cron.EntryID // JobId为定时任务唯一标识，同一类型的定时任务只能有一个
	// todo 保存各个cron插件的信息，利用信息在init中恢复定时任务
}

func NewCronServer() *MyCronServer {
	c := cron.New(cron.WithParser(cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)))
	// 从设置里读取cronJob相关信息

	return &MyCronServer{
		Cron:       c,
		jobId2cron: make(map[string]cron.EntryID),
	}
}

func (cs *MyCronServer) init() {
	// todo 恢复之前持久化的定时任务

}

func (cs *MyCronServer) NewJob(spec string, job cron.Job, id string) bool {
	cs.RemoveCron(id) // 如果定时任务已经存在，则替换
	entryID, err := cs.AddJob(spec, job)
	if err != nil {
		log.Err(err).Msg("添加定时任务失败")
		return false
	}
	cs.jobId2cron[id] = entryID
	cs.Start()
	return true
}

func (cs *MyCronServer) ResetPluginCron(jobId, spec string) {
	logging.Info(fmt.Sprintf("Cron jobId: %s, plugin spec: %s", jobId, spec))
	if id, ok := cs.jobId2cron[jobId]; ok {
		job := cs.Cron.Entry(id).Job
		newId, err := cs.AddJob(spec, job)
		if err != nil {
			log.Err(err).Msg("重置定时任务时间失败")
			return
		}
		cs.jobId2cron[jobId] = newId
		cs.Remove(id)
	}
}

func (cs *MyCronServer) RemoveCron(id string) {
	if id, ok := cs.jobId2cron[id]; ok {
		cs.Remove(id)
	}
}

type CronJob struct {
	*common.Self
	Spec    string
	Uuid    string
	IsGroup bool
	JobName string
}

func (cj *CronJob) CreateSchedule(job cron.Job) {
	cj.Self = common.GetSelf()
	if cronServer.NewJob(cj.Spec, job, cj.GetJobId()) {
		cj.SendText("定时任务设置成功")
	} else {
		cj.SendText("定时任务设置失败")
	}
	// todo 持久化，管理相关

}

func (cj *CronJob) GetJobId() string {
	return cj.Uuid + "_" + cj.JobName
}

func (cj *CronJob) SendText(text string) {
	err := cj.SendTextByUuid(cj.Uuid, text, cj.IsGroup)
	if err != nil {
		log.Err(err).Msg("定时推送消息错误")
	}
}
