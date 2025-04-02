// Package anime
// @Author Clover
// @Data 2025/3/31 上午1:22:00
// @Desc pixiv动漫图搜索
package anime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const sauceNaoAPIEndpoint = "https://saucenao.com/search.php"

// --- 定义响应结构体 ---

// ResponseHeader 对应 SauceNao 响应的全局 header
// 注意：某些字段类型（如 UserID, AccountType, ResultsRequested, MinimumSimilarity）
// 在不同文档或实际响应中可能是字符串或数字，这里根据常见情况设为 int 或 float64，
// 但在实际使用中可能需要根据 API 的确切返回值进行调整。
type ResponseHeader struct {
	UserID            interface{}            `json:"user_id"`      // 可能为 string 或 int
	AccountType       interface{}            `json:"account_type"` // 可能为 string 或 int
	ShortLimit        string                 `json:"short_limit"`
	LongLimit         string                 `json:"long_limit"`
	ShortRemaining    int                    `json:"short_remaining"`
	LongRemaining     int                    `json:"long_remaining"`
	Status            int                    `json:"status"`            // API 请求状态，0 表示成功
	ResultsRequested  interface{}            `json:"results_requested"` // 可能为 string 或 int
	Index             map[string]IndexStatus `json:"index"`
	SearchDepth       string                 `json:"search_depth"`
	MinimumSimilarity float64                `json:"minimum_similarity"`
	QueryImageDisplay string                 `json:"query_image_display"`
	QueryImage        string                 `json:"query_image"`
	ResultsReturned   int                    `json:"results_returned"`
	Message           string                 `json:"message"` // 可能的错误或状态信息
}

// IndexStatus 描述特定索引的状态
type IndexStatus struct {
	Status   int `json:"status"`
	ParentID int `json:"parent_id"`
	ID       int `json:"id"`
	Results  int `json:"results"`
}

// ResultHeader 对应单个结果的 header
type ResultHeader struct {
	Similarity string `json:"similarity"` // 相似度，通常是百分比字符串
	Thumbnail  string `json:"thumbnail"`
	IndexID    int    `json:"index_id"`
	IndexName  string `json:"index_name"`
	Dupes      int    `json:"dupes"`
	Hidden     int    `json:"hidden"`
}

// ResultData 对应单个结果的 data
// 注意：某些 ID 字段（如 PixivID, MemberID, SeigaID）可能是 int 或 string，
// 这里根据常见情况设为 int 或 string，实际使用时可能需要调整。
type ResultData struct {
	ExtURLs           []string    `json:"ext_urls"`
	Source            string      `json:"source"`
	AnidbAid          int         `json:"anidb_aid,omitempty"`
	MalID             int         `json:"mal_id,omitempty"`
	AnilistID         int         `json:"anilist_id,omitempty"`
	Part              string      `json:"part,omitempty"` // Episode/Chapter
	Year              string      `json:"year,omitempty"`
	EstTime           string      `json:"est_time,omitempty"`
	Title             string      `json:"title,omitempty"`
	FaID              int         `json:"fa_id,omitempty"`
	DaID              interface{} `json:"da_id,omitempty"`
	AuthorName        string      `json:"author_name,omitempty"`
	AuthorURL         string      `json:"author_url,omitempty"`
	Creator           interface{} `json:"creator,omitempty"`      // Can be string or []string
	CreatorName       string      `json:"creator_name,omitempty"` // Deprecated?
	TweetID           string      `json:"tweet_id,omitempty"`
	TwitterUserID     string      `json:"twitter_user_id,omitempty"`
	TwitterUserHandle string      `json:"twitter_user_handle,omitempty"`
	PixivID           interface{} `json:"pixiv_id,omitempty"` // 可能为 int 或 string
	MemberName        string      `json:"member_name,omitempty"`
	MemberID          interface{} `json:"member_id,omitempty"` // 可能为 int 或 string
	EngName           string      `json:"eng_name,omitempty"`
	JpName            string      `json:"jp_name,omitempty"`
	FnID              int         `json:"fn_id,omitempty"`
	FnType            string      `json:"fn_type,omitempty"`
	PawooID           string      `json:"pawoo_id,omitempty"`
	PawooUserName     string      `json:"pawoo_user_name,omitempty"`
	PawooUserAccount  string      `json:"pawoo_user_acct,omitempty"`
	PawooDisplayName  string      `json:"pawoo_display_name,omitempty"`
	SeigaID           interface{} `json:"seiga_id,omitempty"` // 可能为 int 或 string
	// 可以根据需要添加更多字段
}

// SauceNaoResult 对应单个搜索结果
type SauceNaoResult struct {
	Header ResultHeader `json:"header"`
	Data   ResultData   `json:"data"`
}

