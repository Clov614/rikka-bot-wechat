// Package matchers
// @Author Clover
// @Data 2025/3/16 下午12:33:00
// @Desc
package matcher

import (
	"context"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
)

// FunctionMatcher 自定义函数匹配器
type FunctionMatcher struct {
	MatchFunc func(ctx context.Context, msg *message.Message) bool
}

func (fm *FunctionMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if fm.MatchFunc == nil {
		return false
	}
	return fm.MatchFunc(ctx, msg)
}
