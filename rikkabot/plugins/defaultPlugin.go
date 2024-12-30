// Package plugins
// @Author Clover
// @Data 2024/7/6 下午11:21:00
// @Desc 系统自带的插件
package plugins

import (
	_ "wechat-demo/rikkabot/plugins/admin"         // 需要副作用 init注册方法
	_ "wechat-demo/rikkabot/plugins/ai"            // 需要副作用 init注册方法
	_ "wechat-demo/rikkabot/plugins/biliUrlDecode" // 需要副作用 init注册方法
	_ "wechat-demo/rikkabot/plugins/game"          // 需要副作用 init注册方法
)

func init() {
}
