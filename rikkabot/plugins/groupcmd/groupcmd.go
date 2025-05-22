package groupcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/coreapi"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/groupmanager"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
)

const (
	pluginName = "GroupCmd"
	pluginHelp = `分组管理插件 (管理员专用):
!group help - 显示此帮助信息
!group create <分组名称> - 创建新分组
!group delete <分组ID或名称> - 删除分组
!group rename <旧分组ID或名称> <新分组名称> - 重命名分组
!group list - 列出所有分组
!group id <分组ID或名称> - 根据ID或名称查询分组信息
!group add <分组ID或名称> 【注意：在需要操作的朋友or群聊中使用】
!group remove <分组ID或名称> <成员ID> - 从分组移除成员
!group members <分组ID或名称> - 查看分组内成员
!group memberof - 查看成员所在的所有分组 【注意：在需要操作的朋友or群聊中使用】`
)

type GroupCommandPlugin struct {
	core  *coreapi.Core
	cache *cache.Cache
	*plugins.Plugin
}

var groupCmdPlugin = &GroupCommandPlugin{}

func init() {
	groupCmdPlugin.Plugin = plugins.DefaultPlugin(pluginName).AsEnable().AsInitFunc(groupCmdPlugin.Init)

	adminCheckerMatcher := matcher.Custom{MatchFunc: func(msg *message.Message) bool {
		if msg.IsMySelf { // 自己是超级管理员
			return true
		}
		if groupCmdPlugin.cache == nil {
			globalCache := cache.GetCache()
			if globalCache == nil {
				return false
			}
			return globalCache.HasAdminUserId(msg.WxId)
		}
		return groupCmdPlugin.cache.HasAdminUserId(msg.WxId)
	}}

	baseGroupCmdMatcher := matcher.Default().And().N(
		matcher.BaseMatcher{AllowMsgType: message.MsgTypeText},
		adminCheckerMatcher,
		matcher.NewPrefixMatcher(false, true, "!group"),
	)

	helpAction := plugins.DefaultActionHandler("groupHelp", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, false, "help"))).
		AsActionFunc(func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
			reply = *recvMsg
			reply.Content = pluginHelp
			return reply, true, nil
		})
	groupCmdPlugin.AsAction(helpAction)

	createAction := plugins.DefaultActionHandler("groupCreate", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "create"))).
		AsActionFunc(groupCmdPlugin.handleCreateGroupCmd)
	groupCmdPlugin.AsAction(createAction)

	deleteAction := plugins.DefaultActionHandler("groupDelete", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "delete"))).
		AsActionFunc(groupCmdPlugin.handleDeleteGroupCmd)
	groupCmdPlugin.AsAction(deleteAction)

	renameAction := plugins.DefaultActionHandler("groupRename", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "rename"))).
		AsActionFunc(groupCmdPlugin.handleRenameGroupCmd)
	groupCmdPlugin.AsAction(renameAction)

	listAction := plugins.DefaultActionHandler("groupList", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, false, "list"))).
		AsActionFunc(groupCmdPlugin.handleListGroupsCmd)
	groupCmdPlugin.AsAction(listAction)

	idAction := plugins.DefaultActionHandler("groupGetInfo", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "id"))).
		AsActionFunc(groupCmdPlugin.handleGetGroupInfoCmd)
	groupCmdPlugin.AsAction(idAction)

	addMemberAction := plugins.DefaultActionHandler("groupAddMember", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "add"))).
		AsActionFunc(groupCmdPlugin.handleAddMemberCmd)
	groupCmdPlugin.AsAction(addMemberAction)

	removeMemberAction := plugins.DefaultActionHandler("groupRemoveMember", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "remove"))).
		AsActionFunc(groupCmdPlugin.handleRemoveMemberCmd)
	groupCmdPlugin.AsAction(removeMemberAction)

	listMembersAction := plugins.DefaultActionHandler("groupListMembers", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "members"))).
		AsActionFunc(groupCmdPlugin.handleListGroupMembersCmd)
	groupCmdPlugin.AsAction(listMembersAction)

	listMemberGroupsAction := plugins.DefaultActionHandler("groupListMemberGroups", true).
		AsMatcher(baseGroupCmdMatcher.And().N(matcher.NewPrefixMatcher(false, true, "memberof"))).
		AsActionFunc(groupCmdPlugin.handleListMemberGroupsCmd)
	groupCmdPlugin.AsAction(listMemberGroupsAction)

	plugins.GetAutoRegister().RegisterPlugin(groupCmdPlugin)
}

