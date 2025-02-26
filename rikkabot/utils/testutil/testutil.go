// Package testutil
// @Author Clover
// @Data 2024/7/31 下午4:32:00
// @Desc
package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Clov614/logging"
	wcf "github.com/Clov614/wcf-rpc-sdk"
)

const (
	maxFileSize = 10 * 1024 * 1024 // 10MB 文件大小限制
)

var (
	fileMutex sync.Mutex
)

// SaveTestMessage 保存测试消息至文件，追加模式，文件过大时覆盖
func SaveTestMessage(msg *wcf.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	msgTypeName := wcf.MsgTypeNames[msg.Type]
	if msgTypeName == "" {
		msgTypeName = "UnknownMsgType"
	}
	safeMsgTypeName := replaceInvalidFilenameChars(msgTypeName)
	filePath := filepath.Join("./data/tmp/test/", safeMsgTypeName+".json") // 确保文件名为 .json 扩展名

	fileMutex.Lock() // 使用互斥锁保护文件操作
	defer fileMutex.Unlock()

	// 检查文件大小，如果超出限制则覆盖文件
	fileFlag := os.O_APPEND | os.O_CREATE | os.O_WRONLY // 默认追加模式
	if isFileSizeExceed(filePath, maxFileSize) {
		fileFlag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC // 文件超限，使用覆盖模式
		logging.Warn("test file size exceed limit, overwrite file", map[string]interface{}{"file": filePath, "limit": maxFileSize})
	}

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory error: %w", err)
		}
	}

	file, err := os.OpenFile(filePath, fileFlag, 0644)
	if err != nil {
		return fmt.Errorf("open file error: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("encode message to json error: %w", err)
	}

	return nil
}

func isFileSizeExceed(filePath string, maxSize int64) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false // 文件不存在或 Stat 操作失败，不视为超出限制 (首次创建时会覆盖)
	}
	return fileInfo.Size() >= maxSize
}

func replaceInvalidFilenameChars(filename string) string {
	// 允许的字符 (字母数字下划线)
	unValidChars := "| " // 不符合字符
	var safeFilenameBuilder []rune
	for _, r := range filename {
		if strings.ContainsRune(unValidChars, r) {
			safeFilenameBuilder = append(safeFilenameBuilder, '_') // 不符合的字符替换为下划线
		} else {
			safeFilenameBuilder = append(safeFilenameBuilder, r)
		}
		// 可以选择忽略或替换其他非法字符
	}
	safeFilename := string(safeFilenameBuilder)

	if safeFilename == "" {
		safeFilename = "default" // 避免文件名为空
	}
	return safeFilename
}
