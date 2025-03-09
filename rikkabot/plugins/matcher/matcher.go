// Package matcher
// @Author Clover
// @Data 2025/3/7 下午5:48:00
// @Desc 规则校验
package matcher

import (
	"context"
	"regexp"
	"strings"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
)

// Matcher 接口，你需要定义具体的 Matcher 接口
type Matcher interface {
	Match(ctx context.Context, msg *message.Message) bool
}

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

// BaseMatcherRule 定义 BaseMatcher 的规则常量，使用位运算
type BaseMatcherRule uint32

const (
	MatchTypeRule BaseMatcherRule = 1 << iota // 消息类型规则 (1 << 0)
	IsGroupRule                               // 群组消息规则 (1 << 1)
	IsAtMeRule                                // @Me 规则 (1 << 2)
	IsSelf                                    // 是否是自己的消息
	// 可以继续添加其他规则，例如 IsFriendRule, IsSystemRule 等
)

// BaseMatcher 基础匹配器，支持组合多种规则
type BaseMatcher struct {
	Rules        BaseMatcherRule // 使用位或组合的规则
	AllowMsgType message.MsgType
}

func (bm *BaseMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil {
		return false
	}
	if bm.AllowMsgType != 0 && msg.Msgtype&bm.AllowMsgType == 0 {
		return false
	}
	if bm.Rules != 0 {
		switch {
		case bm.Rules&IsGroupRule != 0:
			if !msg.IsGroup {
				return false
			}
		case bm.Rules&IsAtMeRule != 0:
			if !msg.IsAtMe {
				return false
			}
		case bm.Rules&IsSelf != 0:
			if !msg.IsMySelf {
				return false
			}
		}
	}
	select {
	case <-ctx.Done():
		return false
	default:
	}
	return true
}

// CompositeAndMatcher  与组合匹配器，需要所有子匹配器都匹配成功
type CompositeAndMatcher struct {
	Matchers []Matcher
}

func DefaultAnd(ml ...Matcher) *CompositeAndMatcher {
	return &CompositeAndMatcher{
		Matchers: ml,
	}
}

func (cam *CompositeAndMatcher) AsAnd(m Matcher) *CompositeAndMatcher {
	if cam == nil {
		cam.Matchers = []Matcher{m}
		return cam
	}
	cam.Matchers = append(cam.Matchers, m)
	return cam
}

func (cam *CompositeAndMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if len(cam.Matchers) == 0 {
		return true // 没有子匹配器时，默认匹配成功 (可以根据需求调整)
	}
	for _, matcher := range cam.Matchers {
		if !matcher.Match(ctx, msg) {
			return false // 只要有一个子匹配器不匹配，就返回 false
		}
	}
	return true // 所有子匹配器都匹配成功，返回 true
}

// CompositeOrMatcher  或组合匹配器，只要有一个子匹配器匹配成功
type CompositeOrMatcher struct {
	Matchers []Matcher
}

func DefaultOr(ml ...Matcher) *CompositeOrMatcher {
	return &CompositeOrMatcher{
		Matchers: ml,
	}
}

func (com *CompositeOrMatcher) AsOr(m Matcher) *CompositeOrMatcher {
	if com == nil {
		com.Matchers = []Matcher{m}
		return com
	}
	com.Matchers = append(com.Matchers, m)
	return com
}

func (com *CompositeOrMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if len(com.Matchers) == 0 {
		return false // 没有子匹配器时，默认匹配失败 (可以根据需求调整)
	}
	for _, matcher := range com.Matchers {
		if matcher.Match(ctx, msg) {
			return true // 只要有一个子匹配器匹配成功，就返回 true
		}
	}
	return false // 所有子匹配器都匹配失败，返回 false
}

type CustomMatcher struct {
	MatchFunc func(msg *message.Message) bool
}

func (cm *CustomMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if cm != nil && cm.MatchFunc != nil && cm.MatchFunc(msg) {
		return true
	}
	return false
}
