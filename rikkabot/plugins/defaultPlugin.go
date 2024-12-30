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
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/message"
	_ "wechat-demo/rikkabot/plugins/admin"         // 需要副作用 init注册方法
	_ "wechat-demo/rikkabot/plugins/ai"            // 需要副作用 init注册方法
	_ "wechat-demo/rikkabot/plugins/biliUrlDecode" // 需要副作用 init注册方法
	_ "wechat-demo/rikkabot/plugins/game"          // 需要副作用 init注册方法
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/control/dialog"
	"wechat-demo/rikkabot/processor/register"
	"wechat-demo/rikkabot/utils/msgutil"
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

	// core
	coreRules := &control.ProcessRules{EnableMsgType: []message.MsgType{message.MsgTypeNewFriendVerify}}

	// manager
	rules := &control.ProcessRules{IsCallMe: true, IsAdmin: true, IsAtMe: true, EnableGroup: true, EnableMsgType: []message.MsgType{message.MsgTypeText}}
	aanf := autoAddNewFriend{
		onceDialogM: dialog.InitOnceDialog("自动添加好友管理", rules),
		onceDialogC: dialog.InitOnceDialog("自动添加好友核心", coreRules),
	}
	aanf.onceDialogC.SetOnceFunc(func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		aanf.recoverCache()               // 恢复缓存
		err := aanf.addNewFriend(recvmsg) // 处理添加好友消息
		if err != nil {
			logging.ErrorWithErr(err, autoAddnewfriendCore)
		}
	})

	aanf.onceDialogM.SetOnceFunc(func(recvmsg message.Message, sendMsg chan<- *message.Message) {
		aanf.recoverCache() // 恢复缓存
		isOrder, _ := msgutil.IsOrder([]string{pluginCallName}, recvmsg.Content)
		if !isOrder { // 命令校验//
			return
		}
		trimedContent := msgutil.TrimPrefix(recvmsg.Content, pluginCallName, false, true)
		switch true {
		case isChoice(trimedContent, "help"): // 帮助信息
			aanf.onceDialogM.SendText(recvmsg.MetaData, aanf.help())
		case isChoice(trimedContent, "true"):
			aanf.SetIsEnable(true)
			aanf.onceDialogM.SendText(recvmsg.MetaData, "开启自动通过好友请求成功")
		case isChoice(trimedContent, "false"):
			aanf.SetIsEnable(false)
			aanf.onceDialogM.SendText(recvmsg.MetaData, "关闭自动通过好友请求成功")
		case isChoice(trimedContent, "verify"):
			bStr := msgutil.TrimPrefix(trimedContent, "verify", false, true)
			b, content := aanf.SetIsVerify(bStr) // 根据文本设置是否开启文本校验
			if b {
				aanf.onceDialogM.SendText(recvmsg.MetaData, content)
			}
		case isChoice(trimedContent, "set"): // 设置校验文本
			verifyText := msgutil.TrimPrefix(trimedContent, "set", false, true)
			if verifyText != "" {
				aanf.onceDialogM.SendText(recvmsg.MetaData, aanf.SetVerifyText(verifyText))
			}
		case isChoice(trimedContent, "state"):
			aanf.onceDialogM.SendText(recvmsg.MetaData, aanf.state())
		}
		// 更新缓存
		aanf.onceDialogM.Cache.UploadCacheByPluginName(autoAddnewfriendName, aanf.Cache)
	})
	register.RegistPlugin(autoAddnewfriendCore, aanf.onceDialogC, 0)
	register.RegistPlugin(autoAddnewfriendName, aanf.onceDialogM, 1)
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
	err = json.Unmarshal(bCache, &af.Cache)
	if err != nil {
		logging.ErrorWithErr(err, "recover auto-add-friend-cache fail")
	} // 恢复缓存
}

func (af *autoAddNewFriend) addNewFriend(msg message.Message) error {
	if !af.Cache.IsEnable { // 不自动添加好友
		return nil
	}
	if af.Cache.IsVerify {
		if !msgutil.HasPrefix(msg.Content, af.Cache.VerifyText, false) { // 不符合规则直接返回
			return irregularVerifyErr
		}
	}
	if !msg.MetaData.AgreeNewFriend() {
		return addFriendErr
	}
	af.onceDialogM.Self.UpdateFriends() // 更新好友列表
	return nil
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
