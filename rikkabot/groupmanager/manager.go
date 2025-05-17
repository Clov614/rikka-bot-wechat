package groupmanager

import (
	"encoding/base64"
	"encoding/json" // 用于json序列化和反序列化
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Clov614/logging"

	cachepkg "github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache" // 导入cache包并使用别名
	"github.com/google/uuid"
	// "go.etcd.io/bbolt" // 不再直接使用 bbolt
)

// Group 定义了分组的结构
type Group struct {
	ID   string `json:"id"`   // 分组唯一ID (例如 UUID)
	Name string `json:"name"` // 分组名称
}

// MemberType 定义了成员的类型，可以是用户或群聊
type MemberType string

const (
	// UserMember 表示成员类型为用户
	UserMember MemberType = "user"
	// GroupMember 表示成员类型为群聊
	GroupMember MemberType = "group_chat"
)

// Member 定义了分组内成员的结构
type Member struct {
	ID   string     `json:"id"`   // wxid 或 room_id
	Type MemberType `json:"type"` // 成员类型
}

// GroupManagerState 封装了 GroupManager 的所有持久化状态
type GroupManagerState struct {
	Groups         map[string]Group    `json:"groups"`           // groupID -> Group object
	MemberToGroups map[string][]string `json:"member_to_groups"` // memberKey -> list of groupIDs
	GroupToMembers map[string][]Member `json:"group_to_members"` // groupID -> list of Member objects
}

// IGroupManager 定义了分组管理器的接口 (保持不变, 已更新为不含 memberType 的签名)
type IGroupManager interface {
	CreateGroup(name string) (*Group, error)
	DeleteGroup(groupID string) error
	RenameGroup(groupID string, newName string) error
	GetAllGroups() ([]*Group, error)
	AddMemberToGroup(groupID string, memberID string) error
	RemoveMemberFromGroup(groupID string, memberID string) error
	GetGroupMembers(groupID string) ([]*Member, error)
	GetMemberGroups(memberID string) ([]*Group, error)
	GetGroupByName(name string) (*Group, error)
	GetGroupByID(groupID string) (*Group, error)
}

// GroupManager 结构体实现了 IGroupManager 接口
type GroupManager struct {
	cache *cachepkg.Cache // 缓存实例，用于持久化完整状态及加速单项查询
	mu    sync.RWMutex    // 读写锁，用于保证并发安全

	state         *GroupManagerState // 内存中的完整状态
	stateCacheKey string             // 用于在 PluginsCache 中存储 GroupManagerState 的键
}

const (
	// 定义用于 GroupManager 特定缓存的键前缀 (用于单项查询加速)
	cachePrefixGroupByID    = "groupmanager:group:id:"
	cachePrefixGroupByName  = "groupmanager:group:name:"
	cachePrefixGroupMembers = "groupmanager:group_members:"
	cachePrefixMemberGroups = "groupmanager:member_groups:" // 此缓存存储 []string (groupIDs)

	// GroupManagerState 在 PluginsCache 中的默认存储键
	defaultStateCacheKey = "groupmanager:internal_full_state_v1"
)

// 定义 GroupManager 使用的静态错误
var (
	ErrCacheInstanceNil          = errors.New("缓存实例不能为空")
	ErrStateNotFoundInCache      = errors.New("状态未在缓存中找到")
	ErrUnknownStateTypeInCache   = errors.New("缓存中的状态数据类型未知")
	ErrDeserializeStateFailed    = errors.New("从缓存反序列化状态失败")
	ErrSerializeStateFailed      = errors.New("序列化状态以便缓存失败")
	ErrGroupExistsWithName       = errors.New("分组名称已存在")
	ErrGroupNotFound             = errors.New("分组未找到")
	ErrGroupNotFoundByName       = errors.New("按名称未找到分组")
	ErrGroupRenameFailed         = errors.New("重命名分组失败")
	ErrGroupDeleteFailed         = errors.New("删除分组失败")
	ErrMemberTypeInferenceFailed = errors.New("无法从ID推断成员类型")
	ErrGroupDoesNotExist         = errors.New("分组不存在")
	ErrMemberAlreadyInGroup      = errors.New("成员已存在于分组中")
	ErrAddMemberFailed           = errors.New("添加成员到分组失败")
	ErrRemoveMemberFailed        = errors.New("从分组移除成员失败")
	ErrMemberNotInGroup          = errors.New("成员未在分组中找到")
	ErrPersistStateFailed        = errors.New("持久化状态失败")
)

