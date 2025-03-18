// Package ai
// @Author Clover
// @Data 2024/8/30 下午5:48:00
// @Desc 过滤器
package ai

import (
	"encoding/json"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/go-ego/gse"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var (
	seg gse.Segmenter
)

var DefaultFilter *Filter

var sensitiveWordsMap = make(map[string]bool)

type Filter struct {
	seg gse.Segmenter
}

// FilterResult 过滤结果
type FilterResult struct {
	IsSensitive bool   `json:"is_sensitive"` // 是否包含敏感词
	Level       int    `json:"level"`        // 敏感词等级
	Words       string `json:"words"`        // 敏感词列表
}

// SensitiveLexicon 敏感词库
type SensitiveLexicon struct {
	LastUpdateDate string   `json:"lastUpdateDate"`
	Words          []string `json:"words"`
}

func init() {
	InitFilter()
}

func GetFilter() *Filter {
	return DefaultFilter
}

func (f *Filter) filter(input string, handle func(content string) (string, error)) (res string, err error) {
	output, err := handle(f.desensitize(input))
	if err != nil {
		return "", fmt.Errorf("filter failed: %w", err)
	}
	return f.desensitize(output), nil
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

// InitFilter 初始化过滤器
func InitFilter() {
	// 优先加载本地敏感词库
	sensitiveWords, err := loadSensitiveWordsFromFile("./data/sensitive/word.json")
	if err != nil {
		logging.ErrorWithErr(err, "从本地文件加载敏感词库失败")
		// 尝试从远程加载
		sensitiveWords, err = loadSensitiveWordsFromGitHub("https://raw.githubusercontent.com/konsheng/Sensitive-lexicon/main/ThirdPartyCompatibleFormats/TrChat/SensitiveLexicon.json")
		if err != nil {
			logging.ErrorWithErr(err, "从 GitHub 加载敏感词库失败")
			// 远程加载也失败，放弃加载敏感词
			logging.Warn("放弃加载敏感词库")
		} else {
			logging.Info("从 GitHub 加载敏感词库成功")
			// 保存到本地文件
			err = saveSensitiveWordsToFile("./data/sensitive/word.json", sensitiveWords)
			if err != nil {
				logging.ErrorWithErr(err, "保存敏感词库到本地文件失败")
			}
		}
	} else {
		logging.Info("从本地文件加载敏感词库成功")
	}

	// 如果加载了敏感词，则添加到 gse.Segmenter 的词典中
	if sensitiveWords != nil {
		// 加载自定义词典
		customDtMap := make([]map[string]string, len(sensitiveWords))
		for i, word := range sensitiveWords {
			sensitiveWordsMap[word] = true
			customDtMap[i] = map[string]string{
				"text": word,
				"freq": "1200",
			}
		}
		_ = seg.LoadDictMap(customDtMap) // 加载自定义词典
	}

	DefaultFilter = &Filter{
		seg: seg,
	}
}

// loadSensitiveWordsFromGitHub 从 GitHub 加载敏感词库
func loadSensitiveWordsFromGitHub(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lexicon SensitiveLexicon
	err = json.Unmarshal(body, &lexicon)
	if err != nil {
		return nil, err
	}

	return lexicon.Words, nil
}

// loadSensitiveWordsFromFile 从本地文件加载敏感词库
func loadSensitiveWordsFromFile(filePath string) ([]string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var lexicon SensitiveLexicon
	err = json.Unmarshal(data, &lexicon)
	if err != nil {
		return nil, err
	}

	return lexicon.Words, nil
}

// saveSensitiveWordsToFile 保存敏感词库到本地文件
func saveSensitiveWordsToFile(filePath string, words []string) error {
	lexicon := SensitiveLexicon{
		LastUpdateDate: "unknown", // 可以根据实际情况更新
		Words:          words,
	}

	data, err := json.MarshalIndent(lexicon, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile(filePath, data, 0644)
}
