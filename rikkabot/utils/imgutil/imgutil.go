// Package imgutil
// @Author Clover
// @Data 2024/7/22 下午1:53:00
// @Desc 图片处理工具类
package imgutil

import (
	"bytes"
	"crypto/tls"
	"errors"
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

// FileType 表示文件类型的枚举
type FileType string

const (
	JPEG FileType = "jpg"
	PNG  FileType = "png"
	GIF  FileType = "gif"
	BMP  FileType = "bmp"
	TIFF FileType = "tiff"
	// 可以根据需要添加更多类型
)

// SignatureMap 存储文件签名和对应的文件类型
var SignatureMap = map[FileType][][]byte{
	JPEG: {
		{0xFF, 0xD8, 0xFF},
	},
	PNG: {
		{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	},
	GIF: {
		{0x47, 0x49, 0x46, 0x38, 0x37, 0x61},
		{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
	},
	BMP: {
		{0x42, 0x4D},
	},
	TIFF: {
		{0x49, 0x49, 0x2A, 0x00},
		{0x4D, 0x4D, 0x00, 0x2A},
	},
}

var (
	ErrUnknowFileType = errors.New("unknown file type")
)

// DetectFileType 检测文件的字节前缀以确定其类型
func DetectFileType(data []byte) (FileType, error) {
	for fileType, signatures := range SignatureMap {
		for _, signature := range signatures {
			if len(data) >= len(signature) && bytes.Equal(data[:len(signature)], signature) {
				return fileType, nil
			}
		}
	}
	return "", fmt.Errorf("detectFileType: %w", ErrUnknowFileType)
}

// GetMimeTypeByFileType 根据 FileType 返回 MIME 类型
func GetMimeTypeByFileType(fileType FileType) string {
	switch fileType {
	case JPEG:
		return "image/jpeg"
	case PNG:
		return "image/png"
	case GIF:
		return "image/gif"
	case BMP:
		return "image/bmp"
	case TIFF:
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}

// GetEtxByFileType 根据 FileType 返回 Ext 文件后缀
func GetEtxByFileType(fileType FileType) string {
	switch fileType {
	case JPEG:
		return ".jpg"
	case PNG:
		return ".png"
	case GIF:
		return ".gif"
	case BMP:
		return ".bmp"
	case TIFF:
		return ".tiff"
	default:
		return ""
	}
}
