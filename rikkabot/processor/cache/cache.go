// Package cache
// @Author Clover
// @Data 2024/7/11 上午12:05:00
// @Desc 缓存，持久化到文件中
package cache

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/logging"
	"wechat-demo/rikkabot/manager"
	"wechat-demo/rikkabot/processor/register"
)

type Cache struct {
	mu             sync.RWMutex
	*cacheExported // 隐藏字段
	config         *config.CommonConfig

	done chan struct{}
	wg   sync.WaitGroup
}

var (
	ErrPluginNotExist = errors.New("plugin not exist")
)

type cacheExported struct {
	WhiteUserIdSet  map[string]bool `json:"white_user_id_set"`  // 用户白名单
	BlackUserIdSet  map[string]bool `json:"black_user_id_set"`  // 用户黑名单
	WhiteGroupIdSet map[string]bool `json:"white_group_id_set"` // 群聊白名单
	BlackGroupIdSet map[string]bool `json:"black_group_id_set"` // 群聊黑名单
	AdminUserIdSet  map[string]bool `json:"admin_user_id_set"`  // 管理员名单 （不计入自己，自己默认管理员）
	EnablePlugins   map[string]bool `json:"enable_plugins"`     // 插件是否启用

	PluginsCache map[string]interface{} `json:"plugins_cache"` // 各个插件模块需要缓存的数据
}

// region Cache crud

// EnablePluginMap 获取插件状态列表 插件名-状态
func (c *Cache) EnablePluginMap() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cpEnablePlugins := make(map[string]bool, len(c.EnablePlugins))
	for k, v := range c.EnablePlugins {
		cpEnablePlugins[k] = v
	}
	return cpEnablePlugins
}

// 插件是否存在
func (c *Cache) isExistPlugin(pluginName string) bool {
	_, ok := c.EnablePlugins[pluginName]
	return ok
}

// EnablePlugin 启用插件
func (c *Cache) EnablePlugin(pluginName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isExistPlugin(pluginName) {
		return fmt.Errorf("%w: %s", ErrPluginNotExist, pluginName)
	}
	c.EnablePlugins[pluginName] = true
	return nil
}

// DisablePlugin 禁用插件
func (c *Cache) DisablePlugin(pluginName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isExistPlugin(pluginName) {
		return fmt.Errorf("%w: %s", ErrPluginNotExist, pluginName)
	}
	c.EnablePlugins[pluginName] = false
	return nil
}

// UploadCacheByPluginName region PluginsCache crud
func (c *Cache) UploadCacheByPluginName(pluginname string, dataCache interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.PluginsCache[pluginname] = dataCache
}

func (c *Cache) GetPluginCacheByName(pluginname string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.PluginsCache[pluginname]
}

func (c *Cache) RemovePluginCache(pluginname string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.PluginsCache, pluginname)
}

//endregion

// HasAdminUserId region Has
func (c *Cache) HasAdminUserId(userId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AdminUserIdSet[userId]
}

func (c *Cache) HasBlackUserId(userId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.BlackUserIdSet[userId]
}

func (c *Cache) HasWhiteUserId(userId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WhiteUserIdSet[userId]
}

func (c *Cache) HasBlackGroupId(groupId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.BlackGroupIdSet[groupId]
}

func (c *Cache) HasWhiteGroupId(groupId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WhiteGroupIdSet[groupId]
}

//endregion

// AddAdminUserId region Add
func (c *Cache) AddAdminUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AdminUserIdSet[userId] = true
}

func (c *Cache) AddBlackUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BlackUserIdSet[userId] = true
}

func (c *Cache) AddBlackGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BlackGroupIdSet[groupId] = true
}

func (c *Cache) AddWhiteUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WhiteUserIdSet[userId] = true
}

func (c *Cache) AddWhiteGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.WhiteGroupIdSet[groupId] = true
}

//endregion

// DeleteAdminUserId region Delete
func (c *Cache) DeleteAdminUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.AdminUserIdSet, userId)
}

func (c *Cache) DeleteBlackUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.BlackUserIdSet, userId)
}