// NewGroupManager 创建一个新的 GroupManager 实例
func NewGroupManager(cache *cachepkg.Cache, optionalStateCacheKey string) (*GroupManager, error) {
	if cache == nil {
		return nil, ErrCacheInstanceNil
	}

	stateKey := optionalStateCacheKey
	if stateKey == "" {
		stateKey = defaultStateCacheKey
	}

	gm := &GroupManager{
		cache:         cache,
		stateCacheKey: stateKey,
		state: &GroupManagerState{ // 初始化空状态
			Groups:         make(map[string]Group),
			MemberToGroups: make(map[string][]string),
			GroupToMembers: make(map[string][]Member),
		},
	}

	// 尝试从 PluginsCache 加载完整状态
	if err := gm.loadStateFromCache(); err != nil {
		// 如果加载失败 (例如首次启动或数据损坏)，记录日志并使用初始化的空状态
		logging.Warn("GroupManager - 无法从缓存加载状态，将使用空状态启动。", map[string]interface{}{"key": gm.stateCacheKey, "error": err.Error()})
		// 确保即使加载过程中部分修改了 state，这里也重置为空状态
		gm.state = &GroupManagerState{
			Groups:         make(map[string]Group),
			MemberToGroups: make(map[string][]string),
			GroupToMembers: make(map[string][]Member),
		}
	} else {
		logging.Info("GroupManager - 状态已成功从缓存加载。", map[string]interface{}{"key": gm.stateCacheKey})
	}
	return gm, nil
}

// loadStateFromCache 从 PluginsCache 加载 GroupManagerState
// 调用此方法前不需要外部锁，此方法内部会加锁。
func (gm *GroupManager) loadStateFromCache() error {
	gm.mu.Lock() // 写锁保护状态加载过程
	defer gm.mu.Unlock()

	cachedData, found := gm.cache.GetPluginInfo(gm.stateCacheKey)
	if !found {
		return ErrStateNotFoundInCache
	}

	var jsonData []byte
	var err error // <--- 为 base64.StdEncoding.DecodeString 和 json.Unmarshal 声明 err

	switch v := cachedData.(type) {
	case []byte:
		jsonData = v
		logging.Debug("GroupManager - Loaded state from cache as []byte.", map[string]interface{}{"key": gm.stateCacheKey, "byte_length": len(jsonData)})
	case string:
		logging.Debug("GroupManager - Loaded state from cache as string, attempting Base64 decode.", map[string]interface{}{"key": gm.stateCacheKey, "string_length": len(v)})
		// 如果缓存中存的是字符串，我们假设它可能是 Base64 编码的 JSON 字节
		// 尝试 Base64 解码
		var decodedBytes []byte
		decodedBytes, err = base64.StdEncoding.DecodeString(v) // <--- 修改: 赋值给新的局部变量
		if err != nil {
			// 如果解码失败，可能它本身就是一个普通的 JSON 字符串，而不是 Base64
			// 记录一个警告，然后尝试直接将其作为 JSON 字符串处理
			logging.Warn("GroupManager - Failed to Base64 decode string from cache, attempting to use as plain JSON string.",
				map[string]interface{}{"key": gm.stateCacheKey, "original_string": v, "decode_error": err.Error()})
			jsonData = []byte(v) // 直接使用原始字符串的字节
			err = nil            // 重置错误，因为我们将尝试按原样处理它
		} else {
			jsonData = decodedBytes // 使用解码后的字节
			logging.Debug("GroupManager - Successfully Base64 decoded string from cache.", map[string]interface{}{"key": gm.stateCacheKey, "decoded_byte_length": len(jsonData)})
		}
	default:
		return fmt.Errorf("%w: %T for key %s", ErrUnknownStateTypeInCache, cachedData, gm.stateCacheKey)
	}

	if len(jsonData) == 0 {
		gm.state = &GroupManagerState{
			Groups:         make(map[string]Group),
			MemberToGroups: make(map[string][]string),
			GroupToMembers: make(map[string][]Member),
		}
		logging.Info("GroupManager - Cache data for state is empty, initializing with empty state.", map[string]interface{}{"key": gm.stateCacheKey})
		return nil
	}

	var loadedState GroupManagerState
	// 确保在 Unmarshal 前 err 是 nil (如果上面 Base64 解码失败但我们决定继续的话)
	if err == nil { // 只有在之前的步骤没有设置不可恢复的错误时才尝试 Unmarshal
		err = json.Unmarshal(jsonData, &loadedState)
		if err != nil {
			return fmt.Errorf("%w on key '%s': %w. JSON data preview (first 100 bytes): '%s'", ErrDeserializeStateFailed, gm.stateCacheKey, err, stringOrPreview(jsonData, 100))
		}
	} else {
		// 如果 err 不为 nil (例如，之前 Base64 解码失败且我们决定不继续)，则直接返回该错误
		// 但根据当前逻辑，如果解码失败，err 会被重置为 nil，除非后续加入更严格的错误处理
		// 为了清晰，如果真的有之前的错误需要传递，应该在这里处理。
		// 不过，按当前改动，这里 err 应该是 nil。
	}

	if loadedState.Groups == nil {
		loadedState.Groups = make(map[string]Group)
	}
	if loadedState.MemberToGroups == nil {
		loadedState.MemberToGroups = make(map[string][]string)
	}
	if loadedState.GroupToMembers == nil {
		loadedState.GroupToMembers = make(map[string][]Member)
	}
	gm.state = &loadedState
	return nil
}

