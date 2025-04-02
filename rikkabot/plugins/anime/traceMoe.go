// Package anime
// @Author Clover
// @Data 2025/3/31 上午12:51:00
// @Desc 搜番接口实现
package anime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Clov614/logging"
)

// TraceMoeResponse 是 trace.moe API 的响应结构体
type TraceMoeResponse struct {
	FrameCount int              `json:"frameCount"`
	Error      string           `json:"error"`
	Result     []TraceMoeResult `json:"result"`
}

// TraceMoeResult 包含单个搜索结果的信息
type TraceMoeResult struct {
	Anilist    interface{} `json:"anilist"`
	Filename   string      `json:"filename"`
	Episode    interface{} `json:"episode"`    // 集数可能是数字或 null
	From       float64     `json:"from"`       // 开始时间（秒）
	To         float64     `json:"to"`         // 结束时间（秒）
	Similarity float64     `json:"similarity"` // 相似度 (0-1)
	Video      string      `json:"video"`      // 预览视频 URL
	Image      string      `json:"image"`      // 预览图片 URL
}

// AnilistInfo 包含 Anilist 的相关信息
type AnilistInfo struct {
	ID       int       `json:"id"`
	IDMal    int       `json:"idMal"` // MyAnimeList ID
	Title    TitleInfo `json:"title"`
	Synonyms []string  `json:"synonyms"` // 同义词
	IsAdult  bool      `json:"isAdult"`  // 是否为成人内容
}

// TitleInfo 包含不同语言的标题
type TitleInfo struct {
	Native  string `json:"native"`  // 原生标题
	Romaji  string `json:"romaji"`  // 罗马音标题
	English string `json:"english"` // 英文标题
}

const traceMoeURL = "https://api.trace.moe/search"

// sendRequestAndParseResponse 是一个辅助函数，用于发送 HTTP 请求并解析 trace.moe 的响应
func sendRequestAndParseResponse(req *http.Request) (*TraceMoeResponse, error) {
	client := &http.Client{Timeout: 60 * time.Second} // 稍微增加超时时间，网络获取可能慢一些
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		// 尝试读取错误信息体
		bodyBytes, _ := io.ReadAll(resp.Body)
		// API 可能会返回 4xx 错误，例如 429 Too Many Requests
		// 或者 400 Bad Request (e.g., invalid image url)
		// 或者 5xx Server Error
		return nil, fmt.Errorf("API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	// 读取并解析 JSON 响应
	var traceResponse TraceMoeResponse
	if err := json.NewDecoder(resp.Body).Decode(&traceResponse); err != nil {
		// 如果状态码是 200 但解析失败，可能是 API 响应格式改变或网络问题导致响应不完整
		bodyBytes, _ := io.ReadAll(resp.Body) // 尝试重新读取原始响应体以供调试
		return nil, fmt.Errorf("解析 JSON 响应失败: %w, 原始响应体: %s", err, string(bodyBytes))
	}

	// 检查 API 返回的内部错误
	if traceResponse.Error != "" {
		return nil, fmt.Errorf("trace.moe API 错误: %s", traceResponse.Error)
	}

	return &traceResponse, nil
}

// SearchAnimeByImage 通过本地图片路径调用 trace.moe API 搜索动漫
// imagePath: 本地图片的完整路径
// 返回 API 响应和可能的错误
func SearchAnimeByImage(imagePath string) (*TraceMoeResponse, error) {
	// 1. 打开图片文件
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开图片文件 '%s': %w", imagePath, err)
	}
	defer file.Close()

	// 2. 读取文件内容
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("无法读取图片文件 '%s': %w", imagePath, err)
	}

	// 3. 获取文件的 MIME 类型
	ext := filepath.Ext(imagePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		switch strings.ToLower(ext) { // 转小写比较
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp": // 添加 webp 支持
			contentType = "image/webp"
		default:
			contentType = http.DetectContentType(fileBytes)
			// 检查推断出的类型是否是图片类型
			if !strings.HasPrefix(contentType, "image/") {
				return nil, fmt.Errorf("不支持的图片格式或无法确定 Content-Type: %s (推断为 %s)", ext, contentType)
			}
		}
		logging.Warn(fmt.Sprintf("无法通过扩展名确定 MIME 类型，尝试推断为: %s", contentType))
	}

	// 4. 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", traceMoeURL, bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP POST 请求失败: %w", err)
	}

	// 5. 设置 Content-Type 请求头
	req.Header.Set("Content-Type", contentType)

	// 6. 发送请求并解析响应 (使用辅助函数)
	return sendRequestAndParseResponse(req)
}

