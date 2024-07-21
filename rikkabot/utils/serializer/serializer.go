// Package serializer @Author Clover
// @Data 2024/7/5 下午5:45:00
// @Desc 串行化器
package serializer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type Serializer interface {
	serialize(interface{}) ([]byte, error)
	unserialize([]byte, interface{}) error
}

var S serializer

type serializer struct{}

func (s serializer) serialize(v interface{}) ([]byte, error) {
	marshal, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serializer Error: %w", err)
	}
	return marshal, nil
}

func (s serializer) unserialize(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("unserializer Error: %w", err)
	}
	return nil
}

// Save certain types of temporary json file
// path: 存放路径（可选），"" 默认存放至同目录
// filename: 存放的文件名
func Save(path string, filename string, v interface{}) error {
	data, err := S.serialize(v)
	if err != nil {
		return err
	}
	path, err = getPath(path, filename, v, true)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	return nil
}

// Load certain types by json file
// path: 存放路径（可选），"" 默认存放至同目录
// filename: 存放的文件名
func Load(path string, filename string, v interface{}) error {
	path, err := getPath(path, filename, v, false)
	if err != nil {
		return fmt.Errorf("get load serialize path %s error: %w", path, err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("path %s loading serializer error: %w", path, err)
	}
	return S.unserialize(bytes, v)
}

func getPath(path string, filename string, v interface{}, iswrite bool) (string, error) {
	if filename == "" {
		vType := reflect.TypeOf(v).Name()
		filename = vType
		if filename == "" {
			filename = "temp"
		}
	}
	filename = filename + ".json"
	if path == "" {
		path = "./"
	}
	path = filepath.Join(path, filename)
	if iswrite {
		// 检测目录是否存在
		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// 创建所需目录
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("error creating directory %s: %w", dir, err)
			}
		}
	}

	return path, nil
}

func IsPathExist(path string, filename string) bool {
	path = filepath.Join(path, filename)
	// 检测目录是否存在
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}
