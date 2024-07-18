// Package common
// @Author Clover
// @Data 2024/7/14 下午10:45:00
// @Desc 对openwechat用户相关操作的进一步封装
package common

import (
	"errors"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"io"
)

type Self struct {
	self       *openwechat.Self
	Groups     *openwechat.Groups
	Friends    *openwechat.Friends
	FileHelper *openwechat.Friend
}

var self Self

var (
	ErrFriendNotFound = errors.New("friend not found")
	ErrGroupNotFound  = errors.New("group not found")
)

func GetSelf() *Self {
	return &self
}

func InitSelf(bot *openwechat.Bot) {
	bself, _ := bot.GetCurrentUser() // ignore error
	friends, _ := bself.Friends()    // ignore err
	groups, _ := bself.Groups()      // ignore err
	helper := bself.FileHelper()
	self = Self{
		self:       bself,
		Groups:     &groups,
		Friends:    &friends,
		FileHelper: helper,
	}
}

// SendText2FriendByNickname 根据好友名称发送文字
func (s *Self) SendText2FriendByNickname(nickname string, text string) error {
	// 查询好友
	results := s.Friends.SearchByNickName(1, nickname) // todo 好友可能名字重复，暂时没有好的解决方案
	friend := results.First()
	if friend == nil {
		err := errors.New("friend not found")
		return fmt.Errorf("SendText2FriendByNickname failed: %w", err)
	}
	_, err := friend.SendText(text)
	if err != nil {
		return fmt.Errorf("SendText2FriendByNickname failed: %w", err)
	}
	return nil
}

// SendImg2FriendByNickname 根据好友名称发送图片
func (s *Self) SendImg2FriendByNickname(nickname string, img io.Reader) error {
	// 查询好友
	results := s.Friends.SearchByNickName(1, nickname) // todo 好友可能名字重复，暂时没有好的解决方案
	friend := results.First()
	if friend == nil {
		err := errors.New("friend not found")
		return fmt.Errorf("SendImg2FriendByNickname failed: %w", err)
	}
	_, err := friend.SendImage(img)
	if err != nil {
		return fmt.Errorf("SendImg2FriendByNickname failed: %w", err)
	}
	return nil
}

// SendFile2FriendByNickname 根据好友名称发送文件
func (s *Self) SendFile2FriendByNickname(nickname string, file io.Reader) error {
	// 查询好友
	results := s.Friends.SearchByNickName(1, nickname) // todo 好友可能名字重复，暂时没有好的解决方案
	friend := results.First()
	if friend == nil {
		return fmt.Errorf("SendFile2FriendByNickname failed: %w", ErrFriendNotFound)
	}
	_, err := friend.SendFile(file)
	if err != nil {
		return fmt.Errorf("SendFile2FriendByNickname failed: %w", err)
	}
	return nil
}

// SendText2FriendById 根据好友id发送文字
func (s *Self) SendText2FriendById(avatarId string, text string) error {
	friend := s.Friends.SearchByID(avatarId).First()
	if friend == nil {
		return fmt.Errorf("SendText2FriendById failed: %w", ErrFriendNotFound)
	}
	_, err := friend.SendText(text)
	if err != nil {
		return fmt.Errorf("SendText2FriendById failed: %w", err)
	}
	return nil
}

// SendImg2FriendById 根据好友id发送图片
func (s *Self) SendImg2FriendById(avatarId string, img io.Reader) error {
	friend := s.Friends.SearchByID(avatarId).First()
	if friend == nil {
		return fmt.Errorf("SendImg2FriendById failed: %w", ErrFriendNotFound)
	}
	_, err := friend.SendImage(img)
	if err != nil {
		return fmt.Errorf("SendImg2FriendById failed: %w", err)
	}
	return nil
}

// SendFile2FriendByAvatarId 根据好友id发送文件
func (s *Self) SendFile2FriendByAvatarId(avatarId string, file io.Reader) error {
	friend := s.Friends.SearchByID(avatarId).First()
	if friend == nil {
		return fmt.Errorf("SendFile2FriendByAvatarId failed: %w", ErrFriendNotFound)
	}
	_, err := friend.SendFile(file)
	if err != nil {
		return fmt.Errorf("SendFile2FriendByAvatarId failed: %w", err)
	}
	return nil
}

// SendText2GroupByNickname 根据群名发送文本
func (s *Self) SendText2GroupByNickname(nickname string, text string) error {
	// 查找群组
	results := s.Groups.SearchByNickName(1, nickname)
	group := results.First()
	if group == nil {
		return fmt.Errorf("SendText2GroupByNickname failed: %w", ErrGroupNotFound)
	}
	_, err := group.SendText(text)
	if err != nil {
		return fmt.Errorf("SendText2GroupByNickname failed: %w", err)
	}
	return nil
}

