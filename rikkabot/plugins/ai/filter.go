// Package ai
// @Author Clover
// @Data 2024/8/30 下午5:48:00
// @Desc 过滤器
package ai

import (
	"fmt"
	"github.com/go-ego/gse"
)

var (
	seg gse.Segmenter
)

var DefaultFilter *Filter

type Filter struct {
	seg gse.Segmenter // 分词器
}

func (f *Filter) isLegal(word string) bool {
	cutWords := f.seg.CutAll(word)
	for _, w := range cutWords {
		if _, exist := sensitiveWordsMap[w]; exist {
			return false
		}
	}
	return true
}

func (f *Filter) filter(input string, handle func(content string) (string, error)) (res string, err error) {
	if !f.isLegal(input) {
		return "filtered", nil
	}
	output, err := handle(input)
	if err != nil {
		return "", fmt.Errorf("filter failed: %w", err)
	}
	if !f.isLegal(output) {
		return "filtered", nil
	}
	return output, nil
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
	DefaultFilter = &Filter{
		seg: seg,
	}
}
