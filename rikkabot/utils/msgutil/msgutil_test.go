// Package msgutil
// @Author Clover
// @Data 2025/3/17 上午10:32:00
// @Desc
package msgutil

import "testing"

func TestIsAtOne(t *testing.T) {
	type args struct {
		atText string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "1",
			args: args{atText: "@123456 "},
			want: true,
		},
		{
			name: "2",
			args: args{atText: "-d"},
			want: false,
		},
		{
			name: "3",
			args: args{atText: "help"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAtOne(tt.args.atText); got != tt.want {
				t.Errorf("IsAtOne() = %v, want %v", got, tt.want)
			}
		})
	}
}