// SauceNaoResponse 对应完整的 SauceNao API 响应
type SauceNaoResponse struct {
	Header  ResponseHeader   `json:"header"`
	Results []SauceNaoResult `json:"results"`
}

// --- 搜索选项 ---

// SearchOptions 包含 SauceNao 搜索的可选参数
type SearchOptions struct {
	NumRes        int     // 结果数量 (numres)
	DB            int     // 搜索特定索引 (db)，999 表示所有。如果设置了 DBs，则忽略此项。
	TestMode      bool    // 测试模式 (testmode=1)
	Dedupe        int     // 去重级别 (dedupe)，有效值 0, 1, 2
	Hide          int     // 隐藏级别 (hide)，有效值 0, 1, 2, 3
	DBMask        *uint64 // 启用索引的位掩码 (dbmask)
	DBMaskI       *uint64 // 禁用索引的位掩码 (dbmaski)
	DBs           []int   // 搜索一个或多个特定索引 (dbs[]=...)，优先于 DB 参数
	MinSimilarity float64 // 客户端过滤：最低相似度要求（0-100）
}

// --- 搜索函数 ---

// SearchSauceNao 使用 SauceNao API 通过图片 URL 搜索图片
// apiKey: 你的 SauceNao API Key
// imageURL: 要搜索的图片的公开 URL
// options: 可选的搜索参数
// 返回值: 解析后的 SauceNao 响应指针和错误信息
func SearchSauceNao(apiKey, imageURL string, options *SearchOptions) (*SauceNaoResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if imageURL == "" {
		return nil, fmt.Errorf("image URL cannot be empty")
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("output_type", "2") // 必须是 JSON 输出
	params.Set("api_key", apiKey)
	params.Set("url", imageURL)

	// 设置默认值和处理可选参数
	numRes := 16   // 默认请求 16 个结果
	dbToUse := 999 // 默认搜索所有库
	dedupe := 2    // 默认去重级别
	hide := 0      // 默认不隐藏

	if options != nil {
		if options.NumRes > 0 {
			numRes = options.NumRes
		}
		// 优先使用 dbs 参数
		if len(options.DBs) > 0 {
			for _, db := range options.DBs {
				params.Add("dbs[]", strconv.Itoa(db))
			}
			dbToUse = -1 // 标记已使用 dbs，避免后面设置 db=999
		} else if options.DB > 0 {
			dbToUse = options.DB
		}

		if options.TestMode {
			params.Set("testmode", "1")
		}
		if options.Dedupe >= 0 && options.Dedupe <= 2 {
			dedupe = options.Dedupe
		}
		if options.Hide >= 0 && options.Hide <= 3 {
			hide = options.Hide
		}
		if options.DBMask != nil {
			params.Set("dbmask", strconv.FormatUint(*options.DBMask, 10))
			dbToUse = -1 // 使用了 mask，避免设置 db=999
		}
		if options.DBMaskI != nil {
			params.Set("dbmaski", strconv.FormatUint(*options.DBMaskI, 10))
			dbToUse = -1 // 使用了 mask，避免设置 db=999
		}
	}

	params.Set("numres", strconv.Itoa(numRes))
	params.Set("dedupe", strconv.Itoa(dedupe))
	params.Set("hide", strconv.Itoa(hide))
	if dbToUse != -1 { // 只有在没有使用 dbs 或 mask 时才设置 db 参数
		params.Set("db", strconv.Itoa(dbToUse))
	}

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s?%s", sauceNaoAPIEndpoint, params.Encode())
	// fmt.Println("Request URL:", requestURL) // 调试时可以取消注释

	// 发送 HTTP GET 请求
	client := &http.Client{Timeout: 60 * time.Second} // 建议复用 http.Client 并设置超时
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// 可以设置 User-Agent
	// req.Header.Set("User-Agent", "YourApp/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to SauceNao: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析 JSON 响应
	return parseSauceNaoResponse(resp.StatusCode, body, options) // 使用辅助函数解析
}

// SearchSauceNaoByFile 使用 SauceNao API 通过上传本地图片文件搜索图片
// apiKey: 你的 SauceNao API Key
// imagePath: 本地图片文件的完整路径
// options: 可选的搜索参数
// 返回值: 解析后的 SauceNao 响应指针和错误信息
func SearchSauceNaoByFile(apiKey string, imagePath string, options *SearchOptions) (*SauceNaoResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if imagePath == "" {
		return nil, fmt.Errorf("image path cannot be empty")
	}

	// 1. 打开图片文件
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file '%s': %w", imagePath, err)
	}
	defer file.Close()

	// 2. 获取文件名
	filename := filepath.Base(imagePath)

	// 3. 创建 multipart/form-data 请求体
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 4. 添加 API Key 和 output_type
	_ = writer.WriteField("api_key", apiKey)
	_ = writer.WriteField("output_type", "2") // 必须是 JSON 输出

	// 5. 添加可选参数到表单字段
	numRes := 16   // 默认请求 16 个结果
	dbToUse := 999 // 默认搜索所有库
	dedupe := 2    // 默认去重级别
	hide := 0      // 默认不隐藏

	if options != nil {
		if options.NumRes > 0 {
			numRes = options.NumRes
		}
		// 优先使用 dbs 参数
		if len(options.DBs) > 0 {
			for _, db := range options.DBs {
				_ = writer.WriteField("dbs[]", strconv.Itoa(db))
			}
			dbToUse = -1 // 标记已使用 dbs，避免后面设置 db=999
		} else if options.DB > 0 {
			dbToUse = options.DB
		}

		if options.TestMode {
			_ = writer.WriteField("testmode", "1")
		}
		if options.Dedupe >= 0 && options.Dedupe <= 2 {
			dedupe = options.Dedupe
		}
		if options.Hide >= 0 && options.Hide <= 3 {
			hide = options.Hide
		}
		if options.DBMask != nil {
			_ = writer.WriteField("dbmask", strconv.FormatUint(*options.DBMask, 10))
			dbToUse = -1 // 使用了 mask，避免设置 db=999
		}
		if options.DBMaskI != nil {
			_ = writer.WriteField("dbmaski", strconv.FormatUint(*options.DBMaskI, 10))
			dbToUse = -1 // 使用了 mask，避免设置 db=999
		}
	}

	_ = writer.WriteField("numres", strconv.Itoa(numRes))
	_ = writer.WriteField("dedupe", strconv.Itoa(dedupe))
	_ = writer.WriteField("hide", strconv.Itoa(hide))
	if dbToUse != -1 { // 只有在没有使用 dbs 或 mask 时才设置 db 参数
		_ = writer.WriteField("db", strconv.Itoa(dbToUse))
	}

	// 6. 创建文件部分
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// 7. 将文件内容写入文件部分 (file 实现了 io.Reader)
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file data to form: %w", err)
	}

	// 8. 关闭 multipart writer 来写入结尾的 boundary
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 9. 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", sauceNaoAPIEndpoint, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	// 10. 设置 Content-Type header，包含 boundary
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// 可以设置 User-Agent
	// req.Header.Set("User-Agent", "YourApp/1.0")

	// 11. 发送请求
	client := &http.Client{Timeout: 90 * time.Second} // 上传文件可能需要更长超时时间
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request to SauceNao: %w", err)
	}
	defer resp.Body.Close()

	// 12. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 13. 解析 JSON 响应 (使用辅助函数)
	return parseSauceNaoResponse(resp.StatusCode, body, options)
}

