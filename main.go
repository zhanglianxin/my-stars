package main

import (
	"time"
	"fmt"
	"os"
	"github.com/sirupsen/logrus"
	"github.com/zhanglianxin/my-stars/config"
	"net/http"
	"strings"
	"io/ioutil"
	"encoding/json"
	"github.com/tomnomnom/linkheader"
	"net/url"
	"strconv"
)

// Repo describes a Github repository with additional field, last commit date
type Repo struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	DefaultBranch  string    `json:"default_branch"`
	Stars          int       `json:"stargazers_count"`
	Forks          int       `json:"forks_count"`
	Issues         int       `json:"open_issues_count"`
	Created        time.Time `json:"created_at"`
	Updated        time.Time `json:"updated_at"`
	URL            string    `json:"html_url"`
	Language       string    `json:"language"`
	LastCommitDate time.Time `json:"-"`
}

// HeadCommit describes a head commit of default branch
type HeadCommit struct {
	Sha string `json:"sha"`
	Commit struct {
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

type Gist struct {
	Id          int       `json:"id"`
	Public      bool      `json:"public"`
	Description string    `json:"description"`
	URL         string    `json:"html_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdateAt    time.Time `json:"updated_at"`
}

const (
	apiHost = "https://api.github.com"
	head    = `# Get All My Starred Repos

Inspired by [go-web-framework-stars](https://github.com/mingrammer/go-web-framework-stars).

| Project Name | Stars | Forks | Language | Description | Last Commit |
| ------------ | ----- | ----- | -------- | ----------- | ----------- |
`
	tail = "\n*Last Update: %v*\n"
)

var (
	conf    *config.Config
	headers = map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	repos    []Repo
	lastPage int
)

func init() {
	logName := time.Now().Format("2006-01-02") + ".log"
	file, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if nil != err {

		panic(err)
	}
	logrus.SetOutput(file)
	conf = config.Load("config.toml")
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", conf.User.Token)
	}
}

func main() {
	getStarred()
	if len(repos) > 0 {
		saveTable(repos)
	}
}

func getStarred() {
	path := apiHost + "/user/starred"
	params := map[string]string{"per_page": "50"}
	page := 1
	// for page := 1; page <= lastPage; page++ {
	params["page"] = strconv.Itoa(page)
	res := makeRequest(path, "get", headers, params)
	// defer res.Body.Close() // FIXME possible resource leak
	if http.StatusOK == res.StatusCode {
		b, _ := ioutil.ReadAll(res.Body)
		if 0 == lastPage {
			links := linkheader.Parse(res.Header.Get("Link"))
			for _, link := range links {
				if "last" == link.Rel {
					params := strings.SplitAfter(link.URL, "?")[1]
					if "" != params {
						if values, err := url.ParseQuery(params); nil != err {
							lastPage, _ = strconv.Atoi(values.Get("page"))
						}
					} else {
						lastPage = 1
					}
				}
			}
		}
		if err := json.Unmarshal(b, &repos); nil != err {
			logrus.Error("decode repos", err)
		}
	}
	res.Body.Close()
	// }
}

func getHeadCommit() {
	// TODO get last commit date
}

func getGists() {
	path := apiHost + "/gists"
	res := makeRequest(path, "get", headers, nil)
	defer res.Body.Close()
	if http.StatusOK == res.StatusCode {
		b, _ := ioutil.ReadAll(res.Body)
		result := string(b)
		fmt.Println(result)
	}
}

func saveTable(repos []Repo) {
	readme, err := os.OpenFile("README.md", os.O_RDWR|os.O_TRUNC, 0666)
	if nil != err {
		panic(err)
	}
	readme.WriteString(head)
	for i := range repos {
		line := fmt.Sprintf("| [%s](%s) | %d | %d | %s | %s | %s |\n",
			repos[i].Name, repos[i].URL, repos[i].Stars, repos[i].Forks, repos[i].Language, repos[i].Description, "-")
		readme.WriteString(line)
	}
	readme.WriteString(fmt.Sprintf(tail, time.Now().Format(time.RFC3339)))
}

func makeRequest(url string, method string, headers map[string]string, params map[string]string) *http.Response {
	method = strings.ToUpper(method)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
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
	return resp
}
