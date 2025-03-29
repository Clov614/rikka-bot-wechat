// Package matchers
// @Author Clover
// @Data 2025/3/16 下午12:32:00
// @Desc 字符串相关匹配器
package matcher

import (
	"context"
	"regexp"
	"strings"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
)

// PrefixMatcher  前缀匹配器
type PrefixMatcher struct {
	Prefixes        []string
	IsCaseSensitive bool
	IsCut           bool // 是否切除匹配
}

func (pm PrefixMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil || msg.Content == "" {
		return false
	}
	var flag bool
	for _, prefix := range pm.Prefixes {
		if pm.IsCaseSensitive {
			flag = strings.HasPrefix(strings.ToLower(msg.Content), strings.ToLower(prefix))
		} else {
			flag = strings.HasPrefix(msg.Content, prefix)
		}
		if flag {
			if pm.IsCut {
				msg.Content = strings.TrimSpace(strings.TrimPrefix(msg.Content, prefix))
			}
			return true
		}
	}
	return false
}

func NewPrefixMatcher(isCaseSensitive bool, isCut bool, prefixes ...string) PrefixMatcher {
	return PrefixMatcher{
		Prefixes:        prefixes,
		IsCaseSensitive: isCaseSensitive,
		IsCut:           isCut,
	}
}

// RegexMatcher 正则表达式匹配器
type RegexMatcher struct {
	Regexs []*regexp.Regexp
	IsCut  bool // 是否切除匹配
}

func (rm RegexMatcher) Match(ctx context.Context, msg *message.Message) bool {
	if msg == nil || msg.Content == "" {
		return false
	}
	for _, regex := range rm.Regexs {
		if regex.MatchString(msg.Content) {
			if rm.IsCut {
				msg.Content = strings.TrimSpace(regex.ReplaceAllString(msg.Content, ""))
			}
			return true
		}
	}
	return false
}

func NewRegexMatcher(isCut bool, regexs ...string) RegexMatcher {
	regexps := make([]*regexp.Regexp, 0, len(regexs))
	for _, regex := range regexs {
		regexps = append(regexps, regexp.MustCompile(regex))
	}
	return RegexMatcher{
		Regexs: regexps,
		IsCut:  isCut,
	}
}

// KeywordMatcher 关键词匹配器
type KeywordMatcher struct {
	Keywords []string
}

func (km KeywordMatcher) Match(ctx context.Context, msg *message.Message) bool {
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

func NewKeywordMatcher(keywords ...string) KeywordMatcher {
	return KeywordMatcher{
		Keywords: keywords,
	}
}
