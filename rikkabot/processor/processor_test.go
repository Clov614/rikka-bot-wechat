// Package processor
// @Author Clover
// @Data 2025/3/11 下午8:17:00
// @Desc 模块处理器测试
package processor

import (
	"context"
	"testing"
	"time"

	wcf "github.com/Clov614/wcf-rpc-sdk"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/processor/cache"
)

// 定义一个专门用于测试的 TestPlugin 结构体
type TestPlugin struct {
	*plugins.Plugin     // 嵌入标准的 Plugin 结构
	executeCount    int // 执行计数器
}

// AddExecCount 增加执行计数
func (p *TestPlugin) AddExecCount(delta int) {
	p.executeCount += delta
}

// NewTestPlugin 创建一个新的 TestPlugin 实例
func NewTestPlugin(name string, level plugins.PluginLevel) *TestPlugin {
	p := &TestPlugin{
		Plugin: plugins.DefaultPlugin(name).AsEnable(), // 使用默认的 Plugin 初始化并启用
	}
	p.Plugin.PluginOpt.Level = level // 设置插件级别
	return p
}

type pTestMatcher struct {
	isThought bool
}

func (m *pTestMatcher) Match(ctx context.Context, msg *message.Message) bool {
	return m.isThought
}

// TestProcessor_RegisterAndStart 测试插件注册和启动流程
func TestProcessor_RegisterAndStart(t *testing.T) {
	plugin := NewTestPlugin("test RegistAndStart-01", plugins.MediumLevel)
	plugin.AsAction(&plugins.ActionHandler{
		Name:     "test action",
		Matcher:  &pTestMatcher{true},
		IsEnable: true,
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
			plugin.AddExecCount(1)
			recvMsg.Content = "add 1 execute count"
			return *recvMsg, true, nil
		},
	})

	autoRegister := plugins.GetAutoRegister()
	autoRegister.RegisterPlugin(plugin)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	inputChan := make(chan *message.Message)                        // 创建 inputChan
	sendChan := make(chan *message.Message, 10)                     // 发送通道
	processor := NewProcessor(ctx, wcf.NewClient(10, false, false)) // 将 inputChan 传递给 NewProcessor

	// 启动 Processor
	processor.Start(inputChan, sendChan)
	// 验证插件是否成功注册
	registeredPlugins := processor.LevelLayer[plugins.MediumLevel].GetPlugins()
	if len(registeredPlugins) != 1 || (*registeredPlugins[0]).GetName() != "test RegistAndStart-01" {
		t.Errorf("Expected 1 registered plugin with name 'test RegistAndStart-01', got: %v", registeredPlugins)
	}

	// 发送消息，触发插件执行
	testMsg := &message.Message{Content: "test message"}
	inputChan <- testMsg

	// 等待一段时间，让插件有机会处理消息
	// 实际项目中，这里可能需要根据插件的处理逻辑进行调整
	<-time.After(time.Second * 1)

	processor.Close()
	// 验证插件的 Action 是否被执行
	if plugin.executeCount != 1 {
		t.Errorf("Expected plugin executeCount to be 1, got: %d", plugin.executeCount)
	}
	if testMsg.Content != "add 1 execute count" {
		t.Errorf("Expected message content change, got: %s", testMsg.Content)
	}
	for outputMsg := range sendChan { // 发送通道消息输出
		t.Logf("Sending message: %s", outputMsg.Content)
	}
}

// TestProcessor_MessageFlow 测试消息在 Processor 中的流动
func TestProcessor_MessageFlow(t *testing.T) {
	// 创建一个高优先级但不匹配任何消息的插件
	highLevelPlugin := NewTestPlugin("test HighLevel-01", plugins.HighLevel)
	highLevelPlugin.AsAction(&plugins.ActionHandler{
		Name:     "high level action",
		Matcher:  &pTestMatcher{false}, // Matcher 返回 false，此插件不会执行
		IsEnable: true,
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
			highLevelPlugin.AddExecCount(1) // 实际不会执行到这里
			return *recvMsg, true, nil
		},
	})

	// 创建一个中优先级且匹配消息的插件
	mediumLevelPlugin := NewTestPlugin("test MessageFlow-01", plugins.MediumLevel)
	mediumLevelPlugin.AsAction(&plugins.ActionHandler{
		Name:     "test action",
		Matcher:  &pTestMatcher{true}, // Matcher 返回 true，此插件会执行
		IsEnable: true,
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
			mediumLevelPlugin.AddExecCount(1)
			recvMsg.Content = "add 1 execute count"
			return *recvMsg, true, nil
		},
	})

	autoRegister := plugins.GetAutoRegister()
	autoRegister.RegisterPlugin(highLevelPlugin)
	autoRegister.RegisterPlugin(mediumLevelPlugin)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	inputChan := make(chan *message.Message)
	sendChan := make(chan *message.Message, 10) // 发送通道
	processor := NewProcessor(ctx, wcf.NewClient(10, false, false))

	processor.Start(inputChan, sendChan)

	testMsg := &message.Message{Content: "test message"}
	inputChan <- testMsg

	<-time.After(time.Second * 1)

	processor.Close()

	// 验证高优先级插件未执行
	if highLevelPlugin.executeCount != 0 {
		t.Errorf("Expected highLevelPlugin executeCount to be 0, got: %d", highLevelPlugin.executeCount)
	}
	// 验证中优先级插件已执行
	if mediumLevelPlugin.executeCount != 1 {
		t.Errorf("Expected mediumLevelPlugin executeCount to be 1, got: %d", mediumLevelPlugin.executeCount)
	}
	if testMsg.Content != "add 1 execute count" {
		t.Errorf("Expected message content change, got: %s", testMsg.Content)
	}
	for outputMsg := range sendChan { // 发送通道消息输出
		t.Logf("Sending message: %s", outputMsg.Content)
	}
}

