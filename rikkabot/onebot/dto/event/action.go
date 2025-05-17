// Package event
// @Author Clover
// @Data 2024/7/22 下午2:49:00
// @Desc 事件动作
package event

// OneBotMessageType 定义了 OneBot 标准中 send_message action 的 message_type 参数
type OneBotMessageType string

const (
	OneBotMessageTypePrivate OneBotMessageType = "private"
	OneBotMessageTypeGroup   OneBotMessageType = "group"
)

type Echo string

type ActionRequest[T SendMsgParams | any] struct {
	Action string `json:"action"` // 动作名称
	Params T      `json:"params"` // 动作参数
	Echo   `json:"echo,omitempty"`
	//Self   `json:"self,omitempty"`
}

// MessageSegment 定义了 OneBot 消息段的结构
type MessageSegment struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type SendMsgParams struct {
	MessageType OneBotMessageType `json:"message_type"`          // 消息类型 ("private", "group")
	UserId      string            `json:"user_id,omitempty"`     // 用户 ID (私聊时)
	GroupId     string            `json:"group_id,omitempty"`    // 群组 ID (群聊时)
	Message     []MessageSegment  `json:"message"`               // 消息内容 (OneBot 标准的消息段数组)
	AutoEscape  bool              `json:"auto_escape,omitempty"` // 是否自动转义 (OneBot V11 字段)
	SendId      string            `json:"send_id,omitempty"`     // 发送者 ID (兼容字段，私聊时为用户 ID, 群聊时为群组 ID)
	GroupIDs    []string          `json:"group_ids,omitempty"`   // 群组标签 ID 列表 (新增字段，用于通过群组标签发送消息)
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
