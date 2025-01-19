// @Author Clover
// @Data 2024/7/5 下午6:09:00
// @Desc
package adapter

import (
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/serializer"
	"github.com/eatmoreapple/openwechat"
	_ "image/png"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// 测试是否转发接收到的消息，二次封装是否正常
func TestReceiveRawMsg(t *testing.T) {

}

// 测试rikkaBot发送的消息是否能转发给 openwechat
func TestSendRikkaMsg(t *testing.T) {

}

// 获取 第三方平台发送来的经转换后的消息json，保存于同级目录下
func TestGetMsgJson(t *testing.T) {
	bot := openwechat.DefaultBot(openwechat.Desktop)

	a := NewAdapter(bot, rikkabot.GetDefaultBot())

	HandleCovert(a)
	defer a.Close()

	t.Logf("Start test\n")

	go func() {
		// 保存 json
		cnt := 7
		for rikkaMsg := range a.selfBot.GetReqMsgRecvChan() {
			fmt.Printf("save json recv rikkaMsg: %#v\n", *rikkaMsg)
			err := serializer.Save("./test/cacheMsg", fmt.Sprintf("rikkaMsg%d", cnt), rikkaMsg)
			senderId := rikkaMsg.SenderId
			receiverId := rikkaMsg.ReceiverId
			t.Logf("sender id: %s, receiver id: %s\n", senderId, receiverId)
			if err != nil {
				t.Logf("err: %v", err)
			}
			cnt++
		}
	}()

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func() {
		err := reloadStorage.Close()
		if err != nil {
			t.Logf("err: %v", err)
		}
	}()
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		t.Error(err)
		return
	}

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		t.Logf("err: %v", err)
		return
	}

	if self == nil {
		t.Errorf("GetCurrentUser err")
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	err = bot.Block()
	if err != nil {
		t.Error(err)
	}
	fmt.Println("hello")
	// Output: hello
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
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			_ = rikkaMsg.Content
			self, _ := a.openwcBot.GetCurrentUser()
			friends, _ := self.Friends()
			groups, _ := self.Groups()
			members, _ := self.Members()
			//rawmsg := rikkaMsg.MetaData.GetRawMsg().(*openwechat.Message)

			fmt.Printf("friends: %#v\n", friends)
			fmt.Printf("groups: %v\n", groups)
			fmt.Printf("members: %v\n", members)

			rawMsg := rikkaMsg.MetaData.GetRawMsg().(*openwechat.Message)

			fmt.Printf("rawMsg: %#v\n", rawMsg)
		}
	}
}

// 测试各种消息的唯一ID
func GetUserId(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:
			if rikkaMsg.Msgtype == message.MsgTypeText {
				if rikkaMsg.IsGroup { // 群消息返回
					rikkaMsg.Content = fmt.Sprintf("groupId= %s \n senderId = %s \n"+
						" receviceId = %s", rikkaMsg.GroupId, rikkaMsg.SenderId, rikkaMsg.ReceiverId)
					rikkaMsg.Raw = []byte(rikkaMsg.Content)
					a.selfBot.GetRespMsgSendChan() <- rikkaMsg // 原路返回消息
				} else {
					rikkaMsg.Content = fmt.Sprintf("senderId = %s \n receviceId = %s",
						rikkaMsg.SenderId, rikkaMsg.ReceiverId)
					rikkaMsg.Raw = []byte(rikkaMsg.Content)
					a.selfBot.GetRespMsgSendChan() <- rikkaMsg // 原路返回消息
				}
			}
		}
	}
}

func echo(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
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
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg
			}
		}
	}
}

func doubleEcho(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
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
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg
			}
		}
	}
}

// todo test 主动发送消息测试 （获取发送者id -> 获取发送者对象（friend），发送）
func doubleEchoActive(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
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

func imgEcho(a *Adapter, done chan struct{}) error {
	recvChan := a.selfBot.GetReqMsgRecvChan()
	for {
		select {
		case <-done:
			return nil
		case rikkaMsg := <-recvChan:

			if rikkaMsg.Msgtype == message.MsgTypeImage {
				msg := rikkaMsg.MetaData.GetRawMsg().(*openwechat.Message)
				err := msg.SaveFileToLocal("./test/testImg.jpg")
				if err != nil {
					logging.WarnWithErr(err, "save img file to local fail")
				}
				a.selfBot.GetRespMsgSendChan() <- rikkaMsg

			}

		}
	}
}

func runBase(t *testing.T, testfunc func(*Adapter, chan struct{}) error) {
	bot := openwechat.DefaultBot(openwechat.Desktop)

	t.Logf("Start test\n")

	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func() {
		err := reloadStorage.Close()
		if err != nil {
			t.Logf("err: %v", err)
		}
	}()
	if err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		t.Error(err)
		return
	}

	a := NewAdapter(bot, rikkabot.GetDefaultBot())
	HandleCovert(a)
	defer a.Close()

	// test 实际回复功能测试
	done := make(chan struct{})
	go func() {
		defer close(done)
		err2 := testfunc(a, done)
		if err2 != nil {
			t.Logf("echo err: %v", err2)
		}

	}()

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		t.Logf("err: %v", err)
		return
	}

	if self == nil {
		t.Errorf("GetCurrentUser err")
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	err = bot.Block()
	if err != nil {
		t.Logf("err: %v", err)
	}
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

func FuzzDelaytime(f *testing.F) {
	f.Fuzz(func(t *testing.T) {

	})
}
