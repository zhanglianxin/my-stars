package req

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/zhanglianxin/my-stars/config"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var Headers = map[string]string{
	"Accept": "application/vnd.github.v3+json",
}

func init() {
	if _, ok := Headers["Authorization"]; !ok {
		Headers["Authorization"] = fmt.Sprintf("token %s", config.Conf.User.Token)
	}
}

func MakeRequest(urlStr string, method string, headers map[string]string, params map[string]string) *http.Response {
	method = strings.ToUpper(method)
	client := &http.Client{}
	req, err := http.NewRequest(method, urlStr, nil)
	if nil != err {
		panic(err)
	}
	for key := range headers {
		req.Header.Add(key, headers[key])
	}
	q := req.URL.Query()
	for key := range params {
		q.Add(key, params[key])
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if nil != err {
		panic(err)
	}

	// check rate limit
	remainingStr := resp.Header.Get("X-RateLimit-Remaining")
	resetStr := resp.Header.Get("X-RateLimit-Reset")
	remaining, _ := strconv.Atoi(remainingStr)
	reset, _ := strconv.Atoi(resetStr)
	if "" != remainingStr && "" != resetStr && 0 == remaining {
		logrus.Infof("url: %s, method: %s, headers: %v, params: %v, original: %s, %s, parsed: %d, %d",
			urlStr, method, headers, params,
			resp.Header.Get("X-RateLimit-Remaining"), resp.Header.Get("X-RateLimit-Reset"),
			remaining, reset)
		panic(fmt.Sprintf("Reach to rate limit, please wait until %s",
			time.Unix(int64(reset), 0).Format(time.RFC3339)))
	}
	return resp
}