// stringOrPreview 返回字符串或其预览（如果太长）
func stringOrPreview(data []byte, maxLength int) string {
	if len(data) > maxLength {
		return string(data[:maxLength]) + "..."
	}
	return string(data)
}

// persistStateToCache 将当前内存状态保存到 PluginsCache
// 此方法必须在已持有 gm.mu (写锁) 的情况下调用
func (gm *GroupManager) persistStateToCache() error {
	jsonData, err := json.Marshal(gm.state)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSerializeStateFailed, err)
	}
	gm.cache.CachePluginInfo(gm.stateCacheKey, jsonData)
	return nil
}

func generateID() string {
	return uuid.NewString()
}

func makeMemberKey(memberID string, memberType MemberType) string {
	return string(memberType) + ":" + memberID
}

func InferMemberTypeFromID(id string) (MemberType, error) {
	if strings.HasPrefix(id, "wxid_") && !strings.Contains(id, "@") {
		return UserMember, nil
	}
	if strings.HasSuffix(id, "@chatroom") {
		return GroupMember, nil
	}
	return "", fmt.Errorf("%w: 未知格式 '%s'", ErrMemberTypeInferenceFailed, id)
}

func (gm *GroupManager) CreateGroup(name string) (*Group, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for _, existingGroup := range gm.state.Groups {
		if existingGroup.Name == name {
			return nil, fmt.Errorf("%w: '%s'", ErrGroupExistsWithName, name)
		}
	}

	newGroup := Group{
		ID:   generateID(),
		Name: name,
	}

	gm.state.Groups[newGroup.ID] = newGroup
	if _, ok := gm.state.GroupToMembers[newGroup.ID]; !ok {
		gm.state.GroupToMembers[newGroup.ID] = make([]Member, 0)
	}

	if err := gm.persistStateToCache(); err != nil {
		delete(gm.state.Groups, newGroup.ID)
		delete(gm.state.GroupToMembers, newGroup.ID)
		return nil, fmt.Errorf("创建分组: %w: %w", ErrPersistStateFailed, err)
	}

	gm.cache.CachePluginInfo(cachePrefixGroupByID+newGroup.ID, newGroup)
	gm.cache.CachePluginInfo(cachePrefixGroupByName+newGroup.Name, newGroup)

	return &newGroup, nil
}

