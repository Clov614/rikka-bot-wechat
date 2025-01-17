// Package event
// @Author Clover
// @Data 2024/7/22 下午2:49:00
// @Desc 事件动作
package event

import "github.com/Clov614/rikka-bot-wechat/rikkabot/message"

type Echo string

type ActionRequest[T SendMsgParams | any] struct {
	Action string `json:"action"` // 动作名称
	Params T      `json:"params"` // 动作参数
	Echo   `json:"echo,omitempty"`
	//Self   `json:"self,omitempty"`
}

type SendMsgParams struct {
	DetailType string          `json:"detail_type"`
	MsgType    message.MsgType `json:"msg_type"`
	SendId     string          `json:"send_id"`
	Message    any             `json:"message"`
}

type ActionResponse struct {
	Status  string      `json:"status"` // ok or failed
	Retcode int64       `json:"retcode"`
	Data    interface{} `json:"data"`    // 动作响应消息
	Message string      `json:"message"` // 错误信息
	Echo    `json:"echo,omitempty"`
}

type MsgRespData struct {
	Time      float64 `json:"time"`
	MessageId string  `json:"message_id"`
}
