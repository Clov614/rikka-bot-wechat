// Package imgutil
// @Author Clover
// @Data 2024/7/30 下午11:39:00
// @Desc 图片工具测试
package imgutil

import "testing"

func TestDetectImgType(t *testing.T) {
	data, err := fetchFromURL("https://www.freeimg.cn/i/2024/04/22/66260f2eed1d6.jpg")
	if err != nil {
		t.Error(err)
	}

	fileType, err := DetectFileType(data)
	if err != nil {
		t.Error(err)
	}
	t.Log(fileType)

	data2, err := fetchFromURL("https://www.freeimg.cn/i/2024/04/22/66260f0ae65a3.png")
	if err != nil {
		t.Error(err)
	}

	fileType2, err := DetectFileType(data2)
	if err != nil {
		t.Error(err)
	}
	t.Log(fileType2)
}