// TestProcessor_CloseAndCache 测试 Processor 的 Close 方法、cache 的联动以及重启后的加载
func TestProcessor_CloseAndCache(t *testing.T) {
	// 准备阶段：创建并注册插件
	pluginName := "testCloseAndCache-01"
	plugin := NewTestPlugin(pluginName, plugins.MediumLevel)
	plugin.AsAction(&plugins.ActionHandler{
		Name:    "test action",
		Matcher: &pTestMatcher{true},
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, ok bool, err error) {
			plugin.AddExecCount(1) // 增加执行计数
			return *recvMsg, true, nil
		},
	})
	plugins.GetAutoRegister().RegisterPlugin(plugin) // 直接获取并注册

	// 阶段 1: 首次运行并关闭，测试缓存
	func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		inputChan := make(chan *message.Message)
		sendChan := make(chan *message.Message, 10)
		processor := NewProcessor(ctx, wcf.NewClient(10, false, false))

		processor.Start(inputChan, sendChan)
		inputChan <- &message.Message{Content: "test message"} // 触发插件执行
		<-time.After(time.Second * 1)                          // 等待执行

		processor.Close() // 关闭并触发缓存

		// 验证插件执行次数和缓存状态
		if plugin.executeCount != 1 {
			t.Errorf("Expected plugin executeCount to be 1, got: %d", plugin.executeCount)
		}
		cachedPlugin, ok := cache.GetCache().GetPluginInfo(pluginName)
		if !ok {
			t.Fatalf("Plugin %s not found in cache", pluginName)
		}
		// 验证缓存的是 PluginOpt
		cachedOpt, ok := cachedPlugin.(plugins.PluginOpt)
		if !ok {
			t.Fatalf("Expected cached plugin info to be PluginOpt, got: %T", cachedPlugin)
		}
		if !cachedOpt.Enable {
			t.Errorf("Expected plugin %s to be enabled in cache", pluginName)
		}
	}() // 使用匿名函数隔离作用域

	// 阶段 2: 重启并验证加载
	func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		inputChan := make(chan *message.Message)
		sendChan := make(chan *message.Message, 10)
		processor := NewProcessor(ctx, wcf.NewClient(10, false, false))

		// 注意：这里不需要重新注册插件，因为 AutoRegister 会从 cache 加载
		processor.Start(inputChan, sendChan)

		// 验证插件已从 cache 加载
		registeredPlugins := processor.LevelLayer[plugins.MediumLevel].GetPlugins()
		// 遍历找到我们的测试插件
		var plugin2 *TestPlugin
		for _, p := range registeredPlugins {
			if (*p).GetName() == pluginName {
				var ok bool
				plugin2, ok = (*p).(*TestPlugin)
				if !ok {
					t.Fatalf("Expected plugin '%s' to be *TestPlugin, got: %T", pluginName, *p)
				}
				break
			}
		}

		if plugin2 == nil {
			t.Fatalf("Plugin '%s' not found in registered plugins", pluginName)
		}

		// 从缓存中恢复
		cachedOpt, ok := cache.GetCache().GetPluginInfo(pluginName)
		if !ok {
			t.Fatalf("Plugin %s not found in cache", pluginName)
		}
		plugin2.PluginOpt = cachedOpt.(plugins.PluginOpt)

		if !plugin2.PluginOpt.Enable {
			t.Errorf("Expected plugin %s to be enabled after restart", pluginName)
		}

		// 再次触发插件执行，验证其功能
		inputChan <- &message.Message{Content: "test message"}
		<-time.After(time.Second * 1)
		if plugin2.executeCount != 2 { // 执行次数应为 2
			t.Errorf("Expected plugin executeCount to be 2, got: %d", plugin2.executeCount)
		}

		processor.Close() // 关闭
	}() // 使用匿名函数隔离作用域
}
