package adapter

import (
	"context"
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/common"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	wcf "github.com/Clov614/wcf-rpc-sdk"
	"math/rand"
	"path/filepath"
	"regexp"
	"time"
)

type Adapter struct {
	ctx      context.Context
	cli      *wcf.Client
	rikkaBot *rikkabot.RikkaBot
}

var (
	ErrNull        = errors.New("something is null")
	ErrNotGroupMsg = errors.New("not a group msg")
	ErrMetaDateNil = errors.New("meta date is nil")
	ErrRawMSgNil   = errors.New("raw message is nil")
)

func NewAdapter(ctx context.Context, cli *wcf.Client, bot *rikkabot.RikkaBot) *Adapter {
	common.InitSelf(context.TODO(), cli) // 将cli初始化至一个独立的模块便于插件的直接调用
	bot.SetSelf(common.GetSelf())

	return &Adapter{
		ctx:      ctx,
		cli:      cli,
		rikkaBot: bot,
	}
}

func (a *Adapter) HandleCovert() {
	go func() {
		for {
			select {
			case <-a.ctx.Done():
				logging.ErrorWithErr(a.ctx.Err(), "handle covert exit")
				return
			case msg := <-a.cli.GetMsgChan(): // 转换收到的消息
				logging.Debug("rikka-bot received message", map[string]interface{}{"sdk-msg": msg})
				a.receiveMsg(msg)
			}
		}
	}()

	sendChan := a.rikkaBot.GetRespMsgRecvChan()
	go func() {
		for {
			select {
			case <-a.ctx.Done():
				logging.ErrorWithErr(a.ctx.Err(), "handle send exit")
				return
			case respMsg := <-sendChan: // 接收到回复消息
				logging.Debug("rikka-bot send message", map[string]interface{}{"sdk-msg": respMsg})
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
	cli        *wcf.Client // 客户端引用
	RawMsg     *wcf.Message
	delayToken chan struct{} // 控制消息的接收与发送的随机间隔
}

func NewMetaData(cli *wcf.Client, rawMsg *wcf.Message) *MetaData {
	return &MetaData{cli: cli, RawMsg: rawMsg, delayToken: make(chan struct{})}
}

func (md *MetaData) GetRawMsg() interface{} {
	return md.RawMsg
}

// GetMsgSenderNickname 获取消息发送者昵称 test
func (md *MetaData) GetMsgSenderNickname() string {
	member, err := md.cli.GetMember(md.RawMsg.WxId)
	if err != nil {
		logging.ErrorWithErr(err, "GetMsgSenderNickName fail")
	}
	if 0 == len(member) {
		logging.WarnWithErr(ErrNull, "GetMsgSenderNickname fail")
		return ""
	} else if nil == member[0] {
		logging.WarnWithErr(ErrNull, "GetMsgSenderNickname fail")
		return ""
	}
	return member[0].NickName
}

// GetGroupNickname 获取群组消息的群名 test
func (md *MetaData) GetGroupNickname() string {
	if !md.RawMsg.IsGroup { // 不是群组直接返回空
		return ""
	}
	member, err := md.cli.GetMember(md.RawMsg.RoomId)
	if err != nil || 0 == len(member) {
		logging.WarnWithErr(ErrNull, "GetMsgGroupNickname fail")
		return ""
	} else if nil == member[0] {
		logging.WarnWithErr(ErrNull, "GetMsgGroupNickname fail")
		return ""
	}
	return member[0].NickName
}

// GetRoomNameByRoomId 根据RoomId 获得群名 test
func (md *MetaData) GetRoomNameByRoomId(id string) (string, error) {
	member, err := md.cli.GetMember(id)
	if err != nil {
	}
	if 0 == len(member) {
		return "", ErrNull
	} else if nil == member[0] {
		return "", ErrNull
	}
	return member[0].NickName, nil
}

// GetImgData 获取图片数据
func (md *MetaData) GetImgData() []byte {
	err := md.RawMsg.FileInfo.DecryptImg()
	if err != nil {
		logging.ErrorWithErr(err, "GetImgData fail")
		return nil
	}
	return md.RawMsg.FileInfo.Data
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
	var chatImgUrl string
	switch msg.Type {
	case wcf.MsgTypeText | wcf.MsgTypeXMLQuote:
		rikkaMsgType = message.MsgTypeText
	case wcf.MsgTypeImage:
		rikkaMsgType = message.MsgTypeImage
		if msg.FileInfo != nil {
			chatImgUrl = generateChatImageWebURL(msg.FileInfo.ExtractRelativePath()) // 生成chatImgUrl
		}
	case wcf.MsgTypeVoice:
		rikkaMsgType = message.MsgTypeVoice
	case wcf.MsgTypeVideo:
		rikkaMsgType = message.MsgTypeVideo
	case wcf.MsgTypeXML: // todo test 解析app消息
		//if msg.AppMsgType == openwechat.AppMsgTypeVideo { // 视频 app 消息
		//	rikkaMsgType = message.MsgTypeApp
		//} else { // todo 消息选择器测试无误后移除
		//	return nil // 忽略未知app消息
		//}
	default:
		return nil // 忽略未知的消息种类
	}

	metaData := NewMetaData(a.cli, msg)
	cfg := config.GetConfig()
	go metaData.runDelayTimer(cfg.AnswerDelayRandMin, cfg.AnswerDelayRandMax) // 消息随机延迟

	//rself := common.GetSelf() // 获取rikka的self对象
	var isAtMe bool
	var AtNameList []string
	var AtWxidList []string
	if msg.IsGroup {
		// 获取消息中艾特成员的成员名
		re := regexp.MustCompile(`@([^\s]+?) `)
		matches := re.FindAllStringSubmatch(msg.Content, -1)
		AtNameList = make([]string, len(matches))
		AtWxidList = make([]string, len(matches))

		for i, match := range matches {
			if len(match) > 1 {
				AtNameList[i] = match[1]
				infos, err := msg.RoomData.GetMembersByNickName(match[1])
				if err != nil {
					logging.WarnWithErr(err, "RoomData.GetMembersByNickName fail")
				} else if len(infos) != 0 || infos[0] != nil || infos[0].Wxid != "" {
					AtWxidList[i] = infos[0].Wxid
				}

				isAtMe = match[1] == a.cli.GetSelfInfo().Name // 是否艾特自己
			}
		}
	}
	return &message.Message{
		Msgtype:         rikkaMsgType,
		MetaData:        metaData,
		RawContent:      msg.Content,
		ChatImgUrl:      chatImgUrl, // 图片url
		Content:         msg.Content,
		MsgId:           msg.MessageId,
		WxId:            msg.WxId,
		RoomId:          msg.RoomId,
		GroupName:       metaData.GetGroupNickname(),
		GroupAtNameList: AtNameList,
		GroupAtWxIdList: AtWxidList,
		IsAtMe:          isAtMe,
		IsGroup:         msg.IsGroup,
		IsFriend:        msg.IsSendByFriend(),
		IsMySelf:        msg.IsSelf, // 是否为自己发送的消息
	}
}

func generateChatImageWebURL(suffixPath string) string {
	return filepath.ToSlash(filepath.Join("/chat_image", suffixPath))
}

// @Author By Clover 2024/7/5 下午5:28:00
// @Reason 处理外部平台消息，转为自身消息
// @Demand Version
func (a *Adapter) receiveMsg(msg *wcf.Message) {
	selfMsg := a.covert(msg)
	logging.Debug("adapter.receiveMsg", map[string]interface{}{"covertedMsg": fmt.Sprintf("%+v", selfMsg)})
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
