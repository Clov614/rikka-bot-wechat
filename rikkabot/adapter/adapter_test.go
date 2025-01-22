// @Author Clover
// @Data 2024/7/5 下午6:09:00
// @Desc 适配器模块测试
package adapter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"math/rand"
	"strings"

	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/serializer"
	wcf "github.com/Clov614/wcf-rpc-sdk"
	"github.com/google/uuid"
)

// 测试是否转发接收到的消息，二次封装是否正常
func TestReceiveRawMsg(t *testing.T) {
	runBase(t, func(a *Adapter, done chan struct{}) error {
		recvChan := a.rikkaBot.GetReqMsgRecvChan()
		for {
			select {
			case <-done:
				return nil
			case rikkaMsg := <-recvChan:
				if rikkaMsg.Msgtype == message.MsgTypeText {
					if rikkaMsg.Content != "test" {
						return fmt.Errorf("TestReceiveRawMsg faild")
					}
					return nil
				}
			}
		}
	})
}

// 测试rikkaBot发送的消息是否能转发给 openwechat
func TestSendRikkaMsg(t *testing.T) {
	runBase(t, func(a *Adapter, done chan struct{}) error {
		var sendMsg *message.Message
		select {
		case <-done:
			return nil
		case msg := <-a.rikkaBot.GetReqMsgRecvChan():
			if msg.Content != "test" {
				t.Errorf("TestSendRikkaMsg faild")
			}
			sendMsg = msg
			sendMsg.Content = "got it testing!"
		}
		a.rikkaBot.GetRespMsgSendChan() <- sendMsg
		return nil
	})
}

// 获取 第三方平台发送来的经转换后的消息json，保存于同级目录下
func TestGetMsgJson(t *testing.T) {
	runBase(t, func(a *Adapter, done chan struct{}) error {
		func() {
			for {
				select {
				case <-done:
					return
				case msg := <-a.cli.GetMsgChan():
					rikkaMsg := a.covert(msg)
					fmt.Printf("save json recv rikkaMsg: %#v\n", *rikkaMsg)
					err := serializer.Save("./test/cacheMsg", fmt.Sprintf("rikkaMsg%s", uuid.New().String()), rikkaMsg)
					if err != nil {
						t.Logf("err: %v", err)
					}
					return
				}
			}
		}()
		return nil
	})
}

// 测试发送xx消息
func TestSendMsg(t *testing.T) {
	runBase(t, echo) // t  测试函数
}

// 测试群聊和个人发送的id解析
func TestMsgSenderId(t *testing.T) {
	runBase(t, GetUserId)
	// Output:
	//groupId= 813467281
	//senderId = 813466966
	//receviceId = 2788092443

	//groupId= 813467281
	//senderId = 2788092443
	//receviceId = 2788092443

	// 大号: 813466966
	// 小号: 2788092443
	// 群号: 813467281
}

func TestAtAnalys(t *testing.T) {
	runBase(t, GetmsgAtAnalysis)
}

func GetmsgAtAnalysis(a *Adapter, done chan struct{}) error {
	recvChan := a.rikkaBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			_ = rikkaMsg.Content
			self := a.cli.GetSelfInfo()
			//rawmsg := rikkaMsg.MetaData.GetRawMsg().(*openwechat.Message)

			fmt.Printf("self: %#v\n", self)

			rawMsg := rikkaMsg.MetaData.GetRawMsg().(*wcf.Message)

			fmt.Printf("rawMsg: %#v\n", rawMsg)
		}
	}
}

// 测试各种消息的唯一ID
func GetUserId(a *Adapter, done chan struct{}) error {
	recvChan := a.rikkaBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			if rikkaMsg.Msgtype == message.MsgTypeText {
				if rikkaMsg.IsGroup { // 群消息返回
					rikkaMsg.Content = fmt.Sprintf("wxId= %s \n roomId = %s \n", rikkaMsg.WxId, rikkaMsg.RoomId)
					rikkaMsg.Raw = []byte(rikkaMsg.Content)
					a.rikkaBot.GetRespMsgSendChan() <- rikkaMsg // 原路返回消息
				} else {
					rikkaMsg.Content = fmt.Sprintf("wxId = %s",
						rikkaMsg.WxId)
					rikkaMsg.Raw = []byte(rikkaMsg.Content)
					a.rikkaBot.GetRespMsgSendChan() <- rikkaMsg // 原路返回消息
				}
			}
		}
	}
}

