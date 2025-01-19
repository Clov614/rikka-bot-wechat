package adapter

import (
	"context"
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/manager"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	wcf "github.com/Clov614/wcf-rpc-sdk"
	"math/rand"
	"regexp"
	"time"
)

type Adapter struct {
	ctx      context.Context
	cli      *wcf.Client
	rikkaBot *rikkabot.RikkaBot
}

var (
	ErrNotGroupMsg = errors.New("not a group msg")
	ErrMetaDateNil = errors.New("meta date is nil")
	ErrRawMSgNil   = errors.New("raw message is nil")
)

func NewAdapter(ctx context.Context, cli *wcf.Client, bot *rikkabot.RikkaBot) *Adapter {
	//common.InitSelf(openwcBot) // 初始化 该用户数据（朋友、群组） todo refactor
	//selfBot.SetSelf(common.GetSelf()) todo refactor

	return &Adapter{
		ctx:      ctx,
		cli:      cli,
		rikkaBot: bot,
	}
}

func (a *Adapter) HandleCovert() {
	sendChan := a.rikkaBot.GetRespMsgRecvChan()
	go func() {
		for {
			select {
			case <-a.ctx.Done():
				logging.ErrorWithErr(a.ctx.Err(), "handle covert exit")
				return
			case msg := <-a.cli.GetMsgChan(): // 转换收到的消息
				a.receiveMsg(msg)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-a.ctx.Done():
				logging.ErrorWithErr(a.ctx.Err(), "handle send exit")
				return
			case respMsg := <-sendChan:
				//rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				//time.Sleep(time.Duration((rnd.Intn(1000) + 1000)) * time.Millisecond)
				err := a.sendMsg(respMsg)
				if err != nil {
					logging.ErrorWithErr(err, "sendMsg fail skip send")
				}
			}
		}
	}()
}

//<editor-fold desc="MetaData">

// MetaData message.IMeta impl
type MetaData struct {
	RawMsg *wcf.Message
	//GroupMember openwechat.Members // todo 群组成员（群组消息才会有） RoomMember
	delayToken chan struct{} // 控制消息的接收与发送的随机间隔
}

func NewMetaData(rawMsg *wcf.Message) *MetaData {
	return &MetaData{RawMsg: rawMsg, delayToken: make(chan struct{})}
}

func (md *MetaData) GetRawMsg() interface{} {
	return md.RawMsg
}

// GetMsgSenderNickname 获取消息发送者昵称 todo
func (md *MetaData) GetMsgSenderNickname() string {
	return ""
}

// GetGroupNickname 获取群组消息的群名
func (md *MetaData) GetGroupNickname() string {
	// todo
	return ""
}

// GetRoomNameByRoomId 根据RoomId 获得群名
func (md *MetaData) GetRoomNameByRoomId(id string) (string, error) {
	// todo
	return "", nil
}

func (md *MetaData) runDelayTimer(delayMin int, delayMax int) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(rnd.Intn(1000*delayMax-1000*delayMin)+1000*delayMin) * time.Millisecond)
	close(md.delayToken)
}

//</editor-fold>

