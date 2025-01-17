// Package plugins
// @Author Clover
// @Data 2024/12/30 下午1:38:00
// @Desc
package plugins

import (
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/msgutil"
	"testing"
)

func Test_autoAddNewFriend_SetIsVerify(t *testing.T) {
	type fields struct {
		isVerify   bool
		verifyText string
	}
	type args struct {
		content string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantedIsVerify bool
	}{
		{
			name: "test01",
			fields: fields{
				isVerify:   false,
				verifyText: "",
			},
			args: args{
				content: msgutil.TrimPrefix("verify true", "verify", false, true),
			},
			wantedIsVerify: true,
		},
		{
			name: "test02",
			fields: fields{
				isVerify:   false,
				verifyText: "",
			},
			args: args{
				content: msgutil.TrimPrefix("verify false", "verify", false, true),
			},
			wantedIsVerify: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := &autoAddNewFriend{
				Cache: cacheExported{
					IsVerify:   tt.fields.isVerify,
					VerifyText: tt.fields.verifyText,
				},
			}
			af.SetIsVerify(tt.args.content)
			if tt.wantedIsVerify != af.Cache.IsVerify {
				t.Errorf("autoAddNewFriend.SetIsVerify() = %v, want %v", af.Cache.IsVerify, tt.wantedIsVerify)
			}
		})
	}
}
