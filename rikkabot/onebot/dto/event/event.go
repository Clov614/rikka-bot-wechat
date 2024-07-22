// Package event
// @Author Clover
// @Data 2024/7/19 下午9:44:00
// @Desc oneBot 事件封装
package event

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/utils/timeutil"
)

type IEvent interface{}

type Event struct {
	Id         string  `json:"id"`
	Time       float64 `json:"time"`
	Type       string  `json:"type"`
	DetailType string  `json:"detail_type"`
	SubType    string  `json:"sub_type"`
}

func (e *Event) InitEvent(etype string, detailType string, subType string) {
	e.Id = uuid.New().String()
	e.Time = timeutil.GetTimeUnix()
	e.Type = etype
	e.DetailType = detailType
	e.SubType = subType
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

func (m *MsgEvent) InitMsgEvent(msgs ...message.Message) *MsgEvent {
	m.Message = msgs
	return m
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

type IEventPool interface {
	AddEvent(event Event) error
	GetEvent() (Event, error)
	StartProcessing(handlers ...func(event Event))
	Close()
}

// EventPool 事件池
type EventPool struct {
	PoolSize int64
	Events   chan IEvent
	once     sync.Once
	wg       sync.WaitGroup
	quit     chan struct{}
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
		Events:   make(chan IEvent, poolSize),
		quit:     make(chan struct{}),
	}
}

// AddEvent 添加事件
func (ep *EventPool) AddEvent(event IEvent) error {
	select {
	case ep.Events <- event:
		return nil
	default:
		return fmt.Errorf("EventPool is full")
	}
}

// GetEvent 获取事件
func (ep *EventPool) GetEvent() (IEvent, error) {
	select {
	case event := <-ep.Events:
		return event, nil
	default:
		return Event{}, fmt.Errorf("no events in pool")
	}
}

// StartProcessing starts processing events from pool
func (ep *EventPool) StartProcessing(handlers ...func(event IEvent)) {
	ep.wg.Add(1)
	go func() {
		defer ep.wg.Done()
		for {
			select {
			case <-ep.quit:
				return
			default:
				event, err := ep.GetEvent()
				if err != nil || event == nil {
					time.Sleep(time.Millisecond * 100) // Wait and retry if no events
					continue
				}
				for _, handler := range handlers {
					ep.wg.Add(1)
					go func(h func(event IEvent), e IEvent) {
						defer ep.wg.Done()
						h(e)
					}(handler, event)
				}
			}
		}
	}()
}

// Close closes the event pool
func (ep *EventPool) Close() {
	time.Sleep(1 * time.Second)
	ep.once.Do(func() {
		close(ep.quit)
		close(ep.Events)
	})
	ep.wg.Wait()
}