func (gm *GroupManager) GetGroupByID(groupID string) (*Group, error) {
	cacheKey := cachePrefixGroupByID + groupID
	if cached, found := gm.cache.GetPluginInfo(cacheKey); found {
		if group, ok := cached.(Group); ok {
			return &group, nil
		}
		logging.Warn("GroupManager: GetGroupByID - 缓存中的分组数据类型无效，将从主状态加载",
			map[string]interface{}{"cache_key": cacheKey, "expected_type": "Group", "actual_type": fmt.Sprintf("%T", cached)})
		gm.cache.RemovePluginCache(cacheKey) // 清除无效缓存
	}

	gm.mu.RLock()
	defer gm.mu.RUnlock()

	group, exists := gm.state.Groups[groupID]
	if !exists {
		return nil, ErrGroupNotFound
	}

	// 将从主状态获取的数据回填到缓存（如果之前未命中或类型错误）
	gm.cache.CachePluginInfo(cacheKey, group)
	gm.cache.CachePluginInfo(cachePrefixGroupByName+group.Name, group)

	return &group, nil
}

func (gm *GroupManager) GetGroupByName(name string) (*Group, error) {
	cacheKeyName := cachePrefixGroupByName + name
	if cached, found := gm.cache.GetPluginInfo(cacheKeyName); found {
		if group, ok := cached.(Group); ok {
			return &group, nil
		}
		logging.Warn("GroupManager: GetGroupByName - 缓存中的分组数据类型无效，将从主状态加载",
			map[string]interface{}{"cache_key": cacheKeyName, "expected_type": "Group", "actual_type": fmt.Sprintf("%T", cached)})
		gm.cache.RemovePluginCache(cacheKeyName) // 清除无效缓存
	}

	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for _, group := range gm.state.Groups {
		if group.Name == name {
			// 将从主状态获取的数据回填到缓存
			gm.cache.CachePluginInfo(cacheKeyName, group)
			gm.cache.CachePluginInfo(cachePrefixGroupByID+group.ID, group)
			return &group, nil
		}
	}
	return nil, ErrGroupNotFoundByName
}

func (gm *GroupManager) GetAllGroups() ([]*Group, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	groups := make([]*Group, 0, len(gm.state.Groups))
	for _, groupValue := range gm.state.Groups {
		g := groupValue
		groups = append(groups, &g)
	}
	return groups, nil
}

func (gm *GroupManager) RenameGroup(groupID string, newName string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	group, exists := gm.state.Groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}

	if group.Name == newName {
		logging.Debug("GroupManager: 分组新名称与旧名称相同，无需重命名。", map[string]interface{}{"group_id": groupID, "name": newName})
		return nil // 无操作，无错误
	}

	for id, g := range gm.state.Groups {
		if id != groupID && g.Name == newName {
			logging.Warn("GroupManager: 尝试将分组重命名为已存在的名称", map[string]interface{}{"group_id": groupID, "new_name": newName})
			return fmt.Errorf("%w: %s", ErrGroupExistsWithName, newName)
		}
	}

	oldName := group.Name
	updatedGroup := group
	updatedGroup.Name = newName
	gm.state.Groups[groupID] = updatedGroup

	if err := gm.persistStateToCache(); err != nil {
		// 恢复旧状态
		gm.state.Groups[groupID] = group
		return fmt.Errorf("%w: %w: %w", ErrGroupRenameFailed, ErrPersistStateFailed, err)
	}

	// 更新相关缓存
	gm.cache.RemovePluginCache(cachePrefixGroupByName + oldName)
	gm.cache.CachePluginInfo(cachePrefixGroupByName+newName, updatedGroup)
	gm.cache.CachePluginInfo(cachePrefixGroupByID+groupID, updatedGroup)

	return nil
}