// SearchSauceNaoByBytes 使用 SauceNao API 通过图片字节数据搜索图片
// apiKey: 你的 SauceNao API Key
// imageData: 图片的 []byte 数据
// filename: 提供给 API 的文件名 (可以是任意字符串，例如 "image.jpg")
// options: 可选的搜索参数
// 返回值: 解析后的 SauceNao 响应指针和错误信息
func SearchSauceNaoByBytes(apiKey string, imageData []byte, filename string, options *SearchOptions) (*SauceNaoResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data cannot be empty")
	}
	if filename == "" {
		// SauceNao 的 multipart 请求需要一个文件名，即使是任意的
		filename = "image.bin" // 提供一个默认文件名
	}

	// 1. 创建 multipart/form-data 请求体
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 2. 添加 API Key 和 output_type
	_ = writer.WriteField("api_key", apiKey)
	_ = writer.WriteField("output_type", "2") // 必须是 JSON 输出

	// 3. 添加可选参数到表单字段 (与 SearchSauceNaoByFile 逻辑相同)
	numRes := 16   // 默认请求 16 个结果
	dbToUse := 999 // 默认搜索所有库
	dedupe := 2    // 默认去重级别
	hide := 0      // 默认不隐藏

	if options != nil {
		if options.NumRes > 0 {
			numRes = options.NumRes
		}
		// 优先使用 dbs 参数
		if len(options.DBs) > 0 {
			for _, db := range options.DBs {
				_ = writer.WriteField("dbs[]", strconv.Itoa(db))
			}
			dbToUse = -1 // 标记已使用 dbs，避免后面设置 db=999
		} else if options.DB > 0 {
			dbToUse = options.DB
		}

		if options.TestMode {
			_ = writer.WriteField("testmode", "1")
		}
		if options.Dedupe >= 0 && options.Dedupe <= 2 {
			dedupe = options.Dedupe
		}
		if options.Hide >= 0 && options.Hide <= 3 {
			hide = options.Hide
		}
		if options.DBMask != nil {
			_ = writer.WriteField("dbmask", strconv.FormatUint(*options.DBMask, 10))
			dbToUse = -1 // 使用了 mask，避免设置 db=999
		}
		if options.DBMaskI != nil {
			_ = writer.WriteField("dbmaski", strconv.FormatUint(*options.DBMaskI, 10))
			dbToUse = -1 // 使用了 mask，避免设置 db=999
		}
	}

	_ = writer.WriteField("numres", strconv.Itoa(numRes))
	_ = writer.WriteField("dedupe", strconv.Itoa(dedupe))
	_ = writer.WriteField("hide", strconv.Itoa(hide))
	if dbToUse != -1 { // 只有在没有使用 dbs 或 mask 时才设置 db 参数
		_ = writer.WriteField("db", strconv.Itoa(dbToUse))
	}

	// 4. 创建文件部分
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// 5. 将字节数据写入文件部分
	_, err = part.Write(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to write image data to form: %w", err)
	}

	// 6. 关闭 multipart writer 来写入结尾的 boundary
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 7. 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", sauceNaoAPIEndpoint, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	// 8. 设置 Content-Type header，包含 boundary
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// 可以设置 User-Agent
	// req.Header.Set("User-Agent", "YourApp/1.0")

	// 9. 发送请求
	client := &http.Client{Timeout: 90 * time.Second} // 上传数据可能需要更长超时时间
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request to SauceNao: %w", err)
	}
	defer resp.Body.Close()

	// 10. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 11. 解析 JSON 响应 (使用辅助函数)
	return parseSauceNaoResponse(resp.StatusCode, body, options)
}