func (p *GroupCommandPlugin) Init() {
	p.core = coreapi.GetCore()
	p.cache = cache.GetCache()
}

// resolveGroup 尝试通过ID或名称解析分组，并返回Group对象。
// 如果找不到或发生错误，则返回相应的错误。
func (p *GroupCommandPlugin) resolveGroup(idOrName string) (*groupmanager.Group, error) {
	if p.core == nil || p.core.GroupManager == nil {
		return nil, errors.New("GroupManager 或 RikkaBot 实例未初始化")
	}
	gm := p.core.GroupManager
	// 尝试按ID获取
	groupByID, errByID := gm.GetGroupByID(idOrName)
	if errByID == nil && groupByID != nil {
		return groupByID, nil
	}
	// 尝试按名称获取
	groupByName, errByName := gm.GetGroupByName(idOrName)
	if errByName == nil && groupByName != nil {
		return groupByName, nil
	}

	// 处理错误情况
	if errors.Is(errByID, groupmanager.ErrGroupNotFound) && errors.Is(errByName, groupmanager.ErrGroupNotFoundByName) {
		return nil, fmt.Errorf("分组 '%s' 未找到 (已尝试作为ID和名称进行查询)", idOrName)
	}
	if errByID != nil && !errors.Is(errByID, groupmanager.ErrGroupNotFound) {
		return nil, fmt.Errorf("通过ID '%s' 查询分组时出错: %w", idOrName, errByID)
	}
	if errByName != nil && !errors.Is(errByName, groupmanager.ErrGroupNotFoundByName) {
		return nil, fmt.Errorf("通过名称 '%s' 查询分组时出错: %w", idOrName, errByName)
	}
	return nil, fmt.Errorf("分组 '%s' 未找到", idOrName)
}

