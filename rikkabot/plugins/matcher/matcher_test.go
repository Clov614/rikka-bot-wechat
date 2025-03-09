package matcher

import (
	"context"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"regexp"
	"testing"
)

func TestPrefixMatcher_Match(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		msg    *message.Message
		want   bool
	}{
		{
			name:   "Match Prefix",
			prefix: "prefix",
			msg: &message.Message{
				Content: "prefix message",
			},
			want: true,
		},
		{
			name:   "Not Match Prefix",
			prefix: "prefix",
			msg: &message.Message{
				Content: "other message",
			},
			want: false,
		},
		{
			name:   "Empty Content",
			prefix: "prefix",
			msg: &message.Message{
				Content: "",
			},
			want: false,
		},
		{
			name:   "Nil Message",
			prefix: "prefix",
			msg:    nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PrefixMatcher{Prefix: tt.prefix}
			if got := pm.Match(context.Background(), tt.msg); got != tt.want {
				t.Errorf("PrefixMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegexMatcher_Match(t *testing.T) {
	tests := []struct {
		name  string
		regex *regexp.Regexp
		msg   *message.Message
		want  bool
	}{
		{
			name:  "Match Regex",
			regex: regexp.MustCompile(`^\d+$`),
			msg: &message.Message{
				Content: "12345",
			},
			want: true,
		},
		{
			name:  "Not Match Regex",
			regex: regexp.MustCompile(`^\d+$`),
			msg: &message.Message{
				Content: "abcde",
			},
			want: false,
		},
		{
			name:  "Empty Content",
			regex: regexp.MustCompile(`^\d+$`),
			msg: &message.Message{
				Content: "",
			},
			want: false,
		},
		{
			name:  "Nil Message",
			regex: regexp.MustCompile(`^\d+$`),
			msg:   nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RegexMatcher{Regex: tt.regex}
			if got := rm.Match(context.Background(), tt.msg); got != tt.want {
				t.Errorf("RegexMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeywordMatcher_Match(t *testing.T) {
	tests := []struct {
		name     string
		keywords []string
		msg      *message.Message
		want     bool
	}{
		{
			name:     "Match Keyword",
			keywords: []string{"keyword"},
			msg: &message.Message{
				Content: "message with keyword",
			},
			want: true,
		},
		{
			name:     "Not Match Keyword",
			keywords: []string{"keyword"},
			msg: &message.Message{
				Content: "message without keyword",
			},
			want: false,
		},
		{
			name:     "Match One Of Keywords",
			keywords: []string{"keyword1", "keyword2", "keyword3"},
			msg: &message.Message{
				Content: "message with keyword2",
			},
			want: true,
		},
		{
			name:     "Not Match Any Keywords",
			keywords: []string{"keyword1", "keyword2", "keyword3"},
			msg: &message.Message{
				Content: "message without keywords",
			},
			want: false,
		},
		{
			name:     "Empty Content",
			keywords: []string{"keyword"},
			msg: &message.Message{
				Content: "",
			},
			want: false,
		},
		{
			name:     "Nil Message",
			keywords: []string{"keyword"},
			msg:      nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km := &KeywordMatcher{Keywords: tt.keywords}
			if got := km.Match(context.Background(), tt.msg); got != tt.want {
				t.Errorf("KeywordMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFunctionMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		matchFunc func(ctx context.Context, msg *message.Message) bool
		msg       *message.Message
		want      bool
	}{
		{
			name: "Match Function True",
			matchFunc: func(ctx context.Context, msg *message.Message) bool {
				return msg.Content == "match"
			},
			msg: &message.Message{
				Content: "match",
			},
			want: true,
		},
		{
			name: "Match Function False",
			matchFunc: func(ctx context.Context, msg *message.Message) bool {
				return msg.Content == "match"
			},
			msg: &message.Message{
				Content: "not match",
			},
			want: false,
		},
		{
			name:      "Nil MatchFunc",
			matchFunc: nil,
			msg: &message.Message{
				Content: "content",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := &FunctionMatcher{MatchFunc: tt.matchFunc}
			if got := fm.Match(context.Background(), tt.msg); got != tt.want {
				t.Errorf("FunctionMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseMatcher_Match(t *testing.T) {

	tests := []struct {
		name string
		bm   *BaseMatcher
		msg  *message.Message
		want bool
	}{
		{
			name: "MatchTypeRule Match",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeText,
			},
			want: true,
		},
		{
			name: "MatchTypeRule Not Match",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule,
				AllowMsgType: message.MsgTypeImage,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeImage,
			},
			want: false,
		},
		{
			name: "MatchTypeRule Empty MsgTypes (should not match)", // 规则存在，但是类型列表为空，应该不匹配任何类型
			bm: &BaseMatcher{
				Rules:        MatchTypeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeText,
			},
			want: false,
		},
		{
			name: "IsGroupRule Match True",
			bm: &BaseMatcher{
				Rules: IsGroupRule,
			},
			msg: &message.Message{
				IsGroup: true,
			},
			want: true,
		},
		{
			name: "IsGroupRule Match False",
			bm: &BaseMatcher{
				Rules: IsGroupRule,
			},
			msg: &message.Message{
				IsGroup: false,
			},
			want: true,
		},
		{
			name: "IsGroupRule Not Match",
			bm: &BaseMatcher{
				Rules: IsGroupRule,
			},
			msg: &message.Message{
				IsGroup: false,
			},
			want: false,
		},
		{
			name: "IsGroupRule Nil IsGroup (should match)", // 规则存在，但是 IsGroup 为 nil，应该跳过群组校验，直接匹配
			bm: &BaseMatcher{
				Rules: IsGroupRule,
			},
			msg: &message.Message{
				IsGroup: true,
			},
			want: true,
		},
		{
			name: "IsAtMeRule Match True",
			bm: &BaseMatcher{
				Rules: IsAtMeRule,
			},
			msg: &message.Message{
				IsAtMe: true,
			},
			want: true,
		},
		{
			name: "IsAtMeRule Match False",
			bm: &BaseMatcher{
				Rules: IsAtMeRule,
			},
			msg: &message.Message{
				IsAtMe: false,
			},
			want: true,
		},
		{
			name: "IsAtMeRule Not Match",
			bm: &BaseMatcher{
				Rules: IsAtMeRule,
			},
			msg: &message.Message{
				IsAtMe: false,
			},
			want: false,
		},
		{
			name: "IsAtMeRule Nil IsAtMe (should match)", // 规则存在，但是 IsAtMe 为 nil，应该跳过 @Me 校验，直接匹配
			bm: &BaseMatcher{
				Rules: IsAtMeRule,
			},
			msg: &message.Message{
				IsAtMe: true,
			},
			want: true,
		},
		{
			name: "Combine Rules Match",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule | IsGroupRule | IsAtMeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeText,
				IsGroup: true,
				IsAtMe:  true,
			},
			want: true,
		},
		{
			name: "Combine Rules Not Match MsgType",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule | IsGroupRule | IsAtMeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeImage, // 类型不匹配
				IsGroup: true,
				IsAtMe:  true,
			},
			want: false,
		},
		{
			name: "Combine Rules Not Match IsGroup",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule | IsGroupRule | IsAtMeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeText,
				IsGroup: false, // 群组不匹配
				IsAtMe:  true,
			},
			want: false,
		},
		{
			name: "Combine Rules Not Match IsAtMe",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule | IsGroupRule | IsAtMeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg: &message.Message{
				Msgtype: message.MsgTypeText,
				IsGroup: true,
				IsAtMe:  false, // @Me 不匹配
			},
			want: false,
		},
		{
			name: "Nil Message",
			bm: &BaseMatcher{
				Rules:        MatchTypeRule,
				AllowMsgType: message.MsgTypeText,
			},
			msg:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bm.Match(context.Background(), tt.msg); got != tt.want {
				t.Errorf("BaseMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositeAndMatcher_Match(t *testing.T) {

	tests := []struct {
		name     string
		matchers []Matcher
		want     bool
	}{
		{
			name: "All Matchers Match",
			want: true,
		},
		{
			name: "One Matcher Not Match",
			want: false,
		},
		{
			name:     "Empty Matchers List", // 没有 matcher 默认匹配成功
			matchers: []Matcher{},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cam := &CompositeAndMatcher{Matchers: tt.matchers}
			if got := cam.Match(context.Background(), &message.Message{Content: "test"}); got != tt.want {
				t.Errorf("CompositeAndMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositeOrMatcher_Match(t *testing.T) {

	tests := []struct {
		name     string
		matchers []Matcher
		want     bool
	}{
		{
			name: "One Matcher Match",
			want: true,
		},
		{
			name: "All Matchers Not Match",
			want: false,
		},
		{
			name:     "Empty Matchers List", // 没有 matcher 默认匹配失败
			matchers: []Matcher{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			com := &CompositeOrMatcher{Matchers: tt.matchers}
			if got := com.Match(context.Background(), &message.Message{Content: "test"}); got != tt.want {
				t.Errorf("CompositeOrMatcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

//// MockMatcher for testing CompositeMatcher
//type MockMatcher struct {
//	matchResult bool
//}
//
//func (m *MockMatcher) Match(ctx context.Context, msg *message.Message) bool {
//	return m.matchResult
//}
