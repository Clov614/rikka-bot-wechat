// Package plugins
// @Author Clover
// @Data 2024/7/6 下午11:21:00
// @Desc 系统自带的插件
package plugins

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/admin"         // 需要副作用 init注册方法
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/ai"            // 需要副作用 init注册方法
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/biliUrlDecode" // 需要副作用 init注册方法
	_ "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/game"          // 需要副作用 init注册方法
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control/dialog"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/msgutil"
)

const (
	autoAddnewfriendName = "auto-add-new-friend-manager"
	autoAddnewfriendCore = "auto-add-new-friend-core"
	pluginCallName       = "auto-add"
)

var (
	addFriendErr       = errors.New("add new friend error")
	irregularVerifyErr = errors.New("irregular verify error")
)

func init() {
}

func isChoice(cotent string, prefix string) bool {
	return msgutil.HasPrefix(cotent, prefix, true)
}

type autoAddNewFriend struct {
	onceDialogM      *dialog.OnceDialog
	onceDialogC      *dialog.OnceDialog
	firstRecoverFlag bool
	Cache            cacheExported `json:"auto-add-new-friend-cache"`
}

type cacheExported struct {
	IsEnable   bool   `json:"is_enable"`   // 是否启用自动通过好友请求
	IsVerify   bool   `json:"is_verify"`   // 是否开启验证
	VerifyText string `json:"verify_text"` // 被管理员控制的验证语
}

func (af *autoAddNewFriend) recoverCache() {
	// 恢复缓存
	if af.firstRecoverFlag {
		return // 只有恢复一次
	}
	af.firstRecoverFlag = true
	c := af.onceDialogM.Cache.GetPluginCacheByName(autoAddnewfriendName)
	bCache, err := json.Marshal(c)
	if err != nil {
		logging.ErrorWithErr(err, "recover auto-add-friend-cache fail")
	}
	err = json.Unmarshal(bCache, &af.Cache)
	if err != nil {
		logging.ErrorWithErr(err, "recover auto-add-friend-cache fail")
	} // 恢复缓存
}

func (af *autoAddNewFriend) SetIsEnable(b bool) {
	af.Cache.IsEnable = b
}

// SetIsVerify 设置是否开启验证
func (af *autoAddNewFriend) SetIsVerify(content string) (isReply bool, replyMsgContent string) {
	if msgutil.HasPrefix(content, "true", false) {
		af.Cache.IsVerify = true
		return true, "开启好友校验成功"
	} else if msgutil.HasPrefix(content, "false", false) {
		af.Cache.IsVerify = false
		return true, "关闭好友校验成功"
	}
	return false, ""
}

func (af *autoAddNewFriend) SetVerifyText(content string) (replyMsgContent string) {
	af.Cache.VerifyText = content
	return "设置添加好友文本规则为: " + af.Cache.VerifyText
}

func (af *autoAddNewFriend) state() string {
	var buf bytes.Buffer
	buf.WriteString("当前自动添加好友模块的状态:\n")
	buf.WriteString("================================\n")
	buf.WriteString(fmt.Sprintf("自动通过好友请求: %t\n", af.Cache.IsEnable))
	buf.WriteString(fmt.Sprintf("好友校验功能启用: %t\n", af.Cache.IsVerify))
	if af.Cache.IsVerify {
		buf.WriteString(fmt.Sprintf("验证文本规则: %s\n", af.Cache.VerifyText))
	}
	buf.WriteString("================================")
	return buf.String()
}

func (af *autoAddNewFriend) help() string {
	var buf bytes.Buffer
	buf.WriteString("自动添加好友模块手册:\t(<call>: 机器人呼唤名)\n")
	buf.WriteString("统一前缀:\t<call> " + pluginCallName + "\n")

	buf.WriteString("启用自动通过好友请求\ttrue\n")
	buf.WriteString("禁用自动通过好友请求\tfalse\n")

	buf.WriteString("启用好友规则校验\tverify true\n")
	buf.WriteString("禁用好友规则校验\tverify false\n")

	buf.WriteString("设置验证文本\tset <文本>\n")
	buf.WriteString("查看设置状态\tstate\n")

	return buf.String()
}
