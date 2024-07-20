// Package oneboterr
// @Author Clover
// @Data 2024/7/19 下午9:32:00
// @Desc OneBot 相关错误返回码
package oneboterr

import "errors"

const (
	OK = 0

	// 1xxxx 动作请求错误
	BAD_REQUEST              = 10001 // 无效的动作请求	格式错误（包括实现不支持 MessagePack 的情况）、必要字段缺失或字段类型错误
	UNSUPPORTED_ACTION       = 10002 // 不支持的动作请求	OneBot 实现没有实现该动作
	BAD_PARAME               = 10003 // 无效的动作请求参数	参数缺失或参数类型错误
	UNSUPPORTED_PARAM        = 10004 // 不支持的动作请求参数	OneBot 实现没有实现该参数的语义
	UNSUPPORTED_SEGMENT      = 10005 // 不支持的消息段类型	OneBot 实现没有实现该消息段类型
	BAD_SEGMENT_DATA         = 10006 // 无效的消息段参数	参数缺失或参数类型错误
	UNSUPPORTED_SEGMENT_DATA = 10007 // 不支持的消息段参数	OneBot 实现没有实现该参数的语义
	WHO_AM_I                 = 10101 // 未指定机器人账号	OneBot 实现在单个 OneBot Connect 连接上支持多个机器人账号，但动作请求未指定要使用的账号
	UNKNOWN_SELF             = 10012 // 未知的机器人账号	动作请求指定的机器人账号不存在

	// 2xxxx 动作处理器错误
	BAD_HANDLER            = 20001 // 动作处理器实现错误	没有正确设置响应状态等
	INTERNAL_HANDLER_ERROR = 20002 // 动作处理器运行时抛出异常	OneBot 实现内部发生了未捕获的意料之外的异常

	// 3xxxx 动作执行错误

)

var (
	ErrHttpPost = errors.New("http 上报器错误")
)
