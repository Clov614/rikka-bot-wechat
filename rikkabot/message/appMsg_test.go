// Package message
// @Author Clover
// @Data 2024/9/2 下午5:15:00
// @Desc 测试解析app消息
package message

import (
	"encoding/xml"
	"fmt"
	"log"
	"testing"
)

func TestAppMsgParse(t *testing.T) {
	data := `<?xml version="1.0"?>
	<msg>
		<appmsg appid="wxcb8d4298c6a09bcb" sdkver="0">
			<title>这真的不是Cg 是cos</title>
			<des>UP主：努力的十八魔 播放：72.2万</des>
			<type>4</type>
			<url>https://b23.tv/MZLsTCt</url>
			<appattach>
				<cdnthumburl>3057020100044b304902010002045192ec6902032f59e102040ec7587d020466d57e0d042463313939666431392d613232322d346238612d383063382d6332333265633363363133350204051808030201000405004c56f900</cdnthumburl>
				<cdnthumbmd5>ea934d14510b70cfd6458d2ea08261bd</cdnthumbmd5>
				<cdnthumblength>29048</cdnthumblength>
				<cdnthumbwidth>160</cdnthumbwidth>
				<cdnthumbheight>160</cdnthumbheight>
				<cdnthumbaeskey>a4f9f09d834eab83217a9aa7b8ae242c</cdnthumbaeskey>
				<aeskey>a4f9f09d834eab83217a9aa7b8ae242c</aeskey>
				<encryver>0</encryver>
				<filekey>wxid_p5z4fuhnbdgs22_1570_1725267527</filekey>
			</appattach>
			<md5>ea934d14510b70cfd6458d2ea08261bd</md5>
			<statextstr>GhQKEnd4Y2I4ZDQyOThjNmEwOWJjYg==</statextstr>
		</appmsg>
		<fromusername></fromusername>
		<scene>0</scene>
		<appinfo>
			<version>8</version>
			<appname>哔哩哔哩</appname>
		</appinfo>
		<commenturl />
	</msg>`

	var msg XMLMsg
	err := xml.Unmarshal([]byte(data), &msg)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("Parsed XML: %+v\n", msg)
}
