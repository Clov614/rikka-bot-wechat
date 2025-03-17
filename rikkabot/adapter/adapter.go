package adapter

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/testutil"
	wcf "github.com/Clov614/wcf-rpc-sdk"
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
	member := md.cli.GetMember(md.RawMsg.WxId, true)
	return member.NickName
}

// GetMsgSenderAlias 获取群成员昵称 test
func (md *MetaData) GetMsgSenderAlias() string {
	return "" // todo
}

// GetGroupNickname 获取群组消息的群名 test
func (md *MetaData) GetGroupNickname() string {
	if !md.RawMsg.IsGroup { // 不是群组直接返回空
		return ""
	}
	room := md.cli.GetMember(md.RawMsg.RoomId, true) // todo 【优化】不通过Member查询，通过Contact查群聊信息
	return room.NickName
}

// GetRoomNameByRoomId 根据RoomId 获得群名 test
func (md *MetaData) GetRoomNameByRoomId(id string) (string, error) {
	member := md.cli.GetMember(id, true)
	if member == nil || member.NickName == "" {
		return "", ErrNotGroupMsg
	}
	return member.NickName, nil
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

var ignoreGHOnce sync.Once

// covert 消息转换处理
func (a *Adapter) covert(msg *wcf.Message) *message.Message {
	if msg.IsGH { // 忽略公众号消息
		ignoreGHOnce.Do(func() { logging.Warn("!!注意：此版本框架自动忽略了公众号的消息！~") })
		logging.Debug("!!注意：此版本框架自动忽略了公众号的消息！~", map[string]interface{}{"msg": msg})
		return nil // 跳过处理公众号消息 todo 处理公众号消息
	}
	var rikkaMsgType message.MsgType
	var chatImgUrl string
	switch msg.Type {
	case wcf.MsgTypeText:
		fallthrough
	case wcf.MsgTypeXMLQuote:
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
	return &message.Message{
		Msgtype:    rikkaMsgType,
		MetaData:   metaData,
		RawContent: msg.Content,
		ChatImgUrl: chatImgUrl, // 图片url
		Content:    msg.Content,
		MsgId:      msg.MessageId,
		WxId:       msg.WxId,
		RoomId:     msg.RoomId,
		RoomName:   metaData.GetGroupNickname(),
		RoomAts:    msg.RoomData.AtedMSequence,
		IsAtMe:     msg.RoomData.IsAtSelf,
		IsGroup:    msg.IsGroup,
		IsFriend:   msg.IsSendByFriend(),
		IsMySelf:   msg.IsSelf, // 是否为自己发送的消息
		FileInfo:   msg.FileInfo,
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
	// 测试插件
	if config.GetConfig().EnableTestPlugin {
		err := testutil.SaveTestMessage(msg)
		if err != nil {
			logging.ErrorWithErr(err, "SaveTestMessage error")
		}
	}
}

func (a *Adapter) sendMsg(sendMsg *message.Message) error {
	if sendMsg == nil {
		return fmt.Errorf("sendMsg is nil")
	}
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
	case message.MsgTypeImage:
		if nil == sendMsg.FileInfo {
			return fmt.Errorf("send img err, FileInfo is: %w", ErrNull)
		}
		err := rawMsg.ReplyImage(sendMsg.FileInfo.FilePath)
		if err != nil {
			logging.ErrorWithErr(err, "SendMsg fail", map[string]interface{}{"filePath": sendMsg.FileInfo.FilePath})
		}
	default:
		logging.Warn("unknown msgType do not handle send")
	}
	return nil
}
