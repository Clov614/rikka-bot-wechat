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
	"sync"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/utils/secretutil"
)

type Self struct {
	self    *openwechat.Self
	Groups  openwechat.Groups
	Friends openwechat.Friends
	MyGroups
	MyFriends
	FileHelper *openwechat.Friend
	*UidDict
	mu sync.RWMutex
}

type MyGroups []*openwechat.User
type MyFriends []*openwechat.User

type UidDict struct {
	UidGroupDict       map[string]*openwechat.Group
	UidFriendDict      map[string]*openwechat.Friend
	UidGroupNotUnique  map[string]bool
	UidFriendNotUnique map[string]bool
}

var self *Self

var (
	ErrFriendNotFound  = errors.New("friend not found")
	ErrGroupNotFound   = errors.New("group not found")
	ErrGroupNotUnique  = errors.New("group not unique")
	ErrFriendNotUnique = errors.New("friend not unique")
)

func GetSelf() *Self {
	return self
}

func InitSelf(bot *openwechat.Bot) {
	bself, _ := bot.GetCurrentUser() // ignore error
	friends, _ := bself.Friends()    // ignore err
	groups, _ := bself.Groups()      // ignore err
	helper := bself.FileHelper()
	self = &Self{
		self:       bself,
		Groups:     groups,
		Friends:    friends,
		FileHelper: helper,
		UidDict: &UidDict{
			UidGroupDict:  make(map[string]*openwechat.Group, groups.Count()),
			UidFriendDict: make(map[string]*openwechat.Friend, friends.Count())},
	}
	// 初始化备份好友群聊
	self.updateMyGroups()
	self.updateMyFriends()
	// 初始化 uid 缓存
	self.UpdateFriendRemarkname()
	self.UpdateGroupRemarkname()

	logging.Info(fmt.Sprintf("初始话用户数据成功！加载到的用户共有 %d 条，群聊共有 %d 条。", friends.Count(), groups.Count()))
	logging.Debug("初始化用户数据", map[string]interface{}{"self": self})
}

// updateMyGroups 更新备份群组
func (s *Self) updateMyGroups() {
	s.MyGroups = make([]*openwechat.User, 0, s.Groups.Count())
	for _, group := range s.Groups {
		copyGroup := *group.User
		s.MyGroups = append(s.MyGroups, &copyGroup)
	}
}

// updateMyFriends 更新备份好友
func (s *Self) updateMyFriends() {
	s.MyFriends = make([]*openwechat.User, 0, s.Friends.Count())
	for _, friend := range s.Friends {
		copyUser := *friend.User
		s.MyFriends = append(s.MyFriends, &copyUser)
	}
}

// UpdateGroups 更新群组
func (s *Self) UpdateGroups() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	s.Groups, err = s.self.Groups(true)
	if err != nil {
		logging.WarnWithErr(err, "更新群组失败")
		return
	}
	s.updateMyGroups()
	s.UpdateGroupRemarkname() // 更新备注信息以及 uid缓存
}

// UpdateFriends 更新好友
func (s *Self) UpdateFriends() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	s.Friends, err = s.self.Friends(true)
	if err != nil {
		logging.WarnWithErr(err, "更新好友失败")
		return
	}
	s.updateMyFriends()
	s.UpdateFriendRemarkname() // 更新备注信息以及 uid缓存
}

