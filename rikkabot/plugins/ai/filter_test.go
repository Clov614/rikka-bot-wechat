// Package ai
// @Author Clover
// @Data 2024/8/30 下午9:39:00
// @Desc
package ai

import (
	"github.com/go-ego/gse"
	"testing"
)

func TestFilter_filter(t *testing.T) {
	type fields struct {
		seg gse.Segmenter
	}
	type args struct {
		input  string
		handle func(content string) (string, error)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRes string
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				seg: seg,
			},
			args: args{
				input:  "共产党",
				handle: func(content string) (string, error) { return content, nil },
			},
			wantRes: "***",
		},
		{
			name: "test2",
			fields: fields{
				seg: seg,
			},
			args: args{
				input:  "rikka",
				handle: func(content string) (string, error) { return content, nil },
			},
			wantRes: "rikka",
		},
		{
			name: "test3",
			fields: fields{
				seg: seg,
			},
			args: args{
				input:  "习主席",
				handle: func(content string) (string, error) { return content, nil },
			},
			wantRes: "***",
		}, {
			name: "test4",
			fields: fields{
				seg: seg,
			},
			args: args{
				input:  "那看到小偷走进房间呢\nSeeing the thief walk into the room, he called the police ",
				handle: func(content string) (string, error) { return content, nil },
			},
			wantRes: "那看到小偷走进房间呢\nSeeing the thief walk into the room, he called the police ",
		}, {
			name: "test5",
			fields: fields{
				seg: seg,
			},
			args: args{
				input:  "1三个代表123平反456平凡",
				handle: func(content string) (string, error) { return content, nil },
			},
			wantRes: "1****123**456平凡",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Filter{
				seg: tt.fields.seg,
			}
			gotRes, err := f.filter(tt.args.input, tt.args.handle)
			if (err != nil) != tt.wantErr {
				t.Errorf("filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRes != tt.wantRes {
				t.Errorf("filter() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}