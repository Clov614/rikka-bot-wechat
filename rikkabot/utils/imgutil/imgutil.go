// Package imgutil
// @Author Clover
// @Data 2024/7/22 下午1:53:00
// @Desc 图片处理工具类
package imgutil

import (
	"crypto/tls"
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
	// 跳过 TLS 验证
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchFromURL: creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
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
