package coreapi

import (
	"errors"
	"fmt"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/groupmanager"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	wcf "github.com/Clov614/wcf-rpc-sdk"
)

var core *Core

var (
	ErrInvalidCall = errors.New("invalid bot call")
	ErrSendMsg     = errors.New("send message error")
	ErrFetchImg    = errors.New("fetch image error")
	ErrUnSupport   = errors.New("unsupported this func")
)

type Core struct {
	cli          *wcf.Client // hook sdk
	GroupManager groupmanager.IGroupManager
}

func NewCore(cli *wcf.Client, groupManager groupmanager.IGroupManager) *Core {
	InitCore(cli, groupManager)
	return core
}

func InitCore(cli *wcf.Client, groupManager groupmanager.IGroupManager) {
	core.cli = cli
	core.GroupManager = groupManager
}

func GetCore() *Core {
	if core.GroupManager == nil || core.cli == nil {
		return nil
	}
	return core
}

// core-function

func (c *Core) GetFullFilePathFromRelativePath(relativePath string) string {
	return c.cli.GetFullFilePathFromRelativePath(relativePath)
}

func (c *Core) GetImgDataByPath(path string) []byte {
	return c.cli.DecodeDatFileToBytes(path)
}

// SendMsg 统一发送消息接口 消息类型 是否群组 发送数据 群/好友 id
// nolint
func (c *Core) SendMsg(msgType message.MsgType, data any, sendId string) error {
	var err error
	switch msgType {
	case message.MsgTypeText:
		text, ok := data.(string)
		if !ok {
			return fmt.Errorf("`SendMsg of text` must be a string: %w", ErrSendMsg)
		}
		err = c.cli.SendText(sendId, text) // 发送消息
		if err != nil {
			return fmt.Errorf("send text to %s error: %w", sendId, err)
		}
	case message.MsgTypeImage:
		src, ok := data.(string)
		if !ok {
			return fmt.Errorf("`SendMsg of image` must be a string(src:<ImgPath or URL>): %w", ErrSendMsg)
		}
		err = c.cli.SendImage(sendId, src) // todo wcf支持直接传递图片数据（非本机无法发送图片问题）（不成立）
		if err != nil {
			return fmt.Errorf("send image to %s error: %w", sendId, err)
		}
	default:
		return fmt.Errorf("`SendMsg of type` must be either text or image: %w", ErrSendMsg)
	}
	return nil
}

// AcceptNewFriend 同意好友请求
func (c *Core) AcceptNewFriend(req wcf.NewFriendReq) bool {
	return c.cli.AcceptNewFriend(req)
}

// GetCtFriends 获取通讯录好友列表
func (c *Core) GetCtFriends() ([]wcf.Friend, bool) {
	friends, err := c.cli.CtFriends()
	if err != nil {
		return nil, false
	}
	return friends, true
}

// GetCtChatRooms 获取通讯录群聊列表
func (c *Core) GetCtChatRooms() ([]wcf.ChatRoom, bool) {
	rooms, err := c.cli.CtChatRooms()
	if err != nil {
		return nil, false
	}
	return rooms, true
}

// GetCtGHs 获取通讯录公众号列表
func (c *Core) GetCtGHs() ([]wcf.GH, bool) {
	hs, err := c.cli.CtGHs()
	if err != nil {
		return nil, false
	}
	return hs, true
}

func (c *Core) Close() {
	c.cli.Close()
}

func init() {
	core = &Core{}
}
