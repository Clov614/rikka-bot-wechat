// @Author Clover
// @Data 2024/7/7 下午11:15:00
// @Desc
package cache

import (
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control"
	"testing"
)

// 触发器 触发校验 测试
func TestIsHandle(t *testing.T) {

	var tests = []struct {
		*control.ProcessRules
		*message.Message
		want bool
	}{
		{ProcessRules: &control.ProcessRules{IsCallMe: true, EnableGroup: true}, // 0
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka hello",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true, EnableGroup: true}, // 1
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true}, // 2
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true}, // 3
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: false}, // 4
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		// 测试群组白名单
		{ProcessRules: &control.ProcessRules{CheckWhiteGroup: true, EnableGroup: true}, // 5 群消息，存在白名单
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteGroup: false, EnableGroup: true}, // 6 不开启，不存在白名单中
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "123123", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteGroup: true, EnableGroup: true}, // 7 群消息，不存在白名单
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "12312311", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteGroup: true}, // 8 不是群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		// 测试群组黑名单
		{ProcessRules: &control.ProcessRules{CheckBlackGroup: true, EnableGroup: true}, // 9 存在黑名单
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackGroup: true}, // 10 不是群组消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "safsdfas",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackGroup: true, IsCallMe: true, EnableGroup: true}, // 11 存在黑名单，但是 callme
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackGroup: true, IsCallMe: true}, // 12 存在黑名单，但是 callme 不是群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackGroup: true, IsCallMe: true, EnableGroup: true}, // 13 不存在黑名单，但是 callme
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "123123123123", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		// 测试管理员
		{ProcessRules: &control.ProcessRules{IsAdmin: true, EnableGroup: true}, // 14 自己管理员测试, 群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "2788092443", ReceiverId: "2788092443", IsGroup: true, IsMySelf: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsAdmin: true}, // 15 自己管理员测试, 用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "2788092443", ReceiverId: "2788092443", IsGroup: false, IsMySelf: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsAdmin: true, EnableGroup: true}, // 16 别的管理员测试, 群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsAdmin: true, EnableGroup: true}, // 17 别的非管理员测试, 群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{IsAdmin: true}, // 18 别的管理员测试, 用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsAdmin: true}, // 19 别的非管理员测试, 用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: false},
			want: false,
		},

		// 不设置规则
		{ProcessRules: &control.ProcessRules{}, // 20 别的非管理员测试, 用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},

		// 用户权限测试
		// 白名单用户
		{ProcessRules: &control.ProcessRules{CheckWhiteUser: true}, // 21 自己，用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "2788092443", ReceiverId: "2788092443", IsGroup: false, IsMySelf: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteUser: true}, // 22 白，用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteUser: true}, // 23 其他，用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: false},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteUser: true, EnableGroup: true}, // 24 自己，群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "2788092443", ReceiverId: "2788092443", IsGroup: true, IsMySelf: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteUser: true, EnableGroup: true}, // 25 白，群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckWhiteUser: true, EnableGroup: true}, // 26 其他人，群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},

		// 黑名单用户
		{ProcessRules: &control.ProcessRules{CheckBlackUser: true}, // 27 自己，用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "2788092443", ReceiverId: "2788092443", IsGroup: false, IsMySelf: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackUser: true}, // 28 黑，用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: false},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackUser: true}, // 29 其他，用户消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: false},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackUser: true, EnableGroup: true}, // 30 自己，群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "2788092443", ReceiverId: "2788092443", IsGroup: true, IsMySelf: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackUser: true, EnableGroup: true}, // 31 黑，群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "813466966", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{CheckBlackUser: true, EnableGroup: true}, // 32 其他人，群消息
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka nihao",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},

		// 测试 execOrder
		{ProcessRules: &control.ProcessRules{IsCallMe: true, ExecOrder: []string{"echo"}, EnableGroup: true}, // 33 符合order
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "  /rikka echo  123456 ",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true, ExecOrder: []string{"echo"}, EnableGroup: true}, // 34 不符合order
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "  /rikka ec ho  123456 ",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true, ExecOrder: []string{"echo"}, EnableGroup: true}, // 35 不符合order
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka  12344 123456 ",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: false, ExecOrder: []string{"echo"}, EnableGroup: true}, // 36 不符合order
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "/rikka  echo 123456 ",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: false,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true, ExecOrder: []string{"echo", "Echo"}, EnableGroup: true}, // 37 符合order
			Message: &message.Message{Msgtype: message.MsgTypeText, Content: "  /rikka Echo  123456 ",
				GroupId: "813467281", SenderId: "12312333", ReceiverId: "2788092443", IsGroup: true},
			want: true,
		},

		// todo test 测试自定义功能， 消息类型

	}

	Init()

	cache.AddWhiteGroupId("813467281")
	cache.AddBlackGroupId("813467281")

	cache.AddAdminUserId("813466966")

	cache.AddWhiteUserId("813466966")
	cache.AddBlackUserId("813466966")

	for i, test := range tests {
		_, ok, _ := cache.IsHandle(test.ProcessRules, *test.Message)
		if ok != test.want {
			t.Errorf("cache.isHandle(%drules) = %v, want %v", i, ok, test.want)
		}
	}
	cache.Close()
}

func TestAlone(t *testing.T) {
	var tests = []struct {
		*control.ProcessRules
		*message.Message
		want bool
	}{
		{ProcessRules: &control.ProcessRules{IsCallMe: true, IsAdmin: true, EnableGroup: true,
			ExecOrder: []string{"add whitelist", "加入白名单"}}, // 0 bug
			Message: &message.Message{IsMySelf: true, IsGroup: true,
				GroupId: "813467281 ", SenderId: "2788092443", ReceiverId: "2788092443",
				Content: "/rikka add whitelist"},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true, IsAdmin: true, EnableGroup: true,
			ExecOrder: []string{"add whitelist", "加入白名单"}}, // 1 bug
			Message: &message.Message{IsMySelf: true, IsGroup: false,
				GroupId: "", SenderId: "2788092443", ReceiverId: "2788092443",
				Content: "/rikka add whitelist"},
			want: true,
		},
		{ProcessRules: &control.ProcessRules{IsCallMe: true, IsAdmin: true, EnableGroup: true,
			ExecOrder: []string{"add whitelist", "加入白名单"}}, // 1 bug
			Message: &message.Message{IsMySelf: false, IsGroup: false,
				GroupId: "", SenderId: "7777777777", ReceiverId: "2788092443",
				Content: "/rikka add whitelist"},
			want: false,
		},
	}

	Init()

	for i, test := range tests {
		_, ok, _ := cache.IsHandle(test.ProcessRules, *test.Message)
		if ok != test.want {
			t.Errorf("cache.isHandle(%drules) = %v, want %v", i, ok, test.want)
		}
	}
	cache.Close()
}
