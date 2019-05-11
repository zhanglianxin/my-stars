package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
	"github.com/zhanglianxin/my-stars/config"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"sort"
)

// Repo describes a Github repository with additional field, last commit date
type Repo struct {
	Name           string    `json:"name"`
	FullName       string    `json:"full_name"`
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
	Id          string               `json:"id"`
	Public      bool                 `json:"public"`
	Description string               `json:"description"`
	URL         string               `json:"html_url"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdateAt    time.Time            `json:"updated_at"`
	Files       *map[string]GistFile `json:"files"`
	Owner struct {
		Login string `json:"login"`
		URL   string `json:"html_url"`
	} `json:"owner"`
}

type GistFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawUrl   string `json:"raw_url"`
	Size     int    `json:"size"`
}

const (
	apiHost = "https://api.github.com"
	head    = `# Get All My Starred Repos and Gists

> Inspired by [go-web-framework-stars](https://github.com/mingrammer/go-web-framework-stars).

* [my starred repos](repo/README.md)

* [my starred gists](gist/README.md)
`
	repoHead = `# All My Starred Repos

| Project Name | Stars | Forks | Language | Description | Last Commit |
| ------------ | ----- | ----- | -------- | ----------- | ----------- |
`
	gistHead = `# All My Starred Gists

| Gist Id | Description | Last Commit |
| ------- | ----------- | ----------- |
`
	repoTmpl = "| [%s](%s) | %d | %d | %s | %s | %s |\n"
	gistTmpl = "| [%s](%s) / [%s](%s) | %s | %s |\n"
	tail     = "\n**Last Update**: *%v*\n"
)

var (
	conf    *config.Config
	headers = map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	repos []Repo
	gists []Gist
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
	getStarredRepos()
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Language < repos[j].Language
	})
	getStarredGists()
	saveTable()
	fmt.Println("--- DONE ---")
}

func getStarredRepos() {
	page, lastPage, perPage := 1, 1, 50
	path := apiHost + "/user/starred"
	params := map[string]string{"per_page": strconv.Itoa(perPage)}

	for ; page <= lastPage; page++ {
		params["page"] = strconv.Itoa(page)
		res := makeRequest(path, "get", headers, params)
		// defer res.Body.Close()
		if http.StatusOK == res.StatusCode {
			b, _ := ioutil.ReadAll(res.Body)
			if 1 == lastPage {
				// Get last page
				lastPage = getLastPage(res)
			}

			var rs []Repo
			if err := json.Unmarshal(b, &rs); nil != err {
				logrus.Error("decode gists", err)
			} else {
				repos = append(repos, rs...)
			}
		}
		res.Body.Close()
	}
}

// Get last page from response header
func getLastPage(res *http.Response) int {
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

func getHeadCommit(repo *Repo) {
	// Get last commit date
	path := fmt.Sprintf("%s/repos/%s/commits/%s", apiHost, repo.FullName, repo.DefaultBranch)
	res := makeRequest(path, "get", headers, nil)
	defer res.Body.Close()
	if http.StatusOK == res.StatusCode {
		var commit HeadCommit
		b, _ := ioutil.ReadAll(res.Body)
		if err := json.Unmarshal(b, &commit); nil != err {
			logrus.Error("decode commit", err)
		}
		repo.LastCommitDate = commit.Commit.Committer.Date
	}
}

func getStarredGists() {
	page, lastPage, perPage := 1, 1, 50
	path := apiHost + "/gists/starred"
	params := map[string]string{"per_page": strconv.Itoa(perPage)}

	for ; page <= lastPage; page++ {
		params["page"] = strconv.Itoa(page)
		res := makeRequest(path, "get", headers, params)
		// defer res.Body.Close()
		if http.StatusOK == res.StatusCode {
			b, _ := ioutil.ReadAll(res.Body)
			if 1 == lastPage {
				// Get last page
				lastPage = getLastPage(res)
			}

			var gs []Gist
			if err := json.Unmarshal(b, &gs); nil != err {
				logrus.Error("decode gists", err)
			} else {
				gists = append(gists, gs...)
			}
		}
		res.Body.Close()
	}
}

func saveTable() {
	readme, err := os.OpenFile("README.md", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	defer readme.Close()
	if nil != err {
		panic(err)
	}
	readme.WriteString(head)

	saveRepoTable()
	saveGistTable()

	// readme.WriteString(fmt.Sprintf(tail, time.Now().Format(time.RFC3339)))
}

func saveRepoTable() {
	readme, err := os.OpenFile("repo/README.md", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	defer readme.Close()
	if nil != err {
		panic(err)
	}
	readme.WriteString(repoHead)

	for i := range repos {
		getHeadCommit(&repos[i])
		line := fmt.Sprintf(repoTmpl,
			repos[i].Name, repos[i].URL, repos[i].Stars, repos[i].Forks, repos[i].Language, repos[i].Description,
			repos[i].LastCommitDate.Format(time.RFC3339))
		readme.WriteString(line)
	}

	readme.WriteString(fmt.Sprintf(tail, time.Now().Format(time.RFC3339)))
}

func saveGistTable() {
	readme, err := os.OpenFile("gist/README.md", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	defer readme.Close()
	if nil != err {
		panic(err)
	}
	readme.WriteString(gistHead)

	for i := range gists {
		line := fmt.Sprintf(gistTmpl,
			gists[i].Owner.Login, gists[i].Owner.URL, gists[i].Id, gists[i].URL,
			gists[i].Description, gists[i].UpdateAt)
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
