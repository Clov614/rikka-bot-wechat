// Package plugin_game
// @Author Clover
// @Data 2024/8/1 下午10:52:00
// @Desc 默认游戏模块
package plugin_game

import (
	"errors"
	"fmt"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/control/dialog"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/register"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/msgutil"
	"github.com/rs/zerolog/log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var (
	errGetSettings = errors.New("get russianRoulettePlugin settings in msg err")
	errParams      = errors.New("get russianRoulettePlugin params in msg err")
)

func init() {
	rrPlugin := russianRoulettePlugin{
		LongDialog: dialog.InitLongDialog("游戏-俄罗斯轮盘", &control.ProcessRules{IsAtMe: true, IsCallMe: true, EnableGroup: true,
			ExecOrder: []string{"俄罗斯轮盘", "轮盘游戏", "Russian Roulette"}, EnableMsgType: []message.MsgType{message.MsgTypeText}}),
	}
	// 运行时逻辑
	rrPlugin.SetLongFunc(func(firstMsg message.Message, recvMsg <-chan message.Message, sendMsg chan<- *message.Message) {
		if firstMsg.Content == "help" {
			rrPlugin.roomId = firstMsg.RoomId
			rrPlugin.sendText(firstMsg.RoomId, rrPlugin.getHelp()) // 发送帮助信息
			return
		}
		rrPlugin.startGame(firstMsg) // 开始游戏
	})

	register.RegistPlugin("Russian Roulette", rrPlugin.LongDialog, 2) // 注册游戏
}

type russianRoulettePlugin struct {
	// todo 设计比分查询
	*dialog.LongDialog
	Initiator      string // 游戏发起者
	Challenger     string // 挑战者
	Winer          string // 获胜者
	Trgger         string // 每回合扣扳机者
	BulletRoulette []int  // 子弹轮盘
	setting        tsettings
	roomId         string // 群id
}

const (
	MaxBulletsNum  = 6  // 最大子弹数量
	MaxBulletsSlot = 12 // 最大子弹槽数
)

type tsettings struct {
	BulletsNum  int // 子弹随机装入的数量
	BulletsSlot int // 子弹槽数
}

func (rrp *russianRoulettePlugin) startGame(firstMsg message.Message) {
	// 获取群id
	rrp.roomId = firstMsg.RoomId
	// 初始化游戏
	err := rrp.InitGame(firstMsg.Content, firstMsg.SenderName)
	if err != nil {
		log.Error().Err(err).Msg("初始化’俄罗斯轮盘‘游戏错误")
		rrp.sendText(firstMsg.RoomId, "初始化’俄罗斯轮盘‘游戏错误: "+err.Error())
		return
	}
	rrp.initGameData() // 初始化游戏数据

	rrp.sendText(fmt.Sprintf("开始游戏，对手是: %s 请在30s内接受挑战，艾特我回复‘接受’即开始游戏", msgutil.AtSomeOne(rrp.Challenger)))
	isAccept := rrp.waitChallengerAccept() // 等待对手接受
	if !isAccept {
		rrp.sendText(fmt.Sprintf("对手 %s 未能在30秒内接受挑战", rrp.Challenger))
		return
	}
	// 接受了挑战，开始遍历子弹
	var endFlag bool
	for i, bullet := range rrp.BulletRoulette {
		if i%2 == 0 { // 挑战者先开始
			endFlag = rrp.doChallengerRound(bullet)
		} else {
			endFlag = rrp.doInitiatorRound(bullet)
		}
		if endFlag {
			break
		}
	}
	// 结束游戏汇报情况
	res := rrp.reportResult()
	rrp.sendText(res)
}

// InitGame 初始化游戏
func (rrp *russianRoulettePlugin) InitGame(startText string, senderUserName string) error {
	var err error
	err = rrp.getTheSettings(startText)
	if err != nil {
		return fmt.Errorf("初始化游戏错误: %w", err)
	}
	rrp.initGameData()
	rrp.Initiator = senderUserName // 发起者id
	return nil
}

