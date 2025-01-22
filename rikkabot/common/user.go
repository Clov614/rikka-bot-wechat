// Package common
// @Author Clover
// @Data 2024/7/14 下午10:45:00
// @Desc 对cli的进一步封装，方便其他模块操作
package common

import (
	"context"
	wcf "github.com/Clov614/wcf-rpc-sdk"
)

type Self struct {
	cli *wcf.Client
}

var self *Self

var (
// ErrFriendNotFound  = errors.New("friend not found")
// ErrGroupNotFound   = errors.New("group not found")
)

func GetSelf() *Self {
	return self
}

func InitSelf(ctx context.Context, cli *wcf.Client) {
	self = &Self{cli}
}

func (s *Self) GetNickName() string {
	return s.cli.GetSelfName()
}

func (s *Self) SendText(receiver string, content string, ats ...string) error {
	return s.cli.SendText(receiver, content, ats...)
}
