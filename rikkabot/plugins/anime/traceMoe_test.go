package anime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createDummyFile 创建一个用于测试的临时文件
func createDummyFile(t *testing.T, name string, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "Failed to create dummy file")
	return filePath
}

// assertResultValid 是一个辅助函数，用于验证单个 TraceMoeResult 对象的字段
func assertResultValid(t *testing.T, result TraceMoeResult) {
	t.Helper() // 标记为测试辅助函数

	// 检查基本字段
	assert.NotEmpty(t, result.Filename, "TraceMoeResult.Filename should not be empty")
	assert.Greater(t, result.Similarity, 0.0, "TraceMoeResult.Similarity should be greater than 0.0")
	assert.GreaterOrEqual(t, result.From, 0.0, "TraceMoeResult.From should be non-negative")
	assert.GreaterOrEqual(t, result.To, 0.0, "TraceMoeResult.To should be non-negative")
	assert.LessOrEqual(t, result.From, result.To, "TraceMoeResult.From should be less than or equal to TraceMoeResult.To")
	assert.NotEmpty(t, result.Video, "TraceMoeResult.Video URL should not be empty")
	assert.NotEmpty(t, result.Image, "TraceMoeResult.Image URL should not be empty")

	// 检查 Anilist 字段 (类型断言)
	// 根据之前的讨论和 JSON 示例，我们期望它是一个数字 (float64)
	anilistIDFloat, ok := result.Anilist.(float64)
	if assert.True(t, ok, "TraceMoeResult.Anilist should be a number (float64)") {
		anilistID := int(anilistIDFloat)
		assert.NotZero(t, anilistID, "Anilist ID (converted from float64) should not be zero")
	}

	// 检查 Episode 字段 (类型断言)
	// 可能是数字 (float64) 或 null (nil)
	if result.Episode != nil {
		episodeFloat, ok := result.Episode.(float64)
		if assert.True(t, ok, "TraceMoeResult.Episode, if not nil, should be a number (float64)") {
			episodeNum := int(episodeFloat)
			assert.GreaterOrEqual(t, episodeNum, 0, "Episode number (converted from float64) should be non-negative") // 允许第 0 集？通常是 >= 1，但 >= 0 更安全
		}
	}
}

// TestSearchAnimeByImage_Success 测试 SearchAnimeByImage 成功的情况
func TestSearchAnimeByImage_Success(t *testing.T) {
	// !!! 重要提示: 确保 "./img.png" 是一个真实有效的图片文件路径 !!!
	realImagePath := "./img.png"

	if _, err := os.Stat(realImagePath); os.IsNotExist(err) {
		t.Skipf("Skipping test: Real image file not found at %s", realImagePath)
		return
	}

	resp, err := SearchAnimeByImage(realImagePath)

	// 基本断言
	assert.NoError(t, err, "SearchAnimeByImage should not return an error with a valid image")
	require.NotNil(t, resp, "Response should not be nil for a valid image")
	assert.Empty(t, resp.Error, "Response.Error should be empty on success") // 检查 API 内部错误字段

	// 检查结果
	assert.GreaterOrEqual(t, len(resp.Result), 0, "Expected 0 or more results")
	if len(resp.Result) > 0 {
		// 使用辅助函数验证第一个结果
		assertResultValid(t, resp.Result[0])
		// 可以选择性地验证更多结果，或所有结果
		// for _, res := range resp.TraceMoeResult {
		//     assertResultValid(t, res)
		// }
	}
}

// TestSearchAnimeByImage_FileNotExist 测试文件不存在的情况
func TestSearchAnimeByImage_FileNotExist(t *testing.T) {
	resp, err := SearchAnimeByImage("non_existent_file.jpg")
	assert.Error(t, err, "Expected an error for non-existent file")
	assert.Nil(t, resp, "Response should be nil on error")
	assert.Contains(t, err.Error(), "无法打开图片文件", "Error message should indicate file open failure")
}

// TestSearchAnimeByImage_UnsupportedFormat 测试不支持的文件格式
func TestSearchAnimeByImage_UnsupportedFormat(t *testing.T) {
	dummyFilePath := createDummyFile(t, "test.txt", "this is not an image")
	resp, err := SearchAnimeByImage(dummyFilePath)
	assert.Error(t, err, "Expected an error for unsupported format")
	assert.Nil(t, resp, "Response should be nil on error")
	assert.Contains(t, err.Error(), "不支持的图片格式", "Error message should indicate unsupported format")
}

// TestSearchAnimeByURL_Success 测试 SearchAnimeByURL 成功的情况
func TestSearchAnimeByURL_Success(t *testing.T) {
	testImageURL := "https://images.plurk.com/32B15UXxymfSMwKGTObY5e.jpg"

	resp, err := SearchAnimeByURL(testImageURL)

	// 基本断言
	assert.NoError(t, err, "SearchAnimeByURL should not return an error with a valid URL")
	require.NotNil(t, resp, "Response should not be nil for a valid URL")
	assert.Empty(t, resp.Error, "Response.Error should be empty on success") // 检查 API 内部错误字段

	// 检查结果
	assert.GreaterOrEqual(t, len(resp.Result), 0, "Expected 0 or more results")
	if len(resp.Result) > 0 {
		// 使用辅助函数验证第一个结果
		assertResultValid(t, resp.Result[0])
	}
}

// TestSearchAnimeByURL_InvalidURL 测试无效 URL 格式
func TestSearchAnimeByURL_InvalidURL(t *testing.T) {
	invalidURL := "not a valid url"
	resp, err := SearchAnimeByURL(invalidURL)
	assert.Error(t, err, "Expected an error for invalid URL format")
	assert.Nil(t, resp, "Response should be nil on error")
	assert.Contains(t, err.Error(), "无效的图片 URL 格式", "Error message should indicate invalid URL format")
}