func echo(a *Adapter, done chan struct{}) error {
	recvChan := a.rikkaBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			context := rikkaMsg.Content
			if strings.HasPrefix(context, "echo ") {
				trimed := strings.TrimPrefix(context, "echo ")
				rikkaMsg.Content = trimed
				rikkaMsg.Raw = []byte(rikkaMsg.Content)
				a.rikkaBot.GetRespMsgSendChan() <- rikkaMsg
			}
		}
	}
}

func doubleEcho(a *Adapter, done chan struct{}) error {
	recvChan := a.rikkaBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:

			context := rikkaMsg.Content
			if strings.HasPrefix(context, "echo ") {
				trimed := strings.TrimPrefix(context, "echo ")
				rikkaMsg.Content = trimed
				rikkaMsg.Raw = []byte(rikkaMsg.Content)
				a.rikkaBot.GetRespMsgSendChan() <- rikkaMsg
				a.rikkaBot.GetRespMsgSendChan() <- rikkaMsg
			}
		}
	}
}

// todo test 主动发送消息测试 （获取发送者id -> 获取发送者对象（friend），发送）
func doubleEchoActive(a *Adapter, done chan struct{}) error {
	recvChan := a.rikkaBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:

			context := rikkaMsg.Content
			if strings.HasPrefix(context, "echo ") {
				trimed := strings.TrimPrefix(context, "echo ")
				rikkaMsg.Content = trimed
				rikkaMsg.Raw = []byte(rikkaMsg.Content)
				rikkaMsg.MetaData.GetRawMsg() // todo 主动发送消息的 rawMsg增强
			}
		}
	}
}

//func imgEcho(a *Adapter, done chan struct{}) error {
//	recvChan := a.rikkaBot.GetReqMsgRecvChan()
//	for {
//		select {
//		case <-done:
//			return nil
//		case rikkaMsg := <-recvChan:
//
//			if rikkaMsg.Msgtype == message.MsgTypeImage {
//				msg := rikkaMsg.MetaData.GetRawMsg().(*wcf.Message)
//				//err := msg.SaveFileToLocal("./test/testImg.jpg")
//				//if err != nil {
//				//	logging.WarnWithErr(err, "save img file to local fail")
//				//}
//				a.rikkaBot.GetRespMsgSendChan() <- rikkaMsg
//
//			}
//
//		}
//	}
//}

func runBase(t *testing.T, testfunc func(*Adapter, chan struct{}) error) {
	//bot := openwechat.DefaultBot(openwechat.Desktop)

	t.Logf("Start test\n")
	ctx := context.Background()
	cli := wcf.NewClient(10)
	cli.Run(false, false, false) // 运行wcf客户端

	rbot := rikkabot.NewRikkaBot(ctx, cli)
	rbot.EnableProcess = true // 允许处理消息
	a := NewAdapter(ctx, cli, rbot)
	a.HandleCovert() // 消息转换

	// test 实际回复功能测试
	done := make(chan struct{})
	go func() {
		defer close(done)
		err2 := testfunc(a, done)
		if err2 != nil {
			t.Logf("echo err: %v", err2)
		}
		time.Sleep(10 * time.Second)
		rbot.Exit() // 退出
	}()
	_ = rbot.Block()

	fmt.Println("hello")
	// Output: hello
}

func TestDelaytime(t *testing.T) {
	delayMin := 1
	delayMax := 3
	for i := 0; i < 10; i++ {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		startTime := time.Now()
		time.Sleep(time.Duration(rnd.Intn(1000*delayMax-1000*delayMin)+1000*delayMin) * time.Millisecond)
		fmt.Printf("delay %v\n", time.Now().Sub(startTime))
	}
}
