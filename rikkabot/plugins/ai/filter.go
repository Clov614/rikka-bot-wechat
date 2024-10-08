// Package ai
// @Author Clover
// @Data 2024/8/30 下午5:48:00
// @Desc 过滤器
package ai

import (
	"fmt"
	"github.com/go-ego/gse"
	"strings"
	"unicode/utf8"
)

var (
	seg gse.Segmenter
)

var DefaultFilter *Filter

type Filter struct {
	seg gse.Segmenter // 分词器
}

func (f *Filter) desensitize(word string) string {
	cutWords := f.seg.Cut(word, true)
	var targetWords = make([]string, len(cutWords))
	var ti int = 0
	for _, w := range cutWords {
		if _, exist := sensitiveWordsMap[w]; exist {
			targetWords[ti] = w
			ti++
		}
	}
	// desensitize
	for i := 0; i < ti; i++ {
		mask := strings.Repeat("*", utf8.RuneCountInString(targetWords[i]))
		word = strings.ReplaceAll(word, targetWords[i], mask)
	}
	return word
}

func (f *Filter) filter(input string, handle func(content string) (string, error)) (res string, err error) {
	output, err := handle(f.desensitize(input))
	if err != nil {
		return "", fmt.Errorf("filter failed: %w", err)
	}
	return f.desensitize(output), nil
}

func init() {
	// 加载默认词典
	_ = seg.LoadDict()
	// 加载默认 embed 词典
	// seg.LoadDictEmbed()
	//
	// 加载简体中文词典
	_ = seg.LoadDict("zh_s")
	_ = seg.LoadDictEmbed("zh_s")
	//
	// 加载繁体中文词典
	_ = seg.LoadDict("zh_t")
	//
	// 加载日文词典
	// seg.LoadDict("jp")
	// 加载自定义词典
	if sensitiveWords == nil {
		sensitiveWords = strings.Split(sensitiveStr, "\n")
	}
	customDtMap := make([]map[string]string, len(sensitiveWords))
	for i, word := range sensitiveWords {
		customDtMap[i] = map[string]string{
			"text": word,
			"freq": "1200",
		}
	}
	_ = seg.LoadDictMap(customDtMap) // 加载自定义词典

	DefaultFilter = &Filter{
		seg: seg,
	}
}