func (p *GroupCommandPlugin) handleCreateGroupCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	groupName := strings.TrimSpace(recvMsg.Content)
	if groupName == "" {
		reply.Content = "用法: !group create <分组名称>"
		return reply, true, nil
	}
	if p.core == nil || p.core.GroupManager == nil {
		reply.Content = "错误: GroupManager 未初始化"
		return reply, true, errors.New("GroupManager uninitialized")
	}
	group, errManager := p.core.GroupManager.CreateGroup(groupName)
	if errManager != nil {
		reply.Content = fmt.Sprintf("创建分组 '%s' 失败: %v", groupName, errManager)
		return reply, true, errManager // 返回底层错误
	}
	reply.Content = fmt.Sprintf("分组 '%s' (ID: %s) 创建成功！", group.Name, group.ID)
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleDeleteGroupCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	groupIDOrName := strings.TrimSpace(recvMsg.Content)
	if groupIDOrName == "" {
		reply.Content = "用法: !group delete <分组ID或名称>"
		return reply, true, nil
	}

	group, errResolve := p.resolveGroup(groupIDOrName)
	if errResolve != nil {
		reply.Content = fmt.Sprintf("删除分组失败: %v", errResolve)
		return reply, true, errResolve // 返回底层错误
	}

	errManager := p.core.GroupManager.DeleteGroup(group.ID)
	if errManager != nil {
		reply.Content = fmt.Sprintf("删除分组 '%s' (ID: %s) 失败: %v", group.Name, group.ID, errManager)
		return reply, true, errManager // 返回底层错误
	}

	reply.Content = fmt.Sprintf("分组 '%s' (ID: %s) 已成功删除。", group.Name, group.ID)
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleRenameGroupCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	args := strings.Fields(strings.TrimSpace(recvMsg.Content))
	if len(args) < 2 {
		reply.Content = "用法: !group rename <旧分组ID或名称> <新分组名称>"
		return reply, true, nil
	}
	oldGroupIDOrName := args[0]
	newGroupName := strings.Join(args[1:], " ")
	if strings.TrimSpace(newGroupName) == "" {
		reply.Content = "错误: 新分组名称不能为空。用法: !group rename <旧分组ID或名称> <新分组名称>"
		return reply, true, nil
	}

	group, errResolve := p.resolveGroup(oldGroupIDOrName)
	if errResolve != nil {
		reply.Content = fmt.Sprintf("重命名分组失败 (查找旧分组时出错): %v", errResolve)
		return reply, true, errResolve // 返回底层错误
	}

	oldNameForFeedback := group.Name
	errManager := p.core.GroupManager.RenameGroup(group.ID, newGroupName)
	if errManager != nil {
		reply.Content = fmt.Sprintf("重命名分组 '%s' (ID: %s) 为 '%s' 失败: %v", oldNameForFeedback, group.ID, newGroupName, errManager)
		return reply, true, errManager // 返回底层错误
	}

	reply.Content = fmt.Sprintf("分组 '%s' (ID: %s) 已成功重命名为 '%s'。", oldNameForFeedback, group.ID, newGroupName)
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleListGroupsCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	if p.core == nil || p.core.GroupManager == nil {
		reply.Content = "错误: GroupManager 未初始化"
		return reply, true, errors.New("GroupManager uninitialized")
	}
	groups, errManager := p.core.GroupManager.GetAllGroups()
	if errManager != nil {
		reply.Content = fmt.Sprintf("获取分组列表失败: %v", errManager)
		return reply, true, errManager // 返回底层错误
	}
	if len(groups) == 0 {
		reply.Content = "当前没有任何分组。"
		return reply, true, nil
	}
	var response strings.Builder
	response.WriteString("所有分组 (All Groups):\n")
	for i, group := range groups {
		response.WriteString(fmt.Sprintf("%d. 名称 (Name): %s (ID: %s)\n", i+1, group.Name, group.ID))
	}
	reply.Content = strings.TrimSpace(response.String())
	return reply, true, nil
}

