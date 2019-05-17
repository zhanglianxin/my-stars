package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"
	"github.com/zhanglianxin/my-stars/config"
	"github.com/zhanglianxin/my-stars/gist"
	"github.com/zhanglianxin/my-stars/rate"
	"github.com/zhanglianxin/my-stars/repo"
	"github.com/zhanglianxin/my-stars/req"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"sort"
	"math/rand"
)

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

	halfYear = 180 * 24 * time.Hour
)

var (
	conf    *config.Config
	headers = map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	repos []repo.Repo
	gists []gist.Gist
	start time.Time
)

func init() {
	start = time.Now()
	rand.Seed(start.Unix())

	logName := time.Now().Format("2006-01-02") + ".log"
	file, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if nil != err {
		panic(err)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
	})
	logrus.SetOutput(file)

	conf = config.Load("config.toml")

	for _, name := range []string{"list.txt", "repo/README.md", "gist/README.md"} {
		os.Remove(name)
	}
}

func main() {
	if !rate.NewLimit().HasRemaining() {
		panic("current token has no remaining times")
	}

	getStarredRepos()
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Language < repos[j].Language
	})

	var wg sync.WaitGroup
	wg.Add(len(repos))
	for i := range repos {
		go func(i int) {
			defer wg.Done()
			var retryTimes int
			getHeadCommit(&repos[i], retryTimes, i)
		}(i)
	}
	wg.Wait()

	filterNoUpdateInHalfYear()
	getStarredGists()
	saveTable()
	fmt.Println("--- DONE ---")
	fmt.Printf("cost: %.3f s\n", time.Now().Sub(start).Seconds())
}

func getStarredRepos() {
	path := apiHost + "/user/starred"
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", conf.User.Token)
	}
	params := map[string]string{
		"per_page": "50",
	}
	res := req.MakeRequest(path, "head", headers, params)
	lastPage := getLastPage(res)
	var wg sync.WaitGroup
	wg.Add(lastPage)

	for page := 1; page <= lastPage; page++ {
		go func(page int, repos *[]repo.Repo) {
			defer wg.Done()
			params := map[string]string{
				"per_page": "50",
				"page":     strconv.Itoa(page),
			}
			res := req.MakeRequest(path, "get", headers, params)
			defer res.Body.Close()
			if http.StatusOK == res.StatusCode {
				b, _ := ioutil.ReadAll(res.Body)
				var rs []repo.Repo
				if err := json.Unmarshal(b, &rs); nil != err {
					logrus.Errorf("[getStarredRepos] decode repos: %s", err)
				} else {
					*repos = append(*repos, rs...)
				}
			} else {
				logrus.Infof("[getStarredRepos] %s: %s", res.Status, path)
			}
		}(page, &repos)
	}
	wg.Wait()
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

// Check if has next page
func hasNextPage(res *http.Response) bool {
	return len(linkheader.Parse(res.Header.Get("Link")).FilterByRel("next")) > 0
}

func getHeadCommit(rp *repo.Repo, retryTimes int, i int) {
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", conf.User.Token)
	}
	// Get last commit date
	path := fmt.Sprintf("%s/repos/%s/commits/%s", apiHost, rp.FullName, rp.DefaultBranch)
	res := req.MakeRequest(path, "get", headers, nil)
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		var commit repo.HeadCommit
		b, _ := ioutil.ReadAll(res.Body)
		if err := json.Unmarshal(b, &commit); nil != err {
			logrus.Errorf("[getHeadCommit] decode commit: %s", err)
		}
		rp.LastCommitDate = commit.Commit.Committer.Date
	case http.StatusNotFound:
		logrus.Infof("[getHeadCommit] %s: %s", res.Status, rp.URL)
		repos = append(repos[:i], repos[i+1:]...)
	case http.StatusForbidden: // 403
		// retry
		if retryTimes < 8 {
			retryTimes++
			time.Sleep(time.Duration(100*retryTimes) * time.Microsecond)
			logrus.Infof("[getHeadCommit] retry: %s, %d", rp.URL, retryTimes)
			getHeadCommit(rp, retryTimes, i)
		}
	default:
		logrus.Infof("%s: %s", res.Status, rp.URL)
	}

}

func filterNoUpdateInHalfYear() {
	f, err := os.OpenFile("list.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	defer f.Close()
	if nil != err {
		panic(err)
	}

	now := time.Now()
	for i := range repos {
		diff := now.Sub(repos[i].LastCommitDate)
		if diff > halfYear {
			f.WriteString(fmt.Sprintln(repos[i].URL, repos[i].LastCommitDate.Format(time.RFC3339)))
		}
	}

}

func getStarredGists() {
	path := apiHost + "/gists/starred"
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", conf.User.Token)
	}
	params := map[string]string{
		"per_page": "50",
	}
	res := req.MakeRequest(path, "head", headers, params)
	lastPage := getLastPage(res)
	var wg sync.WaitGroup
	wg.Add(lastPage)

	for page := 1; page <= lastPage; page++ {
		go func(page int) {
			defer wg.Done()
			params := map[string]string{
				"per_page": "50",
				"page":     strconv.Itoa(page),
			}
			res := req.MakeRequest(path, "get", headers, params)
			defer res.Body.Close()
			if http.StatusOK == res.StatusCode {
				b, _ := ioutil.ReadAll(res.Body)
				var gs []gist.Gist
				if err := json.Unmarshal(b, &gs); nil != err {
					logrus.Errorf("[getStarredGists] decode gists: %s", err)
				} else {
					gists = append(gists, gs...)
				}
			} else {
				logrus.Infof("[getStarredGists] %s: %s", res.Status, path)
			}
		}(page)
	}
	wg.Wait()
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
}

func saveRepoTable() {
	readme, err := os.OpenFile("repo/README.md", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	defer readme.Close()
	if nil != err {
		panic(err)
	}
	readme.WriteString(repoHead)

	for i := range repos {
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