// SendText2FriendByNickname 根据好友名称发送文字
func (s *Self) SendText2FriendByNickname(nickname string, text string) error {
	// 查询好友
	results := s.Friends.SearchByNickName(1, nickname) // todo 好友可能名字重复，暂时没有好的解决方案
	friend := results.First()
	if friend == nil {
		return fmt.Errorf("SendText2FriendByNickname failed: %w", ErrFriendNotFound)
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
		return fmt.Errorf("SendImg2FriendByNickname failed: %w", ErrFriendNotFound)
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

// SendFile2FriendById 根据好友id发送文件
func (s *Self) SendFile2FriendById(avatarId string, file io.Reader) error {
	friend := s.Friends.SearchByID(avatarId).First()
	if friend == nil {
		return fmt.Errorf("SendFile2FriendById failed: %w", ErrFriendNotFound)
	}
	_, err := friend.SendFile(file)
	if err != nil {
		return fmt.Errorf("SendFile2FriendById failed: %w", err)
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

func (s *Self) SendText2GroupById(id string, text string) error {
	grouper := s.Groups.SearchByID(id).First()
	if grouper == nil {
		return fmt.Errorf("SendText2GroupById failed: %w", ErrGroupNotFound)
	}
	_, err := grouper.SendText(text)
	if err != nil {
		return fmt.Errorf("SendText2GroupById failed: %w", err)
	}
	return nil
}

func (s *Self) SendImg2GroupById(id string, img io.Reader) error {
	grouper := s.Groups.SearchByID(id).First()
	if grouper == nil {
		return fmt.Errorf("SendText2GroupById failed: %w", ErrGroupNotFound)
	}
	_, err := grouper.SendImage(img)
	if err != nil {
		return fmt.Errorf("SendText2GroupById failed: %w", err)
	}
	return nil
}

func (s *Self) SendFile2GroupById(id string, file io.Reader) error {
	grouper := s.Groups.SearchByID(id).First()
	if grouper == nil {
		return fmt.Errorf("SendFile2GroupById failed: %w", ErrGroupNotFound)
	}
	_, err := grouper.SendFile(file)
	if err != nil {
		return fmt.Errorf("SendFile2GroupById failed: %w", err)
	}
	return nil
}

func (s *Self) SendTextById(id string, text string, isGroup bool) error {
	var err error
	if isGroup {
		err = s.SendText2GroupById(id, text)
	} else {
		err = s.SendText2FriendById(id, text)
	}
	// 如果找不到群组或用户，更新后重试一次
	if err != nil && errors.Is(err, ErrFriendNotFound) {
		s.UpdateFriends()
		err = s.SendText2FriendById(id, text)
	} else if err != nil && errors.Is(err, ErrGroupNotFound) {
		s.UpdateGroups()
		err = s.SendText2GroupById(id, text)
	}

	return fmt.Errorf("SendTextById failed: %w", err)
}

func (s *Self) SendImgById(id string, img io.Reader, isGroup bool) error {
	var err error
	if isGroup {
		err = s.SendImg2GroupById(id, img)
	} else {
		err = s.SendImg2FriendById(id, img)
	}
	return fmt.Errorf("SendImgById failed: %w", err)
}

func (s *Self) SendFileById(id string, file io.Reader, isGroup bool) error {
	var err error
	if isGroup {
		err = s.SendFile2GroupById(id, file)
	} else {
		err = s.SendFile2FriendById(id, file)
	}
	return fmt.Errorf("SendFileById failed: %w", err)
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
		groupnames[i] = s.Groups[i].NickName
	}
	return groupnames
}

// GetFriendsList 获取所有好友的好友名
func (s *Self) GetFriendsList() []string {
	friendcnt := s.Friends.Count()
	friendnames := make([]string, friendcnt)
	for i := 0; i < friendcnt; i++ {
		friendnames[i] = s.Friends[i].NickName
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
			return "", fmt.Errorf("doGetIdByNickname failed: %w", ErrGroupNotFound)
		}
		return group.AvatarID(), nil
	} else {
		friend := s.Friends.SearchByNickName(1, nickname).First()
		if friend == nil {
			return "", fmt.Errorf("doGetIdByNickname failed: %w", ErrFriendNotFound)
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
			return "", fmt.Errorf("doGetNicknameById failed: %w", ErrGroupNotFound)
		}
		return group.NickName, nil
	} else {
		friend := s.Friends.SearchByID(id).First()
		if friend == nil {
			return "", fmt.Errorf("doGetNicknameById failed: %w", ErrFriendNotFound)
		}
		return friend.NickName, nil
	}
}

// UpdateGroupRemarkname 根据群名称更新备注群名 无法备注，用本地存储
// 如果已有备注名则忽略
// 重复的备注名则增加序号区分
func (s *Self) UpdateGroupRemarkname() {
	// 清空uid映射
	s.UidGroupDict = make(map[string]*openwechat.Group, s.Groups.Count())
	s.UidGroupNotUnique = make(map[string]bool, s.Groups.Count())
	var remarkSet = make(map[string]int)
	for _, group := range s.MyGroups {
		if group.RemarkName == "" { // 没有备注名，自动备注为群名
			newRemark := group.NickName
			cnt := remarkSet[newRemark]
			uuid := secretutil.GenerateUnitId(newRemark)
			if cnt > 0 { // 备注名称存在重复
				// 不注册并删除重复条uuid-group
				if _, ok := s.UidGroupDict[uuid]; ok {
					delete(s.UidGroupDict, uuid)
				}
				// 并且注册为重复uuid，提醒应用端
				s.UidGroupNotUnique[uuid] = true
				continue
			}
			remarkSet[newRemark] = cnt + 1
			group.RemarkName = newRemark
		}
		g := s.Groups.Search(1, func(g *openwechat.Group) bool {
			if g.AvatarID() == group.AvatarID() {
				return true
			}
			if g.RemarkName == group.RemarkName {
				return true
			}
			if g.NickName == group.NickName {
				return true
			}
			return false
		}).First()
		s.SetGroupUid(g, group) // 缓存uid对应群聊
	}
}

// UpdateFriendRemarkname 根据用户名称更新用户备注
func (s *Self) UpdateFriendRemarkname() {
	// 清空uid映射
	s.UidFriendDict = make(map[string]*openwechat.Friend, s.Friends.Count())
	s.UidFriendNotUnique = make(map[string]bool, s.Friends.Count())
	var remarkSet = make(map[string]int)
	for _, friend := range s.MyFriends {
		if friend.RemarkName == "" {
			newRemark := friend.NickName
			cnt := remarkSet[newRemark]
			uuid := secretutil.GenerateUnitId(newRemark)
			if cnt > 0 { // 备注名称存在重复
				// 不注册并删除重复条uuid-group
				if _, ok := s.UidFriendDict[uuid]; ok {
					delete(s.UidFriendDict, uuid)
				}
				// 并且注册为重复uuid，提醒应用端
				s.UidFriendNotUnique[uuid] = true
				continue
			}
			friend.RemarkName = newRemark
			remarkSet[newRemark] = cnt + 1
		}
		f := s.Friends.Search(1, func(f *openwechat.Friend) bool {
			if f.AvatarID() == friend.AvatarID() {
				return true
			}
			if f.RemarkName == friend.RemarkName {
				return true
			}
			if f.NickName == friend.NickName {
				return true
			}
			return false
		}).First()
		s.SetFriendUid(f, friend) // 缓存uid对应用户
	}
}

func (s *Self) SetFriendUid(friend *openwechat.Friend, f *openwechat.User) {
	uid := secretutil.GenerateUnitId(f.RemarkName)
	s.UidFriendDict[uid] = friend
}

func (s *Self) SetGroupUid(group *openwechat.Group, g *openwechat.User) {
	uid := secretutil.GenerateUnitId(g.RemarkName)
	s.UidGroupDict[uid] = group
}

// SendTextByUuid 根据uuid 发送文字
// nolint:wrapcheck
func (s *Self) SendTextByUuid(uuid string, text string, isGroup bool) error {
	var err error
	err = s.doSendTextByUuid(uuid, text, isGroup)
	if errors.Is(err, ErrGroupNotUnique) {
		return fmt.Errorf("sendTextByUuid failed: %w", ErrGroupNotUnique)
	}
	if errors.Is(err, ErrFriendNotUnique) {
		return fmt.Errorf("sendTextByUuid failed: %w", ErrFriendNotUnique)
	}
	if err != nil {
		switch true {
		case errors.Is(err, ErrGroupNotFound):
			s.UpdateGroups() // 尝试更新群组后再重发一次
			err = s.doSendTextByUuid(uuid, text, isGroup)
			if err != nil {
				return fmt.Errorf("second attempt after group update failed: %w", err)
			}
		case errors.Is(err, ErrFriendNotFound):
			s.UpdateFriends()
			err = s.doSendTextByUuid(uuid, text, isGroup)
			if err != nil {
				return fmt.Errorf("second attempt after friend update failed: %w", err)
			}
		default:
			return fmt.Errorf("sendTextByUuid failed with unexpected error: %w", err)
		}
	}
	return nil
}

// nolint:wrapcheck
func (s *Self) doSendTextByUuid(uuid string, text string, isGroup bool) error {
	// todo 重构 回调消息id 支持撤回
	if isGroup {
		// 判断是否为重复uuid
		if s.UidGroupNotUnique[uuid] {
			return fmt.Errorf("sendTextByUuid failed: %w", ErrGroupNotUnique)
		}
		group, ok := s.UidGroupDict[uuid]
		if !ok {
			return fmt.Errorf("sendTextByUuid failed: %w", ErrGroupNotFound)
		}
		_, err := group.SendText(text)
		if err != nil {
			return fmt.Errorf("sendTextByUuid openwechat failed: %w", err)
		}
	} else {
		// 判断是否为重复uuid
		if s.UidFriendNotUnique[uuid] {
			return fmt.Errorf("sendTextByUuid failed: %w", ErrFriendNotUnique)
		}
		friend, ok := s.UidFriendDict[uuid]
		if !ok {
			return fmt.Errorf("sendTextByUuid failed: %w", ErrFriendNotFound)
		}
		_, err := friend.SendText(text)
		if err != nil {
			return fmt.Errorf("sendTextByUuid openwechat failed: %w", err)
		}
	}
	return nil
}

// SendImgByUuid 根据uuid 发送图片
// nolint:wrapcheck
func (s *Self) SendImgByUuid(uuid string, img io.Reader, isGroup bool) error {
	var err error
	err = s.doSendImgByUuid(uuid, img, isGroup)
	if errors.Is(err, ErrGroupNotUnique) {
		return fmt.Errorf("sendTextByUuid failed: %w", ErrGroupNotUnique)
	}
	if errors.Is(err, ErrFriendNotUnique) {
		return fmt.Errorf("sendTextByUuid failed: %w", ErrFriendNotUnique)
	}
	if err != nil {
		switch true {
		case errors.Is(err, ErrGroupNotFound):
			s.UpdateGroups() // 尝试更新群组后再重发一次
			err = s.doSendImgByUuid(uuid, img, isGroup)
			if err != nil {
				return fmt.Errorf("second attempt after group update failed: %w", err)
			}
		case errors.Is(err, ErrFriendNotFound):
			s.UpdateFriends()
			err = s.doSendImgByUuid(uuid, img, isGroup)
			if err != nil {
				return fmt.Errorf("second attempt after friend update failed: %w", err)
			}
		default:
			return fmt.Errorf("sendTextByUuid failed with unexpected error: %w", err)
		}

	}
	return nil
}

// nolint:wrapcheck
func (s *Self) doSendImgByUuid(uuid string, img io.Reader, isGroup bool) error {
	if isGroup {
		// 判断是否为重复uuid
		if s.UidGroupNotUnique[uuid] {
			return fmt.Errorf("sendTextByUuid failed: %w", ErrGroupNotUnique)
		}
		group, ok := s.UidGroupDict[uuid]
		if !ok {
			return fmt.Errorf("sendImgByUuid failed: %w", ErrGroupNotFound)
		}
		_, err := group.SendImage(img)
		if err != nil {
			return fmt.Errorf("sendImgByUuid openwechat failed: %w", err)
		}
	} else {
		// 判断是否为重复uuid
		if s.UidFriendNotUnique[uuid] {
			return fmt.Errorf("sendTextByUuid failed: %w", ErrFriendNotUnique)
		}
		friend, ok := s.UidFriendDict[uuid]
		if !ok {
			return fmt.Errorf("sendImgByUuid failed: %w", ErrFriendNotFound)
		}
		_, err := friend.SendImage(img)
		if err != nil {
			return fmt.Errorf("sendImgByUuid openwechat failed: %w", err)
		}
	}
	return nil
}

// IsUuidValid 是否为uuid
func IsUuidValid(uuid string) bool {
	if len(uuid) != 16 {
		return false
	}
	return true
}

const (
	UUID_NOT_UNIQUE_INGROUPS  = "That uuid is not unique in groups! Error!"
	UUID_NOT_UNIQUE_INFRIENDS = "That uuid is not unique in friends! Error!"
)

// GetUuidById 根据 用户id获取uuid
func (s *Self) GetUuidById(user *openwechat.User, isGroup bool) string {
	remarkName := user.RemarkName
	var uuid = secretutil.GenerateUnitId(remarkName)
	if remarkName == "" {
		uuid = secretutil.GenerateUnitId(user.NickName)
	}
	// 校验是否为重复uuid
	if isGroup {
		if s.UidGroupNotUnique[uuid] {
			uuid = UUID_NOT_UNIQUE_INGROUPS
		}
	} else {
		if s.UidFriendNotUnique[uuid] {
			uuid = UUID_NOT_UNIQUE_INFRIENDS
		}
	}
	return uuid
}

func (mf MyFriends) SearchById(id string) *openwechat.User {
	for _, friend := range mf {
		if friend.AvatarID() == id {
			return friend
		}
	}
	return nil
}

func (mg MyGroups) SearchById(id string) *openwechat.User {
	for _, g := range mg {
		if g.AvatarID() == id {
			return g
		}
	}
	return nil
}
