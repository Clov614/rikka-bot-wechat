// Package zepplife
// @Author Clover
// @Data 2024/9/17 下午7:52:00
// @Desc
package zepplife

import (
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strings"
	"time"
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func FakeIP() string {
	return fmt.Sprintf("223.%d.%d.%d", r.Intn(54)+64, r.Intn(254), r.Intn(254))
}

func GetFromData(data map[string]string) io.Reader {
	formData := url.Values{}
	for k, v := range data {
		formData.Add(k, v)
	}
	return strings.NewReader(formData.Encode())
}

func GetRandInt64(n int64) int64 {
	return r.Int63n(n)
}