func (c *Cache) DeleteBlackGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.BlackGroupIdSet, groupId)
}

func (c *Cache) DeleteWhiteUserId(userId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.WhiteUserIdSet, userId)
}

func (c *Cache) DeleteWhiteGroupId(groupId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.WhiteGroupIdSet, groupId)
}

//region GetList

// AdminIdList 获取所有管理员id
func (c *Cache) AdminIdList() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cnt := len(c.AdminUserIdSet)
	list := make([]string, 0, cnt)
	for k := range c.AdminUserIdSet {
		list = append(list, k)
	}
	return list
}

// WhiteGroupIdList 获取所有群组白名单id
func (c *Cache) WhiteGroupIdList() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cnt := len(c.WhiteGroupIdSet)
	list := make([]string, 0, cnt)
	for k := range c.WhiteGroupIdSet {
		list = append(list, k)
	}
	return list
}

// BlackGroupIdList 获取所有群组黑名单id
func (c *Cache) BlackGroupIdList() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cnt := len(c.BlackGroupIdSet)
	list := make([]string, 0, cnt)
	for k := range c.BlackGroupIdSet {
		list = append(list, k)
	}
	return list
}

// BlackUserIdList 获取所有用户黑名单id
func (c *Cache) BlackUserIdList() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cnt := len(c.BlackUserIdSet)
	list := make([]string, 0, cnt)
	for k := range c.BlackUserIdSet {
		list = append(list, k)
	}
	return list
}

//endregion

//endregion

// AdminUserIdSets 获取所有管理员包括状态
func (c *Cache) AdminUserIdSets() map[string]bool {
	c.mu.RLock()
	copyAdminUserIDs := make(map[string]bool, len(c.AdminUserIdSet))
	for k, v := range c.AdminUserIdSet {
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
	for name := range pluginPool.GetPluginMap() {
		_, ok := c.EnablePlugins[name]
		if ok {
			continue
		}
		c.EnablePlugins[name] = true
	}
}

// 定时持久化cache
func (c *Cache) cycleSave() {
	ticker := time.NewTicker(time.Duration(c.config.CacheSaveInterval) * time.Second)
	c.wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer c.wg.Done()
		for {
			select {
			case <-ticker.C:
				c.handleSave(false)
			case <-c.done: // 主动退出也保存
				c.handleSave(false)
				return
			}
		}
	}()
}

func (c *Cache) Close() {
	close(c.done)
	c.wg.Wait()
	logging.Info("cache closed")
}

func (c *Cache) handleSave(firstLoad bool) {
	c.mu.RLock()
	err := manager.SaveCache(cache)
	c.mu.RUnlock()
	if err != nil {
		logging.ErrorWithErr(err, "cycle save cache")
	}
	if firstLoad {
		logging.Info("load cache successfully")
	} else {
		logging.Debug("cache save successfully", nil)
	}
}

var cache *Cache

func initCache() {
	cache = &Cache{
		mu:     sync.RWMutex{},
		config: config.GetConfig(),
		cacheExported: &cacheExported{
			WhiteUserIdSet:  make(map[string]bool),
			BlackUserIdSet:  make(map[string]bool),
			WhiteGroupIdSet: make(map[string]bool),
			BlackGroupIdSet: make(map[string]bool),
			AdminUserIdSet:  make(map[string]bool),
			EnablePlugins:   make(map[string]bool),
			PluginsCache:    make(map[string]interface{}),
		},
		done: make(chan struct{}),
	}
}

func GetCache() *Cache {
	return cache
}

func Init() *Cache {
	initCache()
	// 初始化读取 Cache
	_, err := manager.LoadCache(cache)
	if err != nil {
		log.Warn().Err(err).Msg("load cache error")
	}
	logging.Debug("init read cache", map[string]interface{}{"cache": cache})

	// 同步新插件或者初始化插件状态
	cache.initEnablePlugins()
	cache.handleSave(true)

	// 启动独立线程定时持久化 cache
	cache.cycleSave()
	return cache
}
