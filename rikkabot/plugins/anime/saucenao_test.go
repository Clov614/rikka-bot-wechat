package anime

import (
	"os"
	"testing"
)

// 从环境变量获取 API Key
func getApiKeyFromEnv(t *testing.T) string {
	apiKey := os.Getenv("SAUCENAO_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping SauceNao test: SAUCENAO_API_KEY environment variable not set.")
	}
	return apiKey
}

// 测试通过上传真实本地文件搜索
func TestSearchSauceNaoByFile(t *testing.T) {
	apiKey := getApiKeyFromEnv(t)

	// 指定真实图片的路径
	//realImagePath := "./img2.jpg" // 确保图片在此路径
	realImagePath := "./104423833_p0.jpg" // 确保图片在此路径

	// 检查真实图片文件是否存在
	if _, err := os.Stat(realImagePath); err != nil {
		if os.IsNotExist(err) {
			// 如果文件不存在，跳过此测试
			t.Skipf("Skipping test: Real image file not found at %s", realImagePath)
			return // 明确返回以停止执行
		}
		// 如果是其他错误（例如权限问题），则测试失败
		t.Fatalf("Error checking real image file at %s: %v", realImagePath, err)
	}

	t.Logf("Found real image file at %s, using it for testing.", realImagePath)

	options := &SearchOptions{
		NumRes: 5, // 请求一些结果
	}

	// 使用真实文件路径调用函数
	resp, err := SearchSauceNaoByFile(apiKey, realImagePath, options)

	// 对真实图片的期望：成功调用，API status 0，可能有一些结果
	if err != nil {
		t.Fatalf("SearchSauceNaoByFile failed with real image '%s': %v", realImagePath, err)
	}
	if resp == nil {
		t.Fatalf("SearchSauceNaoByFile returned nil response without error for real image '%s'", realImagePath)
	}
	if resp.Header.Status != 0 {
		t.Fatalf("SearchSauceNaoByFile returned non-zero API status %d for real image '%s'. Message: %s", resp.Header.Status, realImagePath, resp.Header.Message)
	}

	t.Logf("SearchSauceNaoByFile successful with real image '%s'. Header Status: %d", realImagePath, resp.Header.Status)
	t.Logf("Results Returned: %d", resp.Header.ResultsReturned)

	// 可以根据你的 img.png 添加更具体的断言，例如检查是否有结果返回
	if len(resp.Results) == 0 {
		t.Logf("Warning: No results found for the real image '%s'. This might be expected depending on the image.", realImagePath)
	} else {
		t.Logf("Found %d results for real image.", len(resp.Results))
		// 打印一些结果信息（可选）
		for i, result := range resp.Results {
			sim, _ := result.Header.GetSimilarity()
			t.Logf("  Result %d: Similarity=%.2f%%, Index=%s", i+1, sim, result.Header.IndexName)
		}
	}
}

// 你可以添加更多测试用例，例如测试不同的 SearchOptions 组合
