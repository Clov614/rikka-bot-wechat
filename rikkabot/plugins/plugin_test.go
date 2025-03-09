package plugins

import (
	"context"
	"errors"
	matcher2 "github.com/Clov614/rikka-bot-wechat/rikkabot/plugins/matcher"
	"sync"
	"testing"
	"time"

	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
)

// MockMatcher 模拟的 Matcher，用于测试
type MockMatcher struct {
	matchResult bool
}

func (m *MockMatcher) Match(ctx context.Context, msg *message.Message) bool {
	return m.matchResult
}

// MockActionFunc 模拟的 ActionFunc，用于测试
type MockActionFunc func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error)

func TestActionHandler_doAction(t *testing.T) {
	tests := []struct {
		name        string
		action      MockActionFunc
		matcher     matcher2.Matcher
		recvMsg     *message.Message
		wantReplies []message.Message
		wantErr     error
		childTests  []struct {
			name        string
			action      MockActionFunc
			matcher     matcher2.Matcher
			recvMsg     *message.Message
			wantReplies []message.Message
			wantErr     error
		}
	}{
		{
			name: "Successful Action",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{Content: "reply from action"}, nil
			},
			matcher:     &MockMatcher{matchResult: true},
			recvMsg:     &message.Message{Content: "test message"},
			wantReplies: []message.Message{{Content: "reply from action"}},
			wantErr:     nil,
		},
		{
			name: "Action Returns Error",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{}, errors.New("action error")
			},
			matcher:     &MockMatcher{matchResult: true},
			recvMsg:     &message.Message{Content: "test message"},
			wantReplies: nil,
			wantErr:     errors.New("action error"),
		},
		{
			name: "Matcher Not Match",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{Content: "reply from action"}, nil
			},
			matcher:     &MockMatcher{matchResult: false},
			recvMsg:     &message.Message{Content: "test message"},
			wantReplies: nil, // Matcher 不匹配，不应执行 action，因此 replies 为 nil
			wantErr:     nil,
		},
		{
			name: "Nil RecvMsg",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{Content: "reply from action"}, nil
			},
			matcher:     &MockMatcher{matchResult: true},
			recvMsg:     nil,
			wantReplies: nil,
			wantErr:     ErrRecvMsgNull, // 接收消息为空错误
		},
		{
			name: "With Child Actions",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{Content: "reply from parent action"}, nil
			},
			matcher:     &MockMatcher{matchResult: true},
			recvMsg:     &message.Message{Content: "test message"},
			wantReplies: []message.Message{{Content: "reply from parent action"}, {Content: "reply from child action"}}, // 包含父子 action 的回复
			wantErr:     nil,
			childTests: []struct {
				name        string
				action      MockActionFunc
				matcher     matcher2.Matcher
				recvMsg     *message.Message
				wantReplies []message.Message
				wantErr     error
			}{
				{
					name: "Successful Child Action",
					action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
						return message.Message{Content: "reply from child action"}, nil
					},
					matcher:     &MockMatcher{matchResult: true},
					recvMsg:     &message.Message{Content: "test message"},
					wantReplies: []message.Message{{Content: "reply from child action"}},
					wantErr:     nil,
				},
			},
		},
		{
			name: "Child Action Returns Error",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{Content: "reply from parent action"}, nil
			},
			matcher:     &MockMatcher{matchResult: true},
			recvMsg:     &message.Message{Content: "test message"},
			wantReplies: nil, // 父 action 执行成功，但是子 action 错误，整体应该返回子 action 的错误
			wantErr:     errors.New("child action error"),
			childTests: []struct {
				name        string
				action      MockActionFunc
				matcher     matcher2.Matcher
				recvMsg     *message.Message
				wantReplies []message.Message
				wantErr     error
			}{
				{
					name: "Error Child Action",
					action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
						return message.Message{}, errors.New("child action error")
					},
					matcher:     &MockMatcher{matchResult: true},
					recvMsg:     &message.Message{Content: "test message"},
					wantReplies: nil,
					wantErr:     errors.New("child action error"),
				},
			},
		},
		{
			name: "Child Matcher Not Match",
			action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
				return message.Message{Content: "reply from parent action"}, nil
			},
			matcher:     &MockMatcher{matchResult: true},
			recvMsg:     &message.Message{Content: "test message"},
			wantReplies: []message.Message{{Content: "reply from parent action"}}, // 子 action 的 matcher 不匹配，不应执行子 action
			wantErr:     nil,
			childTests: []struct {
				name        string
				action      MockActionFunc
				matcher     matcher2.Matcher
				recvMsg     *message.Message
				wantReplies []message.Message
				wantErr     error
			}{
				{
					name: "Not Match Child Action",
					action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
						return message.Message{Content: "reply from child action"}, nil
					},
					matcher:     &MockMatcher{matchResult: false}, // 子 action 的 matcher 不匹配
					recvMsg:     &message.Message{Content: "test message"},
					wantReplies: nil,
					wantErr:     nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah := &ActionHandler{
				Name:    tt.name,
				Matcher: tt.matcher,
				Action:  ActionFunc(tt.action),
			}
			for _, childTest := range tt.childTests {
				childAH := &ActionHandler{
					Name:    childTest.name,
					Matcher: childTest.matcher,
					Action:  ActionFunc(childTest.action),
				}
				ah.AsChild(childAH) // 添加子 action
			}

			replies, err := ah.doAction(context.Background(), tt.recvMsg)

			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("ActionHandler.doAction() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			} else if err != nil {
				t.Errorf("ActionHandler.doAction() unexpected error = %v", err)
				return
			}

			if len(replies) != len(tt.wantReplies) {
				t.Errorf("ActionHandler.doAction() replies length = %v, want %v", len(replies), len(tt.wantReplies))
				return
			}

			for i, reply := range replies {
				if reply.Content != tt.wantReplies[i].Content {
					t.Errorf("ActionHandler.doAction() replies[%d] = %v, want %v", i, reply.Content, tt.wantReplies[i].Content)
				}
			}
		})
	}
}

