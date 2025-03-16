// Package matchers
// @Author Clover
// @Data 2025/3/16 下午12:32:00
// @Desc 字符串相关匹配器
package matcher

import (
	"context"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"regexp"
	"strings"
)

// PrefixMatcher  前缀匹配器
type PrefixMatcher struct {
	Prefix          string
	IsCaseSensitive bool
	IsCut           bool // 是否切除匹配
}

func (pm *PrefixMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil || msg.Content == "" {
		return false
	}
	var flag bool
	if pm.IsCaseSensitive {
		flag = strings.HasPrefix(strings.ToLower(msg.Content), strings.ToLower(pm.Prefix))
	} else {
		flag = strings.HasPrefix(msg.Content, pm.Prefix)
	}
	if pm.IsCut {
		msg.Content = strings.TrimSpace(strings.TrimPrefix(msg.Content, pm.Prefix))
	}
	return flag
}

// RegexMatcher 正则表达式匹配器
type RegexMatcher struct {
	Regex *regexp.Regexp
}

func (rm *RegexMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil || msg.Content == "" {
		return false
	}
	return rm.Regex.MatchString(msg.Content)
}

// KeywordMatcher 关键词匹配器
type KeywordMatcher struct {
	Keywords []string
}

func (km *KeywordMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil || msg.Content == "" {
		return false
	}
	for _, keyword := range km.Keywords {
		if strings.Contains(msg.Content, keyword) {
			return true
		}
	}
	return false
}
