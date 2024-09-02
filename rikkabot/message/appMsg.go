// Package message
// @Author Clover
// @Data 2024/9/2 下午5:14:00
// @Desc xml 消息体（app 消息）
package message

import "encoding/xml"

// XMLMsg 定义XML结构体
type XMLMsg struct {
	XMLName    xml.Name `xml:"msg"`
	AppMsg     AppMsg   `xml:"appmsg"`
	AppInfo    AppInfo  `xml:"appinfo"`
	Scene      int      `xml:"scene"`
	CommentURL string   `xml:"commenturl"`
}

type AppMsg struct {
	XMLName   xml.Name  `xml:"appmsg"`
	AppID     string    `xml:"appid,attr"`
	SDKVer    string    `xml:"sdkver,attr"`
	Title     string    `xml:"title"`
	Des       string    `xml:"des"`
	Type      int       `xml:"type"`
	URL       string    `xml:"url"`
	AppAttach AppAttach `xml:"appattach"`
	MD5       string    `xml:"md5"`
	StateText string    `xml:"statextstr"`
}

type AppAttach struct {
	CDNThumbURL    string `xml:"cdnthumburl"`
	CDNThumbMD5    string `xml:"cdnthumbmd5"`
	CDNThumbLength int    `xml:"cdnthumblength"`
	CDNThumbWidth  int    `xml:"cdnthumbwidth"`
	CDNThumbHeight int    `xml:"cdnthumbheight"`
	CDNThumbAESKey string `xml:"cdnthumbaeskey"`
	AESKey         string `xml:"aeskey"`
	EncryVer       int    `xml:"encryver"`
	FileKey        string `xml:"filekey"`
}

type AppInfo struct {
	Version string `xml:"version"`
	AppName string `xml:"appname"`
}