func (gm *GroupManager) DeleteGroup(groupID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	groupToDelete, exists := gm.state.Groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}

	// 备份状态以便回滚
	originalGroup := groupToDelete
	originalMembersOfDeletedGroup, groupHadMembers := gm.state.GroupToMembers[groupID]
	originalMemberToGroups := make(map[string][]string)
	if groupHadMembers {
		for _, member := range originalMembersOfDeletedGroup {
			memberKey := makeMemberKey(member.ID, member.Type)
			if groups, ok := gm.state.MemberToGroups[memberKey]; ok {
				originalMemberToGroups[memberKey] = append([]string(nil), groups...)
			}
		}
	}

	delete(gm.state.Groups, groupID)

	membersOfDeletedGroup, groupHadMembers := gm.state.GroupToMembers[groupID] // 重新获取，因为上面可能修改了
	if groupHadMembers {
		delete(gm.state.GroupToMembers, groupID)
	}

	for _, member := range membersOfDeletedGroup {
		memberKey := makeMemberKey(member.ID, member.Type)
		if groupIDs, ok := gm.state.MemberToGroups[memberKey]; ok {
			newGroupIDs := make([]string, 0, len(groupIDs))
			for _, id := range groupIDs {
				if id != groupID {
					newGroupIDs = append(newGroupIDs, id)
				}
			}
			if len(newGroupIDs) == 0 {
				delete(gm.state.MemberToGroups, memberKey)
			} else {
				gm.state.MemberToGroups[memberKey] = newGroupIDs
			}
		}
	}

	if err := gm.persistStateToCache(); err != nil {
		// 回滚
		gm.state.Groups[groupID] = originalGroup
		if groupHadMembers {
			gm.state.GroupToMembers[groupID] = originalMembersOfDeletedGroup
		}
		for key, groups := range originalMemberToGroups {
			gm.state.MemberToGroups[key] = groups
		}
		// 此处状态可能已部分更改，回滚比较复杂，记录严重错误并返回
		logging.Error("删除分组后持久化状态失败，状态可能已不一致", map[string]interface{}{"groupID": groupID, "error": err.Error()})
		return fmt.Errorf("%w: %w: %w", ErrGroupDeleteFailed, ErrPersistStateFailed, err)
	}

	// 清理相关缓存
	gm.cache.RemovePluginCache(cachePrefixGroupByID + groupID)
	gm.cache.RemovePluginCache(cachePrefixGroupByName + groupToDelete.Name)
	gm.cache.RemovePluginCache(cachePrefixGroupMembers + groupID)
	for _, member := range membersOfDeletedGroup {
		memberKey := makeMemberKey(member.ID, member.Type)
		gm.cache.RemovePluginCache(cachePrefixMemberGroups + memberKey)
	}
	return nil
}

func (gm *GroupManager) AddMemberToGroup(groupID string, memberID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	memberType, err := InferMemberTypeFromID(memberID)
	if err != nil {
		return fmt.Errorf("%w (类型推断): %w", ErrAddMemberFailed, err)
	}

	if _, groupExists := gm.state.Groups[groupID]; !groupExists {
		return fmt.Errorf("%w: 分组 '%s' 不存在", ErrAddMemberFailed, groupID)
	}

	members := gm.state.GroupToMembers[groupID]
	for _, m := range members {
		if m.ID == memberID && m.Type == memberType {
			return fmt.Errorf("%w: 成员 '%s' (%s) 已在分组 '%s' 中", ErrMemberAlreadyInGroup, memberID, memberType, groupID)
		}
	}
	updatedMembers := append(members, Member{ID: memberID, Type: memberType})
	gm.state.GroupToMembers[groupID] = updatedMembers

	memberKey := makeMemberKey(memberID, memberType)
	groupIDs := gm.state.MemberToGroups[memberKey]

	isAlreadyInList := false
	for _, gID := range groupIDs {
		if gID == groupID {
			isAlreadyInList = true
			break
		}
	}
	if !isAlreadyInList {
		updatedGroupIDs := append(groupIDs, groupID)
		gm.state.MemberToGroups[memberKey] = updatedGroupIDs
	}

	if err := gm.persistStateToCache(); err != nil {
		// 回滚
		gm.state.GroupToMembers[groupID] = members
		if !isAlreadyInList { // 如果之前不在列表里，那么回滚时要确保如果列表因此变空则删除key
			if len(groupIDs) == 0 {
				delete(gm.state.MemberToGroups, memberKey)
			} else {
				gm.state.MemberToGroups[memberKey] = groupIDs
			}
		}
		return fmt.Errorf("%w: %w: %w", ErrAddMemberFailed, ErrPersistStateFailed, err)
	}

	// 清理相关缓存
	gm.cache.RemovePluginCache(cachePrefixGroupMembers + groupID)
	gm.cache.RemovePluginCache(cachePrefixMemberGroups + memberKey)
	return nil
}

