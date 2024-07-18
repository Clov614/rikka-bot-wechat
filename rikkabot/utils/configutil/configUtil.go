// Package configutil
// @Author Clover
// @Data 2024/7/6 下午3:51:00
// @Desc
package configutil

import (
	"fmt"
	encoder "github.com/zwgblue/yaml-encoder"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// Save 保存配置文件为 yml
func Save(config interface{}, path string, filename string) error {
	newEncoder := encoder.NewEncoder(config, encoder.WithComments(encoder.CommentsOnHead))
	encode, _ := newEncoder.Encode() // ignore err
	return SaveConfig(encode, path, filename)
}

// Load 读取yml配置文件
func Load(config interface{}, path string, filename string) error {
	return LoadConfig(config, path, filename)
}

// todo 进一步完善校验 （文件格式校验 尾后缀校验）

func SaveConfig(data []byte, path string, filename string) error {
	path, err := getPath(path, filename, true)
	if err != nil {
		return fmt.Errorf("error path load: %w", err)
	}
	err = os.WriteFile(path, data, 0644)
	return fmt.Errorf("error save config: %w", err)
}

func LoadConfig(v interface{}, path string, filename string) error {
	path, err := getPath(path, filename, false)
	if err != nil {
		return fmt.Errorf("error path load: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}
	err = yaml.Unmarshal(data, v)
	return fmt.Errorf("error loading config file: %w", err)
}

func getPath(path string, filename string, iswrite bool) (string, error) {
	if filename == "" {
		filename = "config.yaml"
	}
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
