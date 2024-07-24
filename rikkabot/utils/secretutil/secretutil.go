// Package secretutil
// @Author Clover
// @Data 2024/7/24 下午2:48:00
// @Desc 加密摘要工具类
package secretutil

import (
	"fmt"
	"hash/fnv"
)

// generateFixedLengthID 生成固定长度的唯一标识符
func generateFixedLengthID(data string, length int) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(data)) // ignore err
	sum := hash.Sum64()

	// 转换为十六进制字符串
	hexStr := fmt.Sprintf("%x", sum)

	// 如果字符串长度不够，填充 '0'
	for len(hexStr) < length {
		hexStr = "0" + hexStr
	}

	// 截取到所需长度
	return hexStr[:length]
}

func GenerateUnitId(data string) string {
	return generateFixedLengthID(data, 16)
}
