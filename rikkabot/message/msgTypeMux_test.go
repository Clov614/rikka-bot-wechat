// Package message
// @Author Clover
// @Data 2024/12/29 下午7:21:00
// @Desc
package message

import (
	"testing"
)

func TestMsgTypeMutex_Mutex(t *testing.T) {
	m := GetMsgTypeMux()

	type args struct {
		pluginName string
		msg        Message
	}
	tests := []struct {
		name                string
		args                args
		registedMsgTypeList MsgTypeList
		want                bool
	}{
		{
			name: "test01",
			args: args{
				pluginName: "plugin01",
				msg: Message{
					Msgtype: MsgTypeText,
				},
			},
			registedMsgTypeList: MsgTypeList{
				MsgTypeText,
				MsgTypeImage,
			},
			want: true,
		},
		{
			name: "test02",
			args: args{
				pluginName: "plugin02",
				msg: Message{
					Msgtype: MsgTypeApp,
				},
			},
			registedMsgTypeList: MsgTypeList{
				MsgTypeText,
				MsgTypeImage,
			},
			want: false,
		},
		{
			name: "test03",
			args: args{
				pluginName: "plugin03",
				msg: Message{
					Msgtype: MsgTypeApp,
				},
			},
			registedMsgTypeList: MsgTypeList{
				MsgTypeApp,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.RegistByPluginName(tt.args.pluginName, tt.registedMsgTypeList) // 注册允许的消息类型
			if got := m.Mux(tt.args.pluginName, tt.args.msg); got != tt.want {
				t.Errorf("Mux() = %v, want %v", got, tt.want)
			}
		})
	}
}