// initGameData 初始化游戏数据
func (rrp *russianRoulettePlugin) initGameData() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 初始化插槽
	rrp.BulletRoulette = make([]int, rrp.setting.BulletsSlot)

	// 使用map来确保子弹位置唯一
	bulletPositions := make(map[int]bool)

	// 随机选择子弹位置
	for len(bulletPositions) < rrp.setting.BulletsNum {
		pos := rng.Intn(rrp.setting.BulletsSlot) // 使用[0, BulletsSlot)的范围
		if !bulletPositions[pos] {               // 检查该位置是否已经有子弹
			bulletPositions[pos] = true // 标记该位置已被使用
			rrp.BulletRoulette[pos] = 1 // 在该位置放置子弹
		}
	}
}

// getTheSettings 读取发起用户设置
func (rrp *russianRoulettePlugin) getTheSettings(text string) error {
	var err error
	var settings tsettings
	spled := strings.Split(text, " ")
	bnum, err := strconv.Atoi(spled[0])
	if err != nil {
		return fmt.Errorf("get bnum err %w %w", errParams, err)
	}
	bslot, err := strconv.Atoi(spled[1])
	if err != nil {
		return fmt.Errorf("get bslot err %w %w", errParams, err)
	}
	challengerATName := spled[2]
	// 读取挑战者名称
	if !strings.HasPrefix(challengerATName, "@") {
		return fmt.Errorf("%w 参数不对，第三个参数为对手艾特，请使用\"<游戏名> help\" 查看帮助信息", errParams)
	}
	// 读取挑战者名称
	rrp.Challenger = msgutil.GetNicknameByAt(challengerATName)
	if len(spled) != 3 {
		return fmt.Errorf("%w 参数不对，请使用\"<游戏名> help\" 查看帮助信息", errParams)
	}
	if err != nil {
		return fmt.Errorf("get russianRoulettePlugin settings err: %w", err)
	}

	if bslot <= 0 || bslot > MaxBulletsSlot {
		return fmt.Errorf("(第二个参数)插槽参数异常 需要提供大于子弹数的正数以及不超过最大插槽数%d err: %w", MaxBulletsSlot, errGetSettings)
	}
	if bnum <= 0 || bnum > MaxBulletsNum {
		return fmt.Errorf("(第一个参数)子弹数参数异常 需要提供小于插槽数以及不超过最大子弹数%d err: %w", MaxBulletsNum, errGetSettings)
	}
	if bnum >= bslot {
		return fmt.Errorf("子弹数必须小于插槽数 %w", errGetSettings)
	}
	settings.BulletsNum = bnum
	settings.BulletsSlot = bslot
	rrp.setting = settings
	return nil
}

// waitChallengerAccept 等待挑战者接收挑战
func (rrp *russianRoulettePlugin) waitChallengerAccept() bool {
	var done = make(chan struct{})
	go func() {
		timer := time.NewTimer(time.Second * 30)
		<-timer.C
		close(done)
	}()
	_, b, _ := rrp.RecvMessage(&control.ProcessRules{IsAtMe: true, IsCallMe: true, EnableGroup: true,
		ExecOrder: []string{"接受", "接受挑战", "accept", "ok", "go", "fine"}, CostomTrigger: func(rikkaMsg message.Message) bool {
			return rikkaMsg.SenderName == rrp.Challenger
		}}, done)

	return b
}

var (
	execShotself       = []string{"自己", "射自己", "对自己开枪", "shot me"}
	execShotOpponent   = []string{"对手", "射对手", "对对手开枪", "shot her", "shot him"}
	joinedShotself     = strings.Join(execShotself, ",")
	joinedShotOpponent = strings.Join(execShotOpponent, ",")
)

// doChallengerRound 挑战者回合
func (rrp *russianRoulettePlugin) doChallengerRound(bullet int) (isEnd bool) {
	return rrp.doPlayerRound(bullet, rrp.Challenger, rrp.Initiator)
}

// doInitiatorRound 发起者回合
func (rrp *russianRoulettePlugin) doInitiatorRound(bullet int) (isEnd bool) {
	return rrp.doPlayerRound(bullet, rrp.Initiator, rrp.Challenger)
}

