// @Author Clover
// @Data 2024/7/7 下午9:57:00
// @Desc
package msgutil

import (
	"strings"
)

// 校验是否呼唤机器人 默认不区分大小写
func IsCallMe(me string, s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(me))
}

func IsOrder(orders []string, content string) (isorder bool, order string) {
	content = strings.TrimSpace(content)
	for _, s := range orders {
		s = strings.TrimSpace(s)
		if strings.HasPrefix(content, s) {
			isorder = true
			order = s
		}
	}
	return isorder, order
}

// 去除呼唤机器人的部分 获得剩余部分
func TrimCallMe(me string, s string) string {
	s = strings.TrimSpace(s)
	return TrimPrefix(strings.ToLower(s), strings.ToLower(me), false, true)
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
