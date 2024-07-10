// @Author Clover
// @Data 2024/7/6 下午8:24:00
// @Desc 全局处理器
package processor

import (
	"fmt"
	"sync"
	"time"
	"wechat-demo/rikkabot/message"
	"wechat-demo/rikkabot/processor/control"
	"wechat-demo/rikkabot/processor/register"
	"wechat-demo/rikkabot/utils/serializer"
)

type Processor struct {
	*Cache     // 处理器缓存
	pluginPool *register.PluginRegister

	mu           sync.RWMutex
	longConnPool map[chan message.Message]chan struct{} // 长连接池（保存消息接收通道）
	// todo 方法 消息/群号 锁 保证对话发起者只能拥有一个插件的对话存活
}

func NewProcessor() *Processor {
	return &Processor{
		Cache:        cache,
		pluginPool:   register.GetPluginPool(),
		longConnPool: make(map[chan message.Message]chan struct{}),
	}
}

func (p *Processor) DispatchMsg(recvChan chan *message.Message, sendChan chan *message.Message) {
	for msg := range recvChan {
		go p.broadcastRecv(*msg) // 长连接分发消息
		pluginMap := p.pluginPool.GetPluginMap()
		for name, plugin := range pluginMap {
			dialog := plugin.(control.IDialog)
			if p.isEnable(name) && p.isHandle(dialog.GetProcessRules(), msg) { // 是否满足规则
				recvConn := make(chan message.Message)
				done := make(chan struct{})
				p.registLongConn(recvConn, done)
				go func() { recvConn <- *msg }()
				go dialog.RunPlugin(sendChan, recvConn, done)
			}
		}
	}
}

func (p *Processor) broadcastRecv(recvMsg message.Message) {
	for c, d := range p.getLongconns() {
		conn := c
		done := d
		go func() {
			select {
			case conn <- recvMsg:
				// do send msg
			case <-done: // 对话已关闭
				p.unregistLongconn(conn)
			default:
				// skip send
			}
		}()
	}
}

func (p *Processor) registLongConn(recvChan chan message.Message, done chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.longConnPool[recvChan] = done
}

func (p *Processor) getLongconns() map[chan message.Message]chan struct{} {
	copyconnPool := make(map[chan message.Message]chan struct{})
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.longConnPool {
		copyconnPool[k] = v
	}
	return copyconnPool
}

func (p *Processor) unregistLongconn(recvChan chan message.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.longConnPool, recvChan)
}

type Cache struct {
	mu             sync.RWMutex
	WhiteUserIdSet map[string]bool // 用户白名单
	BlackUserIdSet map[string]bool // 用户黑名单

	WhiteGroupIdSet map[string]bool // 群聊白名单
	BlackGroupIdSet map[string]bool // 群聊黑名单

	AdminUserIdSet map[string]bool // 管理员名单 （不计入自己，自己默认管理员）

	EnablePlugins map[string]bool // 插件是否启用

	done chan struct{} `json:"-" yaml:"-"`
}

// region Cache crud

// region Has
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

// region Add
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

// region Delete
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

//endregion

// todo 获取所有管理员可以使用
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
	for name, _ := range pluginPool.GetPluginMap() {
		_, ok := c.EnablePlugins[name]
		if ok {
			continue
		}
		c.EnablePlugins[name] = true
	}
}

// 定时持久化cache
func (c *Cache) cycleSave() {
	ticker := time.NewTicker(1 * time.Minute) // todo 通过设置项，支持外部更改 更新频率
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.handleSave()
			case <-c.done: // 主动退出也保存
				c.handleSave()
			}
		}
	}()
}

func (c *Cache) Close() {
	close(c.done)
}

func (c *Cache) handleSave() {
	c.mu.RLock()
	err := serializer.Save(cachePath, cacheFilename, cache)
	c.mu.RUnlock()
	if err != nil {
		fmt.Println("cycle save cache error:", err)
	}
}

var cache *Cache = &Cache{
	mu:              sync.RWMutex{},
	WhiteUserIdSet:  make(map[string]bool),
	BlackUserIdSet:  make(map[string]bool),
	WhiteGroupIdSet: make(map[string]bool),
	BlackGroupIdSet: make(map[string]bool),
	AdminUserIdSet:  make(map[string]bool),
	EnablePlugins:   make(map[string]bool),
	done:            make(chan struct{}),
}

const (
	cachePath     = "./db"
	cacheFilename = "cache"
)

func init() {
	// 初始化读取 Cache
	if serializer.IsPathExist(cachePath, cacheFilename) {
		err := serializer.Load(cachePath, cacheFilename, cache)
		if err != nil {
			fmt.Println("load cache error:", err)
		}
	}

	// 同步新插件或者初始化插件状态
	cache.initEnablePlugins()

	// 启动独立线程定时持久化 cache
	cache.cycleSave()
	defer cache.Close()
}