func (gm *GroupManager) RemoveMemberFromGroup(groupID string, memberID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	memberType, err := InferMemberTypeFromID(memberID)
	if err != nil {
		return fmt.Errorf("%w (类型推断): %w", ErrRemoveMemberFailed, err)
	}

	if _, groupExists := gm.state.Groups[groupID]; !groupExists {
		return fmt.Errorf("%w: 分组 '%s' 不存在", ErrRemoveMemberFailed, groupID)
	}

	// 备份状态以便回滚
	originalMembersSnapshot, membersListExists := gm.state.GroupToMembers[groupID]
	var originalMembersCopy []Member
	if membersListExists {
		originalMembersCopy = append([]Member(nil), originalMembersSnapshot...)
	}

	memberKey := makeMemberKey(memberID, memberType)
	originalGroupIDsForMemberSnapshot, memberKeyExisted := gm.state.MemberToGroups[memberKey]
	var originalGroupIDsForMemberCopy []string
	if memberKeyExisted {
		originalGroupIDsForMemberCopy = append([]string(nil), originalGroupIDsForMemberSnapshot...)
	}

	originalMembers, membersListExists := gm.state.GroupToMembers[groupID] // 重新获取
	if !membersListExists {
		logging.Warn("GroupManager: 尝试从没有成员记录的分组中移除成员",
			map[string]interface{}{"group_id": groupID, "member_id": memberID, "member_type": memberType})
		return fmt.Errorf("%w: 成员 '%s' (%s) (分组 '%s' 无成员记录)", ErrMemberNotInGroup, memberID, memberType, groupID)
	}

	foundInGroup := false
	updatedMembers := make([]Member, 0, len(originalMembers))
	for _, m := range originalMembers {
		if m.ID == memberID && m.Type == memberType {
			foundInGroup = true
		} else {
			updatedMembers = append(updatedMembers, m)
		}
	}

	if !foundInGroup {
		return fmt.Errorf("%w: 成员 '%s' (%s) 未在分组 '%s' 中找到", ErrMemberNotInGroup, memberID, memberType, groupID)
	}
	gm.state.GroupToMembers[groupID] = updatedMembers

	memberKey = makeMemberKey(memberID, memberType)
	originalGroupIDs, memberKeyExists := gm.state.MemberToGroups[memberKey] // 重新获取
	groupRemovedFromMember := false

	if memberKeyExists {
		updatedGroupIDs := make([]string, 0, len(originalGroupIDs))
		for _, gID := range originalGroupIDs {
			if gID == groupID {
				groupRemovedFromMember = true
			} else {
				updatedGroupIDs = append(updatedGroupIDs, gID)
			}
		}
		if groupRemovedFromMember {
			if len(updatedGroupIDs) == 0 {
				delete(gm.state.MemberToGroups, memberKey)
			} else {
				gm.state.MemberToGroups[memberKey] = updatedGroupIDs
			}
		} else {
			logging.Warn("GroupManager: 成员从分组移除，但在其自己的分组列表中未找到该分组记录，数据可能存在不一致。",
				map[string]interface{}{"member_id": memberID, "member_type": memberType, "group_id": groupID})
		}
	} else if foundInGroup { // 即 memberKey 不存在，但成员却在分组的成员列表里
		logging.Warn("GroupManager: 成员从分组移除，但其 memberKey 在 MemberToGroups 中无记录，数据可能存在不一致。",
			map[string]interface{}{"member_id": memberID, "member_type": memberType, "group_id": groupID})
	}

	if err := gm.persistStateToCache(); err != nil {
		// 回滚
		if membersListExists {
			gm.state.GroupToMembers[groupID] = originalMembersCopy
		} else {
			delete(gm.state.GroupToMembers, groupID) // 如果原先就没有，回滚时也应该删除
		}

		if groupRemovedFromMember { // 如果成功从成员的分组列表中移除了此组ID
			if memberKeyExisted { // 且成员的key原本存在
				gm.state.MemberToGroups[memberKey] = originalGroupIDsForMemberCopy
			} else {
				// 如果key原本不存在，但不知何故 groupRemovedFromMember 为 true (理论上不应发生)
				// 此时不应创建新的key，但可以记录一个更严重的警告
				logging.Error("GroupManager: 移除成员时发生状态不一致，回滚 MemberToGroups 失败。",
					map[string]interface{}{"member_key": memberKey})
			}
		} else if memberKeyExisted { // 如果没有从成员分组列表中移除（例如它本就不在那），且key原本存在
			gm.state.MemberToGroups[memberKey] = originalGroupIDsForMemberCopy // 保持原样
		} else { // key 原本不存在，也没移除
			delete(gm.state.MemberToGroups, memberKey) // 确保 key 不存在
		}

		return fmt.Errorf("%w: %w: %w", ErrRemoveMemberFailed, ErrPersistStateFailed, err)
	}

	// 清理相关缓存
	gm.cache.RemovePluginCache(cachePrefixGroupMembers + groupID)
	gm.cache.RemovePluginCache(cachePrefixMemberGroups + memberKey)
	return nil
}

