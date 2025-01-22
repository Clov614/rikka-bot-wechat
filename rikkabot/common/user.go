// Package common
// @Author Clover
// @Data 2024/7/14 下午10:45:00
// @Desc 对cli的进一步封装，方便其他模块操作
package common

import (
	"context"
	"errors"
	"github.com/Clov614/logging"
	wcf "github.com/Clov614/wcf-rpc-sdk"
)

type Self struct {
	cli *wcf.Client
}

var self *Self

var (
	// ErrFriendNotFound  = errors.New("friend not found")
	// ErrGroupNotFound   = errors.New("group not found")
	ErrNotFound = errors.New("not found")
)

func GetSelf() *Self {
	return self
}

func InitSelf(ctx context.Context, cli *wcf.Client) {
	self = &Self{cli}
}

func (s *Self) GetNickName() string {
	return s.cli.GetSelfName()
}

func (s *Self) GetMemberByNickName(roomName string) {

}

func (s *Self) GetFriendIdByNickname(nickname string) (string, error) {
	friendList, err := s.cli.GetAllFriend()
	if err != nil {
		logging.ErrorWithErr(err, "GetFriendList")
		return "", ErrNotFound
	}
	for _, friend := range []*wcf.Friend(*friendList) {
		if friend.Name == nickname {
			return friend.Wxid, nil
		}
	}
	return "", ErrNotFound
}

func (s *Self) GetFriendNicknameById(wxid string) (string, error) {
	friendList, err := s.cli.GetAllFriend()
	if err != nil {
		logging.ErrorWithErr(err, "GetFriendList")
		return "", ErrNotFound
	}
	for _, friend := range []*wcf.Friend(*friendList) {
		if friend.Wxid == wxid {
			return friend.Name, nil
		}
	}
	return "", ErrNotFound
}

func (s *Self) GetGroupIdByNickname(nickname string) (string, error) {
	roomList, err := s.cli.GetAllChatRoom()
	if err != nil {
		logging.ErrorWithErr(err, "GetFriendList")
		return "", ErrNotFound
	}
	for _, room := range []*wcf.ChatRoom(*roomList) {
		if room.Name == nickname {
			return room.RoomID, nil
		}
	}
	return "", ErrNotFound
}

func (s *Self) GetGroupNicknameById(roomId string) (string, error) {
	roomList, err := s.cli.GetAllChatRoom()
	if err != nil {
		logging.ErrorWithErr(err, "GetFriendList")
		return "", ErrNotFound
	}
	for _, room := range []*wcf.ChatRoom(*roomList) {
		if room.RoomID == roomId {
			return room.Name, nil
		}
	}
	return "", ErrNotFound
}

func (s *Self) SendText(receiver string, content string, ats ...string) error {
	return s.cli.SendText(receiver, content, ats...)
}