// covert 消息转换处理
func (a *Adapter) covert(msg *wcf.Message) *message.Message {
	var rikkaMsgType message.MsgType
	switch msg.Type {
	case uint32(wcf.MsgTypeText):
		rikkaMsgType = message.MsgTypeText
	case uint32(wcf.MsgTypeImage):
		rikkaMsgType = message.MsgTypeImage
	case uint32(wcf.MsgTypeVoice):
		rikkaMsgType = message.MsgTypeVoice
	case uint32(wcf.MsgTypeVideo):
		rikkaMsgType = message.MsgTypeVideo
	case uint32(wcf.MsgTypeShare): // todo test 解析app消息
		//if msg.AppMsgType == openwechat.AppMsgTypeVideo { // 视频 app 消息
		//	rikkaMsgType = message.MsgTypeApp
		//} else { // todo 消息选择器测试无误后移除
		//	return nil // 忽略未知app消息
		//}
	default:
		return nil // 忽略未知的消息种类
	}

	metaData := NewMetaData(msg)
	cfg := config.GetConfig()
	go metaData.runDelayTimer(cfg.AnswerDelayRandMin, cfg.AnswerDelayRandMax) // 消息随机延迟

	//rself := common.GetSelf() // 获取rikka的self对象

	// 获取消息中艾特成员的成员名
	re := regexp.MustCompile(`@([^\s]+) `)
	matches := re.FindAllStringSubmatch(msg.Content, -1)
	groupAtNameList := make([]string, len(matches))
	var isAtMe bool
	for i, match := range matches {
		if len(match) > 1 {
			groupAtNameList[i] = match[1]
			isAtMe = match[1] == a.cli.GetSelfInfo().Name // 是否艾特自己
		}
	}
	return &message.Message{
		Msgtype:    rikkaMsgType,
		MetaData:   metaData,
		RawContent: msg.Content,
		//ChatImgUrl:      cacheImgCovert2Url(imgData, uuid), // 图片url
		Content: msg.Content,
		MsgId:   msg.MessageId,
		WxId:    msg.WxId,
		RoomId:  msg.RoomId,
		//GroupName: todo
		IsAtMe:  isAtMe,
		IsGroup: msg.IsGroup,
		//IsFriend:        msg.IsSendByFriend(), todo
		IsMySelf: msg.IsSelf, // 是否为自己发送的消息
	}
}

func cacheImgCovert2Url(data []byte, uuid string) string {
	if data == nil || len(data) == 0 {
		return ""
	}
	imgName, nowDate := manager.SaveImg(uuid, data)
	// 拼装返回url
	imgUrl := "/chat_image/" + nowDate + "/" + imgName
	return imgUrl
}

// @Author By Clover 2024/7/5 下午5:28:00
// @Reason 处理外部平台消息，转为自身消息
// @Demand Version
func (a *Adapter) receiveMsg(msg *wcf.Message) {
	selfMsg := a.covert(msg)
	if selfMsg == nil {
		return
	}
	copyMsg := *selfMsg
	a.rikkaBot.DispatchMsgEvent(copyMsg) // 存入事件池
	if a.rikkaBot.EnableProcess {        // 判断是否启动了处理器（防止没有消费者阻塞在此）
		a.rikkaBot.GetReqMsgSendChan() <- selfMsg
	}
}

func (a *Adapter) sendMsg(sendMsg *message.Message) error {
	if sendMsg.MetaData == nil {
		logging.Debug("MetaData is nil", map[string]interface{}{"sendMsg": sendMsg})
		return fmt.Errorf("can't send msg, sendMsg err: %w", ErrMetaDateNil)
	}
	<-sendMsg.MetaData.(*MetaData).delayToken // 需要延迟随机时间后，才能发送消息
	rawMsg, ok := sendMsg.MetaData.GetRawMsg().(*wcf.Message)
	if !ok {
		logging.Debug("get metaData.rawMsg failed", map[string]interface{}{"sendMsg": sendMsg})
		return fmt.Errorf("get metaData.rawMsg failed, err: %w", ErrRawMSgNil)
	}
	switch sendMsg.Msgtype {
	case message.MsgTypeText:
		err := rawMsg.ReplyText(sendMsg.Content)
		if err != nil {
			logging.ErrorWithErr(err, "SendMsg fail")
		}
	//case message.MsgTypeImage:
	//	replyImage, err := rawMsg.ReplyImage(bytes.NewReader(sendMsg.Raw))
	//	if err != nil {
	//		logging.ErrorWithErr(err, "SendMsg fail", map[string]interface{}{"replyImage": replyImage})
	//	}
	default:
		logging.Warn("unknown msgType do not handle send")
	}
	return nil
}
