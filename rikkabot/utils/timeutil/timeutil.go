// Package timeutil
// @Author Clover
// @Data 2024/7/21 下午11:39:00
// @Desc
package timeutil

import (
	"math"
	"strconv"
	"time"
)

func GetTimeUnix() float64 {
	currentTime := float64(time.Now().UnixNano()) / 1e9
	return math.Round(currentTime*1e6) / 1e6
}

// GetTimeStamp 获取 10位 时间戳
func GetTimeStamp() string {
	unix := time.Now().Unix()
	return strconv.FormatInt(unix, 10)
}

// GetNowDate 获取当前年月日
func GetNowDate() string {
	return time.Now().Format("2006-01-02")
}

// IsBeforeThatDay 是否早于 （现在日期 - day）
func IsBeforeThatDay(oDateStr string, offSetDay int) bool {
	targetDate := time.Now().AddDate(0, 0, -offSetDay)
	// 比较传入日期是否早于该日期
	originDate, _ := time.Parse("2006-01-02", oDateStr)
	return originDate.Before(targetDate)
}