func TestPlugin_HandleRecv(t *testing.T) {
	tests := []struct {
		name            string
		plugin          *Plugin
		recvMsg         *message.Message
		expectExecute   bool
		expectedReplies []string
	}{
		{
			name: "Plugin Enabled, Action Match",
			plugin: func() *Plugin {
				p := DefaultPlugin("Plugin Enabled, Action Match")
				p.Enable = true
				p.ActionHandlerList = []*ActionHandler{
					{
						Matcher: &MockMatcher{matchResult: true},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 1"}, nil
						},
					},
				}
				return p
			}(),
			recvMsg:         &message.Message{Content: "test message"},
			expectExecute:   true,
			expectedReplies: []string{"reply from action 1"},
		},
		{
			name: "Plugin Disabled",
			plugin: func() *Plugin {
				p := DefaultPlugin("Plugin Disabled")
				p.Enable = false // 插件禁用
				p.ActionHandlerList = []*ActionHandler{
					{
						Matcher: &MockMatcher{matchResult: true},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 1"}, nil
						},
					},
				}
				return p
			}(),
			recvMsg:         &message.Message{Content: "test message"},
			expectExecute:   false, // 插件禁用，不应执行
			expectedReplies: nil,
		},
		{
			name: "Action Not Match",
			plugin: func() *Plugin {
				p := DefaultPlugin("Action Not Match")
				p.Enable = true
				p.ActionHandlerList = []*ActionHandler{
					{
						Matcher: &MockMatcher{matchResult: false}, // action 不匹配
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 1"}, nil
						},
					},
				}
				return p
			}(),
			recvMsg:         &message.Message{Content: "test message"},
			expectExecute:   false, // 插件本身会执行，但是 action 不匹配
			expectedReplies: nil,
		},
		{
			name: "Multiple Actions, Some Match",
			plugin: func() *Plugin {
				p := DefaultPlugin("Multiple Actions, Some Match")
				p.Enable = true
				p.ActionHandlerList = []*ActionHandler{
					{
						Matcher: &MockMatcher{matchResult: false},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 1"}, nil
						},
					},
					{
						Matcher: &MockMatcher{matchResult: true},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 2"}, nil
						},
					},
					{
						Matcher: &MockMatcher{matchResult: false},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 3"}, nil
						},
					},
				}
				return p
			}(),
			recvMsg:         &message.Message{Content: "test message"},
			expectExecute:   true,
			expectedReplies: []string{"reply from action 2"}, // 只有 action 2 匹配
		},
		{
			name: "Multiple Actions, All Match", // todo test 暂未想到如何解决顺序问题
			plugin: func() *Plugin {
				p := DefaultPlugin("Multiple Actions, All Match")
				p.Enable = true
				p.ActionHandlerList = []*ActionHandler{
					{
						Matcher: &MockMatcher{matchResult: true},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 1"}, nil
						},
					},
					{
						Matcher: &MockMatcher{matchResult: true},
						Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
							return message.Message{Content: "reply from action 2"}, nil
						},
					},
				}
				return p
			}(),
			recvMsg:         &message.Message{Content: "test message"},
			expectExecute:   true,
			expectedReplies: []string{"reply from action 1", "reply from action 2"}, // 两个 action 都匹配
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendChan := make(chan *message.Message, 10) // 创建一个带缓冲的 channel
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			execute := tt.plugin.HandleRecv(ctx, tt.recvMsg, sendChan)

			if execute != tt.expectExecute {
				t.Errorf("Plugin.HandleRecv() execute = %v, want %v", execute, tt.expectExecute)
			}

			// 等待一段时间，确保所有 goroutine 完成发送消息
			time.Sleep(time.Millisecond * 50)

			var receivedReplies []string
			close(sendChan) // 关闭 channel 以结束 range 循环
			for msg := range sendChan {
				receivedReplies = append(receivedReplies, msg.Content)
			}

			if len(receivedReplies) != len(tt.expectedReplies) {
				t.Errorf("Plugin.HandleRecv() received replies length = %v, want %v", len(receivedReplies), len(tt.expectedReplies))
			} else {
				for i, reply := range receivedReplies {
					if reply != tt.expectedReplies[i] {
						t.Errorf("Plugin.HandleRecv() received reply[%d] = %v, want %v", i, reply, tt.expectedReplies[i])
					}
				}
			}
			tt.plugin.Close() // 关闭插件，等待所有 action 完成
		})
	}
}

