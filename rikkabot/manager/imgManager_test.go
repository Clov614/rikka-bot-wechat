// Package manager
// @Author Clover
// @Data 2024/8/11 下午5:44:00
// @Desc 本地图片文件模式下
package manager

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"wechat-demo/rikkabot/logging"
)

func init() {
	var err error
	imgDirpath = "./data/img"
	_, err = ValidPath(imgDirpath, true)
	if err != nil {
		log.Fatal().Err(err).Str("path", mDBPath).Msg("validate img_dir path")
	}
	ic = &imgCache{
		ImgValidDuration: 1,    // 图片有效期 7 day
		CheckInterval:    24,   // 检查是否过期间隔 24 hour
		IsCacheByFile:    true, // 是否文件方式存储图片
	}
	// 循环检测图片是否过期
	go cycleCheckOutDate(*ic)
}

// cycleCheckOutDate 循环检查图片桶是否过期
func cycleCheckOutDate(i imgCache) {
	firstFlag := true
	for {
		if firstFlag {
			logging.Warn("跳过第一次检测")
			firstFlag = false
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		logging.Warn("循环校验图片是否过期，间隔: " + strconv.Itoa(i.CheckInterval) + " Hour")
		var err error
		if i.IsCacheByFile {
			i.checkByFile(err)
		} else {
			i.checkByDB(err)
		}
		time.Sleep(time.Duration(10) * time.Second)
	}
}

func TestSaveImg(t *testing.T) {
	// read test_img
	data, _ := os.ReadFile("./test_img.jpg")

	imgName, imgDate := SaveImg("test-uuid", data)
	t.Logf("imgName: %s, imgDate: %s", imgName, imgDate)
}

func TestGetImg(t *testing.T) {
	var tests = []struct {
		imgdate string
		imgid   string
	}{
		{imgdate: "2024-08-11", imgid: "1723371050_test-uuid.jpg"},
		{imgdate: "2024-08-11", imgid: "1723370928_test-uuid.jpg"},
	}
	for i, test := range tests {
		data := GetImg(test.imgid, test.imgdate)
		if len(data) == 0 {
			t.Errorf("第 %d 条失败", i)
			t.Fail()
		}
		t.Log(len(data))
	}
}

func TestCycleCheckOutDate(t *testing.T) {
	done := make(chan struct{})
	go doblock(done)
	data, _ := os.ReadFile("./test_img.jpg")
	SaveImg("test-uuid", data)
	GetImg("1723371050_test-uuid.jpg", "2024-08-10")
	GetImg("1723370928_test-uuid.jpg", "2024-08-10")
	GetImg("1723370928_test-uuid.jpg", "2024-08-11")
	<-done
	t.Log("done")
}

func doblock(done chan struct{}) {
	fmt.Println("退出请输入 q 或者 exit")

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		command := strings.TrimSpace(input.Text())
		if command == "q" || command == "exit" {
			close(done)
			break
		}
		fmt.Println("输入内容未匹配退出条件，请继续输入：")
	}
}