// handleGetGroupInfoCmd 根据名称或ID查询分组信息
func (p *GroupCommandPlugin) handleGetGroupInfoCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	groupIDOrName := strings.TrimSpace(recvMsg.Content)
	if groupIDOrName == "" {
		reply.Content = "用法: !group id <分组ID或名称>"
		return reply, true, nil
	}

	group, errResolve := p.resolveGroup(groupIDOrName)
	if errResolve != nil {
		reply.Content = fmt.Sprintf("查询分组 '%s' 信息失败: %v", groupIDOrName, errResolve)
		return reply, true, errResolve // 返回底层错误
	}

	reply.Content = fmt.Sprintf("分组信息 (Group Info):\n名称 (Name): %s\nID: %s", group.Name, group.ID)
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleAddMemberCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	args := strings.Fields(strings.TrimSpace(recvMsg.Content))
	if len(args) < 1 {
		reply.Content = "用法: !group add <分组ID或名称>"
		return reply, true, nil
	}
	groupIDOrName := args[0]
	memberID := recvMsg.WxId // 直接获取消息场地
	if recvMsg.IsGroup {
		memberID = recvMsg.RoomId
	}

	group, errResolve := p.resolveGroup(groupIDOrName)
	if errResolve != nil {
		reply.Content = fmt.Sprintf("添加成员到分组失败 (查找分组时出错): %v", errResolve)
		return reply, true, errResolve // 返回底层错误
	}

	errManager := p.core.GroupManager.AddMemberToGroup(group.ID, memberID)
	if errManager != nil {
		reply.Content = fmt.Sprintf("添加成员 '%s' 到分组 '%s' (ID: %s) 失败: %v", memberID, group.Name, group.ID, errManager)
		return reply, true, errManager // 返回底层错误
	}

	reply.Content = fmt.Sprintf("成员 '%s' 已成功添加到分组 '%s' (ID: %s)。", memberID, group.Name, group.ID)
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleRemoveMemberCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	args := strings.Fields(strings.TrimSpace(recvMsg.Content))
	if len(args) < 2 {
		reply.Content = "用法: !group remove <分组ID或名称> <成员ID>"
		return reply, true, nil
	}
	groupIDOrName := args[0]
	memberID := args[1]
	if strings.TrimSpace(memberID) == "" {
		reply.Content = "错误: 成员ID不能为空。用法: !group remove <分组ID或名称> <成员ID>"
		return reply, true, nil
	}

	group, errResolve := p.resolveGroup(groupIDOrName)
	if errResolve != nil {
		reply.Content = fmt.Sprintf("从分组移除成员失败 (查找分组时出错): %v", errResolve)
		return reply, true, errResolve // 返回底层错误
	}

	errManager := p.core.GroupManager.RemoveMemberFromGroup(group.ID, memberID)
	if errManager != nil {
		reply.Content = fmt.Sprintf("从分组 '%s' (ID: %s) 移除成员 '%s' 失败: %v", group.Name, group.ID, memberID, errManager)
		return reply, true, errManager // 返回底层错误
	}

	reply.Content = fmt.Sprintf("成员 '%s' 已成功从分组 '%s' (ID: %s) 移除。", memberID, group.Name, group.ID)
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleListGroupMembersCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	groupIDOrName := strings.TrimSpace(recvMsg.Content)
	if groupIDOrName == "" {
		reply.Content = "用法: !group members <分组ID或名称>"
		return reply, true, nil
	}

	group, errResolve := p.resolveGroup(groupIDOrName)
	if errResolve != nil {
		reply.Content = fmt.Sprintf("查看分组 '%s' 成员失败 (查找分组时出错): %v", groupIDOrName, errResolve)
		return reply, true, errResolve // 返回底层错误
	}

	members, errManager := p.core.GroupManager.GetGroupMembers(group.ID)
	if errManager != nil {
		reply.Content = fmt.Sprintf("获取分组 '%s' (ID: %s) 成员列表失败: %v", group.Name, group.ID, errManager)
		return reply, true, errManager // 返回底层错误
	}

	if len(members) == 0 {
		reply.Content = fmt.Sprintf("分组 '%s' (ID: %s) 中没有成员。", group.Name, group.ID)
		return reply, true, nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("分组 '%s' (ID: %s) 的成员 (Members of group '%s'):\n", group.Name, group.ID, group.Name))
	for i, member := range members {
		response.WriteString(fmt.Sprintf("%d. ID: %s (类型 Type: %s)\n", i+1, member.ID, member.Type))
	}
	reply.Content = strings.TrimSpace(response.String())
	return reply, true, nil
}

func (p *GroupCommandPlugin) handleListMemberGroupsCmd(ctx context.Context, recvMsg *message.Message) (reply message.Message, send bool, err error) {
	reply = *recvMsg
	memberID := recvMsg.WxId
	if recvMsg.IsGroup {
		memberID = recvMsg.RoomId
	}
	if p.core == nil || p.core.GroupManager == nil {
		reply.Content = "错误: GroupManager 未初始化"
		return reply, true, errors.New("GroupManager uninitialized")
	}
	groups, errManager := p.core.GroupManager.GetMemberGroups(memberID)
	if errManager != nil {
		reply.Content = fmt.Sprintf("查询成员 '%s' 所在分组失败: %v", memberID, errManager)
		return reply, true, errManager // 返回底层错误
	}
	if len(groups) == 0 {
		reply.Content = fmt.Sprintf("成员 '%s' 不属于任何分组。", memberID)
		return reply, true, nil
	}
	var response strings.Builder
	response.WriteString(fmt.Sprintf("成员 '%s' 所在的分组 (Groups member '%s' belongs to):\n", memberID, memberID))
	for i, group := range groups {
		response.WriteString(fmt.Sprintf("%d. 名称 (Name): %s (ID: %s)\n", i+1, group.Name, group.ID))
	}
	reply.Content = strings.TrimSpace(response.String())
	return reply, true, nil
}

var _ plugins.IPlugin = &GroupCommandPlugin{}
