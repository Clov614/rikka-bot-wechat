// Package timeutil
// @Author Clover
// @Data 2024/7/21 下午11:39:00
// @Desc
package timeutil

import (
	"math"
	"time"
)

func GetTimeUnix() float64 {
	currentTime := float64(time.Now().UnixNano()) / 1e9
	return math.Round(currentTime*1e6) / 1e6
}