func TestPlugin_Close(t *testing.T) {
	t.Run("Close Plugin and Wait Actions", func(t *testing.T) {
		p := DefaultPlugin("Close Plugin and Wait Actions")
		p.Enable = true
		var actionExecuted sync.WaitGroup
		actionExecuted.Add(1)
		p.ActionHandlerList = []*ActionHandler{
			{
				Matcher: &MockMatcher{matchResult: true},
				Action: func(ctx context.Context, recvMsg *message.Message) (reply message.Message, err error) {
					defer actionExecuted.Done()
					time.Sleep(time.Millisecond * 100) // 模拟 action 执行时间
					return message.Message{Content: "reply from action"}, nil
				},
			},
		}

		sendChan := make(chan *message.Message, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go p.HandleRecv(ctx, &message.Message{Content: "test message"}, sendChan)
		closeStart := time.Now()
		time.Sleep(time.Millisecond * 10)

		p.Close() // 关闭插件
		closeDuration := time.Since(closeStart)

		if closeDuration < time.Millisecond*110 { // Close 方法应该至少等待 action 执行完成
			t.Errorf("Plugin.Close() returned too quickly, waited only %v, expected to wait for action to complete (at least 100ms)", closeDuration)
		}

		actionExecuted.Wait() // 确保 action 确实执行完成
	})
}

//func TestAutoRegister_RegisterAndGet(t *testing.T) {
//	ar := GetAutoRegister()
//	ar.pluginLevelList = make([]map[string]IPlugin, LevelSize) // reset plugin list for test isolation
//	ar.size = 0
//	ar.enableSize = 0
//
//	plugin1 := DefaultPlugin("plugin1")
//	plugin1.Name = "plugin1"
//	plugin1.Level = HighLevel
//	plugin1.Enable = true
//
//	plugin2 := DefaultPlugin("plugin2")
//	plugin2.Name = "plugin2"
//	plugin2.Level = MediumLevel
//	plugin2.Enable = false
//
//	ar.RegisterPlugin(plugin1)
//	ar.RegisterPlugin(plugin2)
//
//	if ar.size != 2 {
//		t.Errorf("AutoRegister.RegisterPlugin() size = %v, want %v", ar.size, 2)
//	}
//	if ar.enableSize != 1 {
//		t.Errorf("AutoRegister.RegisterPlugin() enableSize = %v, want %v", ar.enableSize, 1)
//	}
//
//	levelPlugins := ar.Get(HighLevel)
//	if len(levelPlugins) != 1 || levelPlugins[0].Name != "plugin1" {
//		t.Errorf("AutoRegister.Get(HighLevel) = %v, want [{Name:plugin1}]", levelPlugins)
//	}
//
//	allPlugins := ar.Plugins()
//	if len(allPlugins[HighLevel]) != 1 || allPlugins[HighLevel][0].Name != "plugin1" {
//		t.Errorf("AutoRegister.Plugins()[HighLevel] = %v, want [{Name:plugin1}]", allPlugins[HighLevel])
//	}
//	if len(allPlugins[MediumLevel]) != 1 || allPlugins[MediumLevel][0].Name != "plugin2" {
//		t.Errorf("AutoRegister.Plugins()[MediumLevel] = %v, want [{Name:plugin2}]", allPlugins[MediumLevel])
//	}
//}
//
//func TestAutoRegister_EnableDisableByName(t *testing.T) {
//	ar := GetAutoRegister()
//	ar.pluginLevelList = make([]map[string]*Plugin, LevelSize) // reset plugin list for test isolation
//	ar.size = 0
//	ar.enableSize = 0
//
//	plugin1 := DefaultPlugin()
//	plugin1.Name = "plugin1"
//	plugin1.Level = HighLevel
//	plugin1.Enable = false
//	ar.RegisterPlugin(plugin1)
//
//	plugin2 := DefaultPlugin()
//	plugin2.Name = "plugin2"
//	plugin2.Level = MediumLevel
//	plugin2.Enable = true
//	ar.RegisterPlugin(plugin2)
//
//	// Disable plugin2
//	if !ar.DisableByName("plugin2") {
//		t.Errorf("AutoRegister.DisableByName(\"plugin2\") should return true")
//	}
//	if ar.enableSize != 0 {
//		t.Errorf("AutoRegister.DisableByName(\"plugin2\") enableSize = %v, want %v", ar.enableSize, 0)
//	}
//	if ar.pluginLevelList[MediumLevel]["plugin2"].Enable {
//		t.Errorf("AutoRegister.DisableByName(\"plugin2\") plugin2.Enable should be false")
//	}
//
//	// Enable plugin1
//	if !ar.EnableByName("plugin1") {
//		t.Errorf("AutoRegister.EnableByName(\"plugin1\") should return true")
//	}
//	if ar.enableSize != 1 {
//		t.Errorf("AutoRegister.EnableByName(\"plugin1\") enableSize = %v, want %v", ar.enableSize, 1)
//	}
//	if !ar.pluginLevelList[HighLevel]["plugin1"].Enable {
//		t.Errorf("AutoRegister.EnableByName(\"plugin1\") plugin1.Enable should be true")
//	}
//
//	// Disable non-existent plugin
//	if ar.DisableByName("plugin3") {
//		t.Errorf("AutoRegister.DisableByName(\"plugin3\") should return false for non-existent plugin")
//	}
//
//	// Enable non-existent plugin
//	if ar.EnableByName("plugin3") {
//		t.Errorf("AutoRegister.EnableByName(\"plugin3\") should return false for non-existent plugin")
//	}
//}