// SearchAnimeByURL 通过图片 URL 调用 trace.moe API 搜索动漫
// imageURL: 图片的公开访问 URL
// 返回 API 响应和可能的错误
func SearchAnimeByURL(imageURL string) (*TraceMoeResponse, error) {
	// 1. 验证 URL (基础检查)
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return nil, fmt.Errorf("无效的图片 URL 格式: %s", imageURL)
	}

	// 2. 构建带 URL 参数的 API 请求地址
	// 使用 url.QueryEscape 对传入的 URL 进行编码，防止特殊字符干扰
	apiURL := traceMoeURL + "?url=" + url.QueryEscape(imageURL)

	// 3. 创建 HTTP GET 请求
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP GET 请求失败: %w", err)
	}

	// 4. 发送请求并解析响应 (使用辅助函数)
	return sendRequestAndParseResponse(req)
}

// SearchAnimeByBytes 通过图片的字节数据调用 trace.moe API 搜索动漫
// imageData: 图片的 []byte 数据
// contentType: 图片的 MIME 类型 (例如 "image/jpeg", "image/png")
// 返回 API 响应和可能的错误
func SearchAnimeByBytes(imageData []byte, contentType string) (*TraceMoeResponse, error) {
	// 1. 检查 contentType 是否有效
	if !strings.HasPrefix(contentType, "image/") {
		logging.Warn(fmt.Sprintf("提供的 Content-Type '%s' 可能不是有效的图片类型", contentType))
		// 即使警告，也尝试发送请求，让 API 判断
	}
	if len(imageData) == 0 {
		return nil, fmt.Errorf("图片数据不能为空")
	}

	// 2. 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", traceMoeURL, bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP POST 请求失败: %w", err)
	}

	// 3. 设置 Content-Type 请求头
	req.Header.Set("Content-Type", contentType)

	// 4. 发送请求并解析响应 (使用辅助函数)
	return sendRequestAndParseResponse(req)
}

// // 示例用法 (可以放在 main 包或者测试文件中)
// func main() {
// 	imagePath := "path/to/your/demo.jpg" // 替换成你的图片路径
// 	response, err := SearchAnimeByImage(imagePath)
// 	if err != nil {
// 		fmt.Printf("搜番失败: %v\n", err)
// 		return
// 	}

// 	if len(response.TraceMoeResult) == 0 {
// 		fmt.Println("没有找到相似的动漫场景。")
// 		return
// 	}

// 	fmt.Printf("找到 %d 个结果:\n", len(response.TraceMoeResult))
// 	for i, result := range response.TraceMoeResult {
// 		fmt.Printf("\n--- 结果 %d ---\n", i+1)
// 		fmt.Printf("动漫名称 (Romaji): %s\n", result.Anilist.Title.Romaji)
// 		fmt.Printf("动漫名称 (Native): %s\n", result.Anilist.Title.Native)
// 		if result.Anilist.Title.English != "" {
// 			fmt.Printf("动漫名称 (English): %s\n", result.Anilist.Title.English)
// 		}
// 		if result.Episode != nil {
// 			fmt.Printf("集数: %v\n", result.Episode)
// 		}
// 		fmt.Printf("时间点: %.2f - %.2f 秒\n", result.From, result.To)
// 		fmt.Printf("相似度: %.2f%%\n", result.Similarity*100)
// 		fmt.Printf("文件名: %s\n", result.Filename)
// 		// fmt.Printf("预览视频: %s\n", result.Video) // 可能需要处理 URL
// 		// fmt.Printf("预览图片: %s\n", result.Image) // 可能需要处理 URL
// 	}
// }
