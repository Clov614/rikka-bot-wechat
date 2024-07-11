// @Author Clover
// @Data 2024/7/11 上午12:05:00
// @Desc 缓存，持久化到文件中
package cache

import (
	"fmt"
	"sync"
	"time"
	"wechat-demo/rikkabot/processor/register"
	"wechat-demo/rikkabot/utils/serializer"
)

type Cache struct {
	mu              sync.RWMutex
	whiteUserIdSet  map[string]bool `json:"white_user_id_set"`  // 用户白名单
	blackUserIdSet  map[string]bool `json:"black_user_id_set"`  // 用户黑名单
	whiteGroupIdSet map[string]bool `json:"white_group_id_set"` // 群聊白名单
	blackGroupIdSet map[string]bool `json:"black_group_id_set"` // 群聊黑名单
	adminUserIdSet  map[string]bool `json:"admin_user_id_set"`  // 管理员名单 （不计入自己，自己默认管理员）
	enablePlugins   map[string]bool `json:"enable_plugins"`     // 插件是否启用

	done chan struct{} `json:"-" yaml:"-"`
	wg   sync.WaitGroup
}

// region Cache crud

// region Has
func (c *Cache) HasAdminUserId(userId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.adminUserIdSet[userId]
}

func (c *Cache) HasBlackUserId(userId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.blackUserIdSet[userId]
}

func (c *Cache) HasWhiteUserId(userId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.whiteUserIdSet[userId]
}

func (c *Cache) HasBlackGroupId(groupId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.blackGroupIdSet[groupId]
}

func (c *Cache) HasWhiteGroupId(groupId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.whiteGroupIdSet[groupId]
}

//endregion

// region Add
func (c *Cache) AddAdminUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.adminUserIdSet[userId] = true
}

func (c *Cache) AddBlackUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blackUserIdSet[userId] = true
}

func (c *Cache) AddBlackGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blackGroupIdSet[groupId] = true
}

func (c *Cache) AddWhiteUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.whiteUserIdSet[userId] = true
}

func (c *Cache) AddWhiteGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.whiteGroupIdSet[groupId] = true
}

//endregion

// region Delete
func (c *Cache) DeleteAdminUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.adminUserIdSet, userId)
}

func (c *Cache) DeleteBlackUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.blackUserIdSet, userId)
}

func (c *Cache) DeleteBlackGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.blackGroupIdSet, groupId)
}

func (c *Cache) DeleteWhiteUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.whiteUserIdSet, userId)
}

func (c *Cache) DeleteWhiteGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.whiteGroupIdSet, groupId)
}

//endregion

// todo 获取所有管理员可以使用
func (c *Cache) AdminUserIdSets() map[string]bool {
	c.mu.RLock()
	copyAdminUserIDs := make(map[string]bool, len(c.adminUserIdSet))
	for k, v := range c.adminUserIdSet {
		copyAdminUserIDs[k] = v
	}
	c.mu.RUnlock()
	return copyAdminUserIDs
}

//endregion

// 同步新插入的插件/初始化载入插件信息
func (c *Cache) initEnablePlugins() {
	c.mu.Lock()
	defer c.mu.Unlock()
	pluginPool := register.GetPluginPool()
	for name, _ := range pluginPool.GetPluginMap() {
		_, ok := c.enablePlugins[name]
		if ok {
			continue
		}
		c.enablePlugins[name] = true
	}
}

// 定时持久化cache
func (c *Cache) cycleSave() {
	ticker := time.NewTicker(1 * time.Minute) // todo 通过设置项，支持外部更改 更新频率
	c.wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer c.wg.Done()
		for {
			select {
			case <-ticker.C:
				c.handleSave()
			case <-c.done: // 主动退出也保存
				c.handleSave()
				return
			}
		}
	}()
}

func (c *Cache) Close() {
	close(c.done)
	c.wg.Wait()
}

func (c *Cache) handleSave() {
	c.mu.RLock()
	err := serializer.Save(cachePath, cacheFilename, cache)
	c.mu.RUnlock()
	if err != nil {
		fmt.Println("cycle save cache error:", err)
	}
}

// todo 临时的，为了避免 plugin引用导致注册器无法正确注册，实现将废弃
var cache *Cache

func initCache() {
	cache = &Cache{
		mu:              sync.RWMutex{},
		whiteUserIdSet:  make(map[string]bool),
		blackUserIdSet:  make(map[string]bool),
		whiteGroupIdSet: make(map[string]bool),
		blackGroupIdSet: make(map[string]bool),
		adminUserIdSet:  make(map[string]bool),
		enablePlugins:   make(map[string]bool),
		done:            make(chan struct{}),
	}
}

// deprecated
func GetCache() *Cache {
	return cache
}

func Init() *Cache {
	initCache()
	// 初始化读取 Cache
	if serializer.IsPathExist(cachePath, cacheFilename) {
		err := serializer.Load(cachePath, cacheFilename, cache)
		if err != nil {
			fmt.Println("load cache error:", err)
		}
	}

	// 同步新插件或者初始化插件状态
	cache.initEnablePlugins()
	cache.handleSave()

	// 启动独立线程定时持久化 cache
	cache.cycleSave()
	return cache
}

const (
	cachePath     = "./db"
	cacheFilename = "cache"
)