// doPlayerRound 玩家回合处理
func (rrp *russianRoulettePlugin) doPlayerRound(bullet int, player1 string, player2 string) (isEnd bool) {
	atText := msgutil.AtSomeOne(player1)
	rrp.sendText(atText + "你的回合！（可选择: ’射自己‘ 或 ‘射对手’）")
	done := make(chan struct{})
	_, b, order := rrp.RecvMessage(&control.ProcessRules{IsAtMe: true, IsCallMe: true, EnableGroup: true,
		ExecOrder: append(execShotself, execShotOpponent...), CostomTrigger: func(rikkaMsg message.Message) bool {
			return rikkaMsg.SenderName == player1
		}}, done)
	if !b { // 是否超时
		rrp.sendText(atText + "30s未作选择，视为弃权")
		rrp.Winer = player2 // 发起者胜出
		return true
	}
	// 判断指令
	if strings.Contains(joinedShotself, order) {
		// 射自己
		rrp.Trgger = player1
		if rrp.isbulletfatal(bullet) {
			rrp.Winer = player2
			rrp.sendText(rrp.Trgger + " 扣下了扳机，bang！很遗憾它带了！！")
			return true
		}
	} else if strings.Contains(joinedShotOpponent, order) {
		// 射对手
		rrp.Trgger = player2
		if rrp.isbulletfatal(bullet) {
			rrp.Winer = player1
			rrp.sendText(rrp.Trgger + " 扣下了扳机，bang！很遗憾它带了！！")
			return true
		}
	} else {
		rrp.sendText("请选择 ‘自己’ 或 ‘对手’，例：’<艾特机器人> 自己‘")
		return rrp.doChallengerRound(bullet) // 递归执行
	}
	rrp.sendText(rrp.Trgger + " 扣下了扳机，ding！并没有开火，它幸存了下来")
	return false
}

// isbulletfatal 是否致命
func (rrp *russianRoulettePlugin) isbulletfatal(bullet int) bool {
	return bullet == 1
}

// 报告结果
func (rrp *russianRoulettePlugin) reportResult() (res string) {
	rrp.MsgBuf.WriteString("游戏结束\n")
	rrp.MsgBuf.WriteString("本局装弹情况: ")
	for i, bullet := range rrp.BulletRoulette {
		symbol := ""
		if bullet == 1 {
			symbol = "1"
		} else {
			symbol = "-"
		}
		rrp.MsgBuf.WriteString(symbol)
		if i != len(rrp.BulletRoulette)-1 {
			rrp.MsgBuf.WriteString(" ")
		}
	}
	rrp.MsgBuf.WriteString("\n本局获胜者: " + rrp.Winer)
	res = rrp.MsgBuf.String()
	rrp.MsgBuf.Truncate(0) // 清空字符串缓存
	return res
}

//func (rrp *russianRoulettePlugin) doTimer(done chan struct{}) {
//	timer := time.NewTimer(time.Second * 30) // 30秒超时
//	go func() {
//		<-timer.C
//		close(done)
//	}()
//}

func (rrp *russianRoulettePlugin) sendText(receiver string, text string, ats ...string) {
	var err error
	err = rrp.Self.SendText(receiver, text, ats...)
	if err != nil {
		log.Warn().Err(err).Msg("俄罗斯轮盘游戏发送消息失败")
	}
}

// getHelp 获取游戏帮助信息
func (rrp *russianRoulettePlugin) getHelp() string {
	var help strings.Builder

	help.WriteString("🎯 俄罗斯轮盘游戏帮助 🎯\n\n")

	help.WriteString("游戏规则:\n")
	help.WriteString("1. 两名玩家轮流选择是否射击自己或对手，直到子弹射出。\n")
	help.WriteString(fmt.Sprintf("2. 子弹槽最大数量为 %d，子弹最大数量为 %d。\n", MaxBulletsSlot, MaxBulletsNum))
	help.WriteString("3. 发起者需设置子弹数、槽数，并 @对手。\n")
	help.WriteString("4. 挑战者有30秒时间接受挑战，否则游戏取消。\n")
	help.WriteString("5. 如果子弹在扣扳机时射出，射手失败，对手获胜。\n\n")

	help.WriteString("命令格式:\n")
	help.WriteString("启动游戏: '<游戏名> <子弹数> <槽数> <@对手>'\n")
	help.WriteString("接受挑战: '@<机器人名> 接受挑战'\n")
	help.WriteString("射击自己: '@<机器人名> 射自己'\n")
	help.WriteString("射击对手: '@<机器人名> 射对手'\n\n")

	help.WriteString("示例:\n")
	help.WriteString("发起挑战: '俄罗斯轮盘 2 6 @李四'\n")
	help.WriteString("接受挑战: '@RikkaBot 接受挑战'\n")

	return help.String()
}