// parseSauceNaoResponse 是一个辅助函数，用于解析 SauceNao 的响应并进行错误检查和过滤
func parseSauceNaoResponse(statusCode int, body []byte, options *SearchOptions) (*SauceNaoResponse, error) {
	var sauceNaoResp SauceNaoResponse
	if err := json.Unmarshal(body, &sauceNaoResp); err != nil {
		// 尝试找出非 200 状态码时的错误信息
		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("SauceNao API request failed with status code %d. Response body: %s", statusCode, string(body))
		}
		// 如果状态码是 200 但解析失败，可能是 API 结构变化或返回了非预期的 JSON
		return nil, fmt.Errorf("failed to decode SauceNao JSON response (status code %d): %w. Body: %s", statusCode, err, string(body))
	}

	// 检查响应头中的 API 状态码
	// status == 0: 成功
	// status > 0: 服务器端错误
	// status < 0: 客户端错误 (如 API Key 无效、搜索次数耗尽、图片无效等)
	if sauceNaoResp.Header.Status != 0 {
		errMsg := fmt.Sprintf("SauceNao API returned an error status %d", sauceNaoResp.Header.Status)
		if sauceNaoResp.Header.Message != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, sauceNaoResp.Header.Message)
		}
		// 根据文档，status < 0 是客户端错误，> 0 是服务端错误
		if sauceNaoResp.Header.Status < 0 {
			// 特殊处理 -2: "Search Rate Too High" / "Daily Search Limit Reached"
			// 特殊处理 -1: "Bad API Key or Account Disabled"
			// ... 可以根据需要添加更多特定错误码的处理
			return nil, fmt.Errorf("client-side error: %s", errMsg)
		}
		return nil, fmt.Errorf("server-side error: %s", errMsg)
	}

	// 可选：根据 options.MinSimilarity 在客户端过滤结果
	if options != nil && options.MinSimilarity > 0 && len(sauceNaoResp.Results) > 0 {
		filteredResults := make([]SauceNaoResult, 0, len(sauceNaoResp.Results))
		for _, result := range sauceNaoResp.Results {
			// 尝试解析相似度
			similarity, err := result.Header.GetSimilarity()
			// 如果解析失败或相似度低于阈值，则跳过
			if err != nil || similarity < options.MinSimilarity {
				continue
			}
			filteredResults = append(filteredResults, result)
		}
		sauceNaoResp.Results = filteredResults
	}

	return &sauceNaoResp, nil
}

// GetSimilarity 将 ResultHeader 中的相似度字符串转换为 float64 (0-100)
func (rh *ResultHeader) GetSimilarity() (float64, error) {
	similarity, err := strconv.ParseFloat(rh.Similarity, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse similarity '%s': %w", rh.Similarity, err)
	}
	return similarity, nil
}
