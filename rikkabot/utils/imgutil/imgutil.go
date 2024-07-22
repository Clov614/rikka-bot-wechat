// Package imgutil
// @Author Clover
// @Data 2024/7/22 下午1:53:00
// @Desc 图片处理工具类
package imgutil

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func ImgFetch(path string) ([]byte, error) {
	if isURL(path) {
		return fetchFromURL(path)
	}
	return fetchFromFile(path)
}

func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// fetchFromURL fetches the content from the URL
func fetchFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetchFromURL: http.Get(%q): %w", url, err)
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchFromURL: resp.Body.ReadAll(): %w", err)
	}
	return bytes, nil
}

// fetchFromFile fetches the content from the file
func fetchFromFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("fetchFromFile: os.Open(%q): %w", filePath, err)
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("fetchFromFile: io.ReadAll(file): %w", err)
	}
	return bytes, nil
}
