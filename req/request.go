package req

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
	remaining, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	reset, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Reset"))
	if 0 == remaining {
		logrus.Info("123", resp.Header.Get("X-RateLimit-Remaining"), resp.Header.Get("X-RateLimit-Reset"))
		panic(fmt.Sprintf("Reach to rate limit, please wait until %s",
			time.Unix(int64(reset), 0).Format(time.RFC3339)))
	}
	return resp
}
