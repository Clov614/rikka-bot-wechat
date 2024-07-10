// @Author Clover
// @Data 2024/7/7 下午9:57:00
// @Desc
package msgutil

import (
	"strings"
	"wechat-demo/rikkabot"
)

// 校验是否呼唤机器人 默认不区分大小写
func IsCallMe(s string) bool {
	bot := rikkabot.Bot()
	config := bot.Config
	symbol := config.Symbol
	botname := config.Botname
	s = strings.TrimSpace(s)
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(symbol+botname))
}

func IsOrder(content string, order string) bool {
	content = strings.TrimSpace(content)
	return strings.HasPrefix(content, order)
}

// 去除呼唤机器人的部分 获得剩余部分
func TrimCallMe(s string) string {
	bot := rikkabot.Bot()
	config := bot.Config
	symbol := config.Symbol
	botname := config.Botname
	s = strings.TrimSpace(s)
	return TrimPrefix(strings.ToLower(s), strings.ToLower(symbol+botname), false, true)
}

func TrimPrefix(s string, prefix string, caseSensitive bool, isTrimSpace bool) string {
	if !caseSensitive {
		return strings.TrimPrefix(strings.ToLower(s), strings.ToLower(prefix))
	}
	if isTrimSpace {
		return strings.TrimSpace(strings.TrimPrefix(s, prefix))
	}
	return strings.TrimPrefix(s, prefix)
}

// ContainsInt checks if a slice contains a specific integer
func ContainsInt(slice []int, element int) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