func (gm *GroupManager) GetGroupMembers(groupID string) ([]*Member, error) {
	cacheKey := cachePrefixGroupMembers + groupID
	if cached, found := gm.cache.GetPluginInfo(cacheKey); found {
		if memberSlice, ok := cached.([]Member); ok {
			result := make([]*Member, len(memberSlice))
			for i := range memberSlice {
				mCopy := memberSlice[i]
				result[i] = &mCopy
			}
			return result, nil
		}
		logging.Warn("GroupManager: GetGroupMembers - 缓存中的成员列表数据类型无效，将从主状态加载",
			map[string]interface{}{"cache_key": cacheKey, "expected_type": "[]Member", "actual_type": fmt.Sprintf("%T", cached)})
		gm.cache.RemovePluginCache(cacheKey) // 清除无效缓存
	}

	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if _, groupExists := gm.state.Groups[groupID]; !groupExists {
		gm.cache.CachePluginInfo(cacheKey, []Member{})
		logging.Debug("GroupManager: GetGroupMembers - 查询的分组不存在，返回空成员列表。", map[string]interface{}{"group_id": groupID})
		return []*Member{}, nil
	}

	members, exists := gm.state.GroupToMembers[groupID]
	if !exists {
		gm.cache.CachePluginInfo(cacheKey, []Member{})
		logging.Debug("GroupManager: GetGroupMembers - 分组存在但无成员记录，返回空成员列表。", map[string]interface{}{"group_id": groupID})
		return []*Member{}, nil
	}

	// 复制以缓存和返回
	membersCopyForCache := make([]Member, len(members))
	copy(membersCopyForCache, members)
	gm.cache.CachePluginInfo(cacheKey, membersCopyForCache)

	result := make([]*Member, len(members))
	for i := range members {
		mCopy := members[i]
		result[i] = &mCopy
	}
	return result, nil
}

func (gm *GroupManager) GetMemberGroups(memberID string) ([]*Group, error) {
	memberType, err := InferMemberTypeFromID(memberID)
	if err != nil {
		return nil, fmt.Errorf("获取成员分组失败 (类型推断): %w", err)
	}
	memberKey := makeMemberKey(memberID, memberType)

	cacheKeyIDs := cachePrefixMemberGroups + memberKey
	var groupIDs []string
	foundIDsInCache := false

	if cached, found := gm.cache.GetPluginInfo(cacheKeyIDs); found {
		if ids, ok := cached.([]string); ok {
			groupIDs = ids
			foundIDsInCache = true
		} else {
			logging.Warn("GroupManager: GetMemberGroups - 缓存中的 groupID 列表数据类型无效，将从主状态加载",
				map[string]interface{}{"cache_key": cacheKeyIDs, "expected_type": "[]string", "actual_type": fmt.Sprintf("%T", cached)})
			gm.cache.RemovePluginCache(cacheKeyIDs) // 清除无效缓存
		}
	}

	if !foundIDsInCache {
		gm.mu.RLock()
		idsFromState, exists := gm.state.MemberToGroups[memberKey]
		if !exists {
			gm.mu.RUnlock()
			// 成员不属于任何分组，缓存空ID列表并返回空分组列表
			gm.cache.CachePluginInfo(cacheKeyIDs, []string{})
			logging.Debug("GroupManager: GetMemberGroups - 成员不属于任何分组。", map[string]interface{}{"member_id": memberID, "member_type": memberType})
			return []*Group{}, nil
		}
		groupIDs = make([]string, len(idsFromState))
		copy(groupIDs, idsFromState)
		gm.mu.RUnlock()
		// 将从主状态获取的数据回填到缓存
		gm.cache.CachePluginInfo(cacheKeyIDs, groupIDs)
	}

	groupsResult := make([]*Group, 0, len(groupIDs))
	for _, gID := range groupIDs {
		group, err := gm.GetGroupByID(gID)
		if err != nil {
			logging.Warn("GetMemberGroups - 获取分组失败，跳过此分组", map[string]interface{}{"groupID": gID, "memberID": memberID, "error": err.Error()})
			continue
		}
		groupsResult = append(groupsResult, group)
	}
	return groupsResult, nil
}

