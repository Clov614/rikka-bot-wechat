// Package ai
// @Author Clover
// @Data 2024/8/30 下午5:48:00
// @Desc 过滤器
package ai

import (
	"fmt"
	"github.com/yanyiwu/gojieba"
)

var x *gojieba.Jieba = gojieba.NewJieba()

var defaultFilter *Filter

type Filter struct {
	x *gojieba.Jieba // 分词器
}

func (f *Filter) isLegal(word string) bool {
	cutWords := f.x.CutAll(word)
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
	defaultFilter = &Filter{
		x: x,
	}
}
