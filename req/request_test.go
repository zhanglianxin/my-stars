package req

import (
	"fmt"
	"github.com/zhanglianxin/my-stars/config"
	"strconv"
	"testing"
	"time"
)

func TestMakeRequest(t *testing.T) {
	headers := map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	conf := config.Load("../config.toml")
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", conf.User.Token)
	}
	res := MakeRequest("https://api.github.com/rate_limit", "get", headers, nil)

	ok := res.Header.Get("OK")
	remaining := res.Header.Get("X-RateLimit-Remaining")
	reset, _ := strconv.Atoi(res.Header.Get("X-RateLimit-Reset"))

	fmt.Println(ok, remaining, time.Unix(int64(reset), 0).Format(time.RFC3339))
}
