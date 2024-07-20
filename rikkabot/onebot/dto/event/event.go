// Package event
// @Author Clover
// @Data 2024/7/19 下午9:44:00
// @Desc oneBot 事件封装
package event

import (
	"fmt"
	"sync"
	"time"
	"wechat-demo/rikkabot/message"
)

type Event struct {
	Id         string  `json:"id"`
	Time       float64 `json:"time"`
	Type       string  `json:"type"`
	DetailType string  `json:"detail_type"`
	SubType    string  `json:"sub_type"`
}

type Self struct {
	PlatForm string `json:"plat_form"`
	UserId   string `json:"user_id"`
}

// MsgEvent 消息事件 (type: event)
type MsgEvent struct {
	Event
	Message []message.Message `json:"message"`
}

// HeartBeatEvent 心跳事件 (type: meta)
type HeartBeatEvent struct {
	Event
	Interval int64 `json:"interval"`
}

// StateEvent 元事件 状态事件 (type: meta)
type StateEvent struct {
	Event
	Good bool `json:"good"`
	// todo 设计状态
	Status interface{} `json:"status"`
}

type Echo string

type ActionRequest struct {
	Action string         `json:"action"` // 动作名称
	Params map[string]any `json:"params"` // 动作参数
	Echo   `json:"echo,omitempty"`
	//Self   `json:"self,omitempty"`
}

type ActionResponse struct {
	Status  string      `json:"status"` // ok or failed
	Retcode int64       `json:"retcode"`
	Data    interface{} `json:"data"`    // 动作响应消息
	Message string      `json:"message"` // 错误信息
	Echo    `json:"echo,omitempty"`
}

// EventPool 事件池
type EventPool struct {
	PoolSize int64
	Events   chan Event
	once     sync.Once
}

const (
	maxpoolsize = 1000
)

// NewEventPool 创建事件线程池
func NewEventPool(poolSize int64) *EventPool {
	if poolSize <= 0 || poolSize > maxpoolsize {
		poolSize = maxpoolsize
	}
	return &EventPool{
		PoolSize: poolSize,
		Events:   make(chan Event, poolSize),
	}
}

// AddEvent 添加事件
func (ep *EventPool) AddEvent(event Event) error {
	select {
	case ep.Events <- event:
		return nil
	default:
		return fmt.Errorf("EventPool is full")
	}
}

// GetEvent 获取事件
func (ep *EventPool) GetEvent() (Event, error) {
	select {
	case event := <-ep.Events:
		return event, nil
	default:
		return Event{}, fmt.Errorf("no events in pool")
	}
}

// StartProcessing starts processing events from pool
func (ep *EventPool) StartProcessing(handlers ...func(event Event)) {
	go func() {
		for {
			event, err := ep.GetEvent()
			if err != nil {
				time.Sleep(time.Millisecond * 100) // Wait and retry if no events
				continue
			}
			for _, handler := range handlers {
				handler(event)
			}
		}
	}()
}

// Close closes the event pool
func (ep *EventPool) Close() {
	ep.once.Do(func() {
		close(ep.Events)
	})
}