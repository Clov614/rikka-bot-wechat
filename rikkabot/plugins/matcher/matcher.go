// Package matcher
// @Author Clover
// @Data 2025/3/7 下午5:48:00
// @Desc 规则校验
package matcher

import (
	"context"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
)

// Matcher 接口
type Matcher interface {
	Match(ctx context.Context, msg *message.Message) bool
}

// MatcherMode 定义 DefaultMatcher 的组合模式
type MatcherMode uint8

const (
	AndMode  MatcherMode = iota // AND 模式: 所有子 Matcher 都必须匹配
	OrMode                      // OR 模式: 至少一个子 Matcher 匹配
	NandMode                    // NAND 模式: 与 AND 模式相反
	NorMode                     // NOR 模式: 与 OR 模式相反
)

// DefaultMatcher 默认匹配器 (支持嵌套和组合)
type DefaultMatcher struct {
	Matchers []Matcher       // 子 Matcher 列表
	Mode     MatcherMode     // 组合模式 (And, Or, Nand, Nor)
	last     *DefaultMatcher // 指向上一个 DefaultMatcher，用于嵌套
}

// Default 创建一个默认的 DefaultMatcher，初始模式为 OrMode
func Default() DefaultMatcher {
	return DefaultMatcher{Mode: OrMode}
}

// And 将当前 DefaultMatcher 的模式设置为 AndMode，并返回一个新的 DefaultMatcher (用于链式调用)
func (dm DefaultMatcher) And() DefaultMatcher {
	newDm := dm.deepCopy()
	if newDm.last == nil {
		newDm.Mode = AndMode
	}
	return DefaultMatcher{Mode: AndMode, last: newDm}
}

// Or 将当前 DefaultMatcher 的模式设置为 OrMode，并返回一个新的 DefaultMatcher (用于链式调用)
func (dm DefaultMatcher) Or() DefaultMatcher {
	newDm := dm.deepCopy()
	if newDm.last == nil {
		newDm.Mode = OrMode
	}
	return DefaultMatcher{Mode: OrMode, last: newDm}
}

// Nand 将当前 DefaultMatcher 的模式设置为 NandMode，并返回一个新的 DefaultMatcher (用于链式调用)
func (dm DefaultMatcher) Nand() DefaultMatcher {
	newDm := dm.deepCopy()
	if newDm.last == nil {
		newDm.Mode = NandMode
	}
	return DefaultMatcher{Mode: NandMode, last: newDm}
}

// Nor 将当前 DefaultMatcher 的模式设置为 NorMode，并返回一个新的 DefaultMatcher (用于链式调用)
func (dm DefaultMatcher) Nor() DefaultMatcher {
	newDm := dm.deepCopy()
	if newDm.last == nil {
		newDm.Mode = NorMode
	}
	return DefaultMatcher{Mode: NorMode, last: newDm}
}

// N 添加 n 个 Matcher 到当前 DefaultMatcher
func (dm DefaultMatcher) N(matchers ...Matcher) DefaultMatcher {
	dm.Matchers = append(dm.Matchers, matchers...)
	return dm
}

func (dm DefaultMatcher) match(ctx context.Context, msg *message.Message) (bool, bool) {
	var lastResult bool

	if dm.last != nil {
		lastResult, _ = dm.last.match(ctx, msg)
	}

	// 如果当前 DefaultMatcher 没有子 Matcher，则根据模式返回默认值
	if len(dm.Matchers) == 0 {
		if dm.last == nil {
			return dm.Mode == AndMode || dm.Mode == NorMode, true
		} else {
			return lastResult, false
		}
	}

	var currentResult bool
	switch dm.Mode {
	case AndMode:
		currentResult = true
		for _, matcher := range dm.Matchers {
			if !matcher.Match(ctx, msg) {
				currentResult = false
				break
			}
		}
	case OrMode:
		currentResult = false
		for _, matcher := range dm.Matchers {
			if matcher.Match(ctx, msg) {
				currentResult = true
				break
			}
		}
	case NandMode:
		currentResult = false
		for _, matcher := range dm.Matchers {
			if !matcher.Match(ctx, msg) {
				currentResult = true
				break
			}
		}
	case NorMode:
		currentResult = true
		for _, matcher := range dm.Matchers {
			if matcher.Match(ctx, msg) {
				currentResult = false
				break
			}
		}
	}
	switch dm.Mode {
	case AndMode:
		return currentResult && lastResult, false
	case OrMode:
		return currentResult || lastResult, false
	case NandMode:
		return !(currentResult && lastResult), false
	case NorMode:
		return !(currentResult || lastResult), false
	}
	return currentResult, false
}

// Match 检查消息是否匹配当前 DefaultMatcher 的规则
func (dm DefaultMatcher) Match(ctx context.Context, msg *message.Message) bool {
	ok, _ := dm.match(ctx, msg)
	return ok
}

// Not 匹配结果取反
type Not struct {
	m Matcher
}

func DefaultNot(m Matcher) Not {
	return Not{m}
}

func (n Not) N(m Matcher) Not {
	n.m = m
	return n
}

func (n Not) Match(ctx context.Context, msg *message.Message) bool {
	return !n.m.Match(ctx, msg)
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

func (bm BaseMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil {
		return false
	}
	if bm.AllowMsgType != 0 && msg.Msgtype&bm.AllowMsgType == 0 {
		return false
	}
	if bm.Rules != 0 {
		if bm.Rules&IsGroupRule != 0 && !msg.IsGroup {
			return false
		}
		if bm.Rules&IsAtMeRule != 0 && !msg.IsAtMe {
			return false
		}
		if bm.Rules&IsSelf != 0 && !msg.IsMySelf {
			return false
		}
	}
	select {
	case <-ctx.Done():
		return false
	default:
	}
	return true
}

type Custom struct {
	MatchFunc func(msg *message.Message) bool
}

func (cm Custom) Match(ctx context.Context, msg *message.Message) bool {
	if cm.MatchFunc != nil && cm.MatchFunc(msg) {
		return true
	}
	return false
}

func (dm DefaultMatcher) deepCopy() *DefaultMatcher {
	newDm := &DefaultMatcher{
		Mode:     dm.Mode,
		Matchers: make([]Matcher, len(dm.Matchers)),
		last:     dm.last,
	}
	copy(newDm.Matchers, dm.Matchers)
	return newDm
}
