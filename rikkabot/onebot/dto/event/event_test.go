// Package event
// @Author Clover
// @Data 2024/7/21 下午4:07:00
// @Desc 事件池测试
package event

import (
	"math"
	"testing"
	"time"
	"wechat-demo/rikkabot/message"
)

import "github.com/google/uuid"

func TestEventPool(t *testing.T) {
	eventPool := NewEventPool(10)

	event := MsgEvent{
		Event: Event{
			Id:         uuid.New().String(),
			Time:       getTimeUnix(),
			Type:       "message",
			DetailType: "private",
			SubType:    "",
		},
		Message: []message.Message{
			{
				Msgtype:         message.MsgTypeText,
				MetaData:        nil,
				Raw:             nil,
				RawContent:      "",
				Content:         "测试消息",
				GroupId:         "66666666",
				SenderId:        "123123",
				ReceiverId:      "222222",
				GroupNameList:   nil,
				GroupAtNameList: nil,
				IsAtMe:          true,
				IsGroup:         false,
				IsFriend:        false,
				IsMySelf:        false,
				IsSystem:        false,
			},
		},
	}
	err := eventPool.AddEvent(event)
	if err != nil {
		t.Error(err)
	}
	getEvent, err := eventPool.GetEvent()
	if err != nil {
		t.Error(err)
	}

	t.Logf("%#v", getEvent)
	t.Logf("%f", getEvent.(MsgEvent).Event.Time)

	err = eventPool.AddEvent(event)
	if err != nil {
		t.Error(err)
	}

	var hds []func(event IEvent)

	hds = make([]func(event IEvent), 0)

	hds = append(hds, func(event IEvent) {
		println("start processing event")
		switch event.(type) {
		case MsgEvent:
			println(event.(MsgEvent).Message[0].Content)
		}
	})

	hds = append(hds, func(event IEvent) {
		println("start processing event")
		switch event.(type) {
		case MsgEvent:
			println(event.(MsgEvent).Message[0].Msgtype)
		}
	})

	eventPool.StartProcessing(hds...)

	eventPool.Close()
}

func getTimeUnix() float64 {
	currentTime := float64(time.Now().UnixNano()) / 1e9
	return math.Round(currentTime*1e6) / 1e6
}