// Helper function to safely get members for rollback; returns a copy
func (gm *GroupManager) getClonedMembers(groupID string) []Member {
	gm.mu.RLock() // Ensure read lock if called from a context that might not have it
	defer gm.mu.RUnlock()
	if members, ok := gm.state.GroupToMembers[groupID]; ok {
		cloned := make([]Member, len(members))
		copy(cloned, members)
		return cloned
	}
	return nil
}

// Helper function to safely get group IDs for a member for rollback; returns a copy
func (gm *GroupManager) getClonedMemberGroups(memberKey string) ([]string, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	if groupIDs, ok := gm.state.MemberToGroups[memberKey]; ok {
		cloned := make([]string, len(groupIDs))
		copy(cloned, groupIDs)
		return cloned, true
	}
	return nil, false
}

// IsMemberInGroup 检查特定成员是否在特定分组中 (辅助函数示例，可根据需要添加)
// 注意：此函数未在接口中定义，仅为内部或特定场景使用
func (gm *GroupManager) IsMemberInGroup(groupID string, memberID string) (bool, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	memberType, err := InferMemberTypeFromID(memberID)
	if err != nil {
		return false, fmt.Errorf("检查成员是否在分组中失败 (类型推断): %w", err)
	}

	if _, groupExists := gm.state.Groups[groupID]; !groupExists {
		return false, fmt.Errorf("分组 '%s' 不存在", groupID)
	}

	members, exists := gm.state.GroupToMembers[groupID]
	if !exists {
		return false, nil // 分组存在但没有成员列表
	}

	for _, m := range members {
		if m.ID == memberID && m.Type == memberType {
			return true, nil
		}
	}
	return false, nil
}

// ClearAllGroupData 清空所有分组数据 (测试或重置时可能用到)
// 警告: 此操作不可逆，将删除所有分组和成员关系
func (gm *GroupManager) ClearAllGroupData() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 重置内存状态
	gm.state = &GroupManagerState{
		Groups:         make(map[string]Group),
		MemberToGroups: make(map[string][]string),
		GroupToMembers: make(map[string][]Member),
	}

	// 持久化空状态
	if err := gm.persistStateToCache(); err != nil {
		// 即使持久化失败，内存状态也已清空。记录错误。
		logging.Error("清空所有分组数据后持久化空状态失败", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("清空所有分组数据: %w: %w", ErrPersistStateFailed, err)
	}

	// 清理相关缓存 (这是一个简化操作，实际可能需要更精细的模式匹配删除)
	// 注意: cache.RemovePluginCache 通常需要精确的键。
	// 如果缓存键有规律，可以遍历已知的分组ID和成员ID来删除。
	// 或者，如果 Cache 接口支持按前缀删除，会更方便。
	// 这里假设我们先获取所有旧的分组/成员键来尝试删除。
	// 但由于状态已清空，我们无法从 gm.state 获取这些键。
	// 因此，更稳妥的做法是 cache 包提供一个按前缀删除的功能，
	// 或者 GroupManager 在创建时记录所有使用过的缓存键前缀。
	// 作为简化的示例，我们仅删除状态本身的缓存。
	gm.cache.RemovePluginCache(gm.stateCacheKey)
	// 理想情况下，还需要删除 cachePrefixGroupByID, cachePrefixGroupByName 等键，
	// 但这需要知道所有曾经存在的 groupID 和 name，或者 Cache 支持前缀删除。

	logging.Info("所有分组数据已清空。")
	return nil
}
