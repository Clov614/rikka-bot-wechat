// Package admin
// @Author Clover
// @Data 2025/3/17 上午12:40:00
// @Desc
package admin

import (
	"context"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"testing"
)

func TestAdminMatcher_Match(t *testing.T) {
	// 创建一些 MockMatcher 用于测试
	tests := []struct {
		text    string
		msg     *message.Message
		handler plugins.ActionHandler
		want    bool
	}{
		{
			text: "op help",
			want: true,
		},
		{
			text: "op @AkiAoi-evil ",
			want: true,
		},
		{
			text: "op @123456 ",
			want: true,
		},
		{
			text: "op -d @123456 ",
			want: true,
		},
	}
	sendChan := make(chan *message.Message, len(tests))
	defer close(sendChan)
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			isExec := AA.HandleRecv(context.Background(), &message.Message{Content: tt.text, Msgtype: message.MsgTypeText, IsMySelf: true}, sendChan)
			if isExec != tt.want {
				t.Errorf("isExec = %v, want %v", isExec, tt.want)
			}
		})
	}

}
