package rate

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/zhanglianxin/my-stars/config"
	"github.com/zhanglianxin/my-stars/req"
	"io/ioutil"
	"net/http"
)

const ApiHost = "https://api.github.com/"

type Rate struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

type Resources struct {
	Core                *Rate `json:"core"`
	Search              *Rate `json:"search"`
	Graphql             *Rate `json:"graphql"`
	IntegrationManifest *Rate `json:"integration_manifest"`
}

type Limit struct {
	Resources *Resources `json:"resources"`
	Rate      *Rate      `json:"rate"`
}

var (
	conf    *config.Config
	headers = map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	path = ApiHost + "rate_limit"
)

func init() {
	conf = config.Load("config.toml")
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", conf.User.Token)
	}
}

func NewLimit() (limit *Limit) {
	res := req.MakeRequest(path, "get", headers, nil)
	defer res.Body.Close()
	if http.StatusOK == res.StatusCode {
		b, _ := ioutil.ReadAll(res.Body)
		if err := json.Unmarshal(b, &limit); nil != err {
			logrus.Errorf("decode limit: %s", err)
		}
	}
	return limit
}

func (limit *Limit) HasRemaining() bool {
	return limit.Rate.Remaining > 0
}
