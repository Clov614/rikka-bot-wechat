// @Author Clover
// @Data 2024/7/5 下午8:47:00
// @Desc
package adapter

import (
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"math/rand"
	"time"
)

var Covert = (*Adapter).covert

var SendMsg = (*Adapter).sendMsg

var HandleCovert = func(a *Adapter) {
	a.openwcBot.MessageHandler = func(msg *openwechat.Message) {
		fmt.Printf("收到消息： %#v\n", msg)
		a.receiveMsg(msg)
	}
	go func() {
		respMsgRecvChan := a.selfBot.GetRespMsgRecvChan()
		for {
			select {
			case <-a.done:
				return
			case respMsg := <-respMsgRecvChan:
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				time.Sleep(time.Duration((rnd.Intn(1000) + 1000)) * time.Millisecond)
				a.sendMsg(respMsg) // todo 错误处理
			}
		}
	}()
}
