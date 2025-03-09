// Package processor
// @Author Clover
// @Data 2025/3/11 下午8:17:00
// @Desc 模块处理器测试
package processor

import (
	"context"
	"testing"
	"time"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/plugins"
)

type pTestMatcher struct {
	isThought bool
}

func (m *pTestMatcher) Match(ctx context.Context, msg *message.Message) bool {
	return m.isThought
}

// TestPlugin 是一个真实的插件，用于测试
type TestPlugin struct {
	*plugins.Plugin // 嵌入标准的 Plugin 结构
	executeCount    int
}

func (p *TestPlugin) AddExecCount(delta int) {
	p.executeCount += delta
}

// NewTestPlugin 创建一个新的 TestPlugin 实例
func NewTestPlugin(name string, level plugins.PluginLevel) *TestPlugin {
	p := &TestPlugin{
		Plugin: plugins.DefaultPlugin(name).AsEnable(), // 使用默认的 Plugin 初始化 // 设置插件名称
	}
	p.Plugin.PluginOpt.Level = level // 设置插件级别
	return p
}

// TestProcessor_RegisterAndStart 测试插件注册和启动流程
func TestProcessor_RegisterAndStart(t *testing.T) {
	plugin := NewTestPlugin("test RegistAndStart-01", plugins.MediumLevel)
	plugin.AsAction(&plugins.ActionHandler{
		Name:     "test action",
		Matcher:  &pTestMatcher{true},
		IsEnable: true,
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
			plugin.AddExecCount(1)
			recvMsg.Content = "add 1 execute count"
			return *recvMsg, nil
		},
	})

	autoRegister := plugins.GetAutoRegister()
	autoRegister.RegisterPlugin(plugin)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	inputChan := make(chan *message.Message)    // 创建 inputChan
	sendChan := make(chan *message.Message, 10) // 发送通道
	processor := NewProcessor(ctx)              // 将 inputChan 传递给 NewProcessor

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
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
			highLevelPlugin.AddExecCount(1) // 实际不会执行到这里
			return *recvMsg, nil
		},
	})

	// 创建一个中优先级且匹配消息的插件
	mediumLevelPlugin := NewTestPlugin("test MessageFlow-01", plugins.MediumLevel)
	mediumLevelPlugin.AsAction(&plugins.ActionHandler{
		Name:     "test action",
		Matcher:  &pTestMatcher{true}, // Matcher 返回 true，此插件会执行
		IsEnable: true,
		Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
			mediumLevelPlugin.AddExecCount(1)
			recvMsg.Content = "add 1 execute count"
			return *recvMsg, nil
		},
	})

	autoRegister := plugins.GetAutoRegister()
	autoRegister.RegisterPlugin(highLevelPlugin)
	autoRegister.RegisterPlugin(mediumLevelPlugin)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	inputChan := make(chan *message.Message)
	sendChan := make(chan *message.Message, 10) // 发送通道
	processor := NewProcessor(ctx)

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