// SendImg2GroupByNickname 根据群名发送图片
func (s *Self) SendImg2GroupByNickname(nickname string, img io.Reader) error {
	results := s.Groups.SearchByNickName(1, nickname)
	group := results.First()
	if group == nil {
		return fmt.Errorf("SendFile2GroupByNickname failed: %w", ErrGroupNotFound)
	}
	_, err := group.SendImage(img)
	if err != nil {
		return fmt.Errorf("SendImg2GroupByNickname failed: %w", err)
	}
	return nil
}

// SendFile2GroupByNickname 根据群名发送文件
func (s *Self) SendFile2GroupByNickname(nickname string, file io.Reader) error {
	results := s.Groups.SearchByNickName(1, nickname)
	group := results.First()
	if group == nil {
		return fmt.Errorf("SendFile2GroupByNickname failed: %w", ErrGroupNotFound)
	}
	_, err := group.SendFile(file)
	if err != nil {
		return fmt.Errorf("SendFile2GroupByNickname failed: %w", err)
	}
	return nil
}

// AddFriendInGroupByNickname 拉好友进群
func (s *Self) AddFriendInGroupByNickname(groupname string, friendname string) error {
	// 搜索群
	group := s.Groups.SearchByNickName(1, groupname).First()
	if group == nil {
		return fmt.Errorf("AddFriendInGroupByNickname failed: %w", ErrGroupNotFound)
	}
	// 搜索好友
	friend := s.Friends.SearchByNickName(1, friendname).First()
	if friend == nil {
		return fmt.Errorf("AddFriendInGroupByNickname failed: %w", ErrFriendNotFound)
	}
	err := group.AddFriendsIn(friend)
	if err != nil {
		return fmt.Errorf("AddFriendInGroupByNickname failed: %w", err)
	}
	return nil
}

// GetGroupnameList 获取所有群的群名
func (s *Self) GetGroupnameList() []string {
	groupcnt := s.Groups.Count()
	groupnames := make([]string, groupcnt)
	for i := 0; i < groupcnt; i++ {
		groupnames[i] = (*s.Groups)[i].NickName
	}
	return groupnames
}

// GetFriendsList 获取所有好友的好友名
func (s *Self) GetFriendsList() []string {
	friendcnt := s.Friends.Count()
	friendnames := make([]string, friendcnt)
	for i := 0; i < friendcnt; i++ {
		friendnames[i] = (*s.Friends)[i].NickName
	}
	return friendnames
}

// IsFriend 根据 nickname 判断是否为好友
func (s *Self) IsFriend(nickname string) bool {
	results := s.Friends.SearchByNickName(1, nickname)
	return results != nil && results.First() == nil
}

// IsGroup 根据 nickname 判断是否为已有群聊
func (s *Self) IsGroup(nickname string) bool {
	results := s.Groups.SearchByNickName(1, nickname)
	return results != nil && results.First() == nil
}

// GetNickname 获取用户名
func (s *Self) GetNickname() string {
	return s.self.NickName
}

// GetFriendIdByNickname 根据 nickname 查找出 用户id
func (s *Self) GetFriendIdByNickname(nickname string) (string, error) {
	return s.doGetIdByNickname(nickname, false)
}

// GetGroupIdByNickname 根据 nickname 查找出 id
func (s *Self) GetGroupIdByNickname(nickname string) (string, error) {
	return s.doGetIdByNickname(nickname, true)
}

func (s *Self) doGetIdByNickname(nickname string, isGroup bool) (string, error) {
	if isGroup {
		group := s.Groups.SearchByNickName(1, nickname).First()
		if group == nil {
			return "", errors.New("doGetIdByNickname failed: group not found")
		}
		return group.AvatarID(), nil
	} else {
		friend := s.Friends.SearchByNickName(1, nickname).First()
		if friend == nil {
			return "", errors.New("doGetIdByNickname failed: friend not found")
		}
		return friend.AvatarID(), nil
	}
}

// GetGroupNicknameById 根据群组id 查找出 群名
func (s *Self) GetGroupNicknameById(id string) (string, error) {
	return s.doGetNicknameById(id, true)
}

// GetFriendNicknameById 根据用户id 查找出 用户昵称
func (s *Self) GetFriendNicknameById(id string) (string, error) {
	return s.doGetNicknameById(id, false)
}

func (s *Self) doGetNicknameById(id string, isGroup bool) (string, error) {
	if isGroup {
		group := s.Groups.SearchByID(id).First()
		if group == nil {
			return "", errors.New("doGetNicknameById failed: group not found")
		}
		return group.NickName, nil
	} else {
		friend := s.Friends.SearchByID(id).First()
		if friend == nil {
			return "", errors.New("doGetNicknameById failed: friend not found")
		}
		return friend.NickName, nil
	}
}
