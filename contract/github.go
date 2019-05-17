package contract

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type GitHub interface {
	GetLastPage(path string, pageSize int) int
	GetPagedData(path string, pageSize int, page int) ([]byte, error)
	GetAll(path string) ([]byte, error)
}

// Get last page from response header
func LastPage(res *http.Response) int {
	lastPage := 1
	links := linkheader.Parse(res.Header.Get("Link"))
	for _, link := range links {
		if "last" == link.Rel {
			params := strings.SplitAfter(link.URL, "?")[1]
			if "" != params {
				if values, err := url.ParseQuery(params); nil == err {
					lastPage, _ = strconv.Atoi(values.Get("page"))
				}
			}
		}
	}
	return lastPage
}

func PagedData(res *http.Response) (slice []interface{}) {
	defer res.Body.Close()
	if http.StatusOK == res.StatusCode {
		decoder := json.NewDecoder(res.Body)
		var rs []interface{}
		if err := decoder.Decode(&rs); nil != err {
			logrus.Errorf("[GetPagedData] decode paged data: %s", err)
		} else {
			slice = append(slice, rs...)
		}
	} else {
		logrus.Infof("[GetPagedData] %s: %s", res.Status)
	}
	return slice
}
