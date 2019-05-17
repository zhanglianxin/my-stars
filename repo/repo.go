package repo

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/zhanglianxin/my-stars/config"
	"github.com/zhanglianxin/my-stars/contract"
	"github.com/zhanglianxin/my-stars/req"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const apiHost = "https://api.github.com/"

// Repo describes a Github repository with additional field, last commit date
type Repo struct {
	contract.GitHub `json:"-"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	DefaultBranch   string    `json:"default_branch"`
	Stars           int       `json:"stargazers_count"`
	Forks           int       `json:"forks_count"`
	Issues          int       `json:"open_issues_count"`
	Created         time.Time `json:"created_at"`
	Updated         time.Time `json:"updated_at"` // when starred
	Pushed          time.Time `json:"pushed_at"`
	URL             string    `json:"html_url"`
	Language        string    `json:"language"`
	LastCommitDate  time.Time `json:"-"`
}

// HeadCommit describes a head commit of default branch
type HeadCommit struct {
	Sha    string `json:"sha"`
	Commit struct {
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

var (
	headers    = req.Headers
	retryTimes int
)

func init() {
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = fmt.Sprintf("token %s", config.Conf.User.Token)
	}
}

func NewRepo() (rp *Repo) {
	return &Repo{}
}

func (rp *Repo) GetLastPage(path string, pageSize int) int {
	params := map[string]string{
		"per_page": strconv.Itoa(pageSize),
	}
	res := req.MakeRequest(path, "head", headers, params)
	return contract.LastPage(res)
}

func (rp *Repo) GetPagedData(path string, pageSize int, page int) ([]byte, error) {
	params := map[string]string{
		"per_page": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}
	res := req.MakeRequest(path, "get", headers, params)
	return json.Marshal(contract.PagedData(res))
}

func (rp *Repo) GetAll(path string) ([]byte, error) {
	var repos []Repo

	pageSize := 50
	lastPage := rp.GetLastPage(path, pageSize)

	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(lastPage)

	for page := 1; page <= lastPage; page++ {
		go func(page int) {
			defer wg.Done()
			mutex.Lock()
			defer mutex.Unlock()
			if b, e := rp.GetPagedData(path, pageSize, page); nil == e {
				var rs []Repo
				json.Unmarshal(b, &rs)
				repos = append(repos, rs...)
			}
		}(page)
		time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()

	return json.Marshal(repos)
}

// Get last commit date
func (rp *Repo) getHeadCommit() (commit *HeadCommit) {
	path := fmt.Sprintf("%srepos/%s/commits/%s", apiHost, rp.FullName, rp.DefaultBranch)
	res := req.MakeRequest(path, "get", headers, nil)
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		decoder := json.NewDecoder(res.Body)
		if err := decoder.Decode(&commit); nil != err {
			logrus.Errorf("[getHeadCommit] decode commit: %s", err)
		}
	case http.StatusNotFound:
		logrus.Errorf("[getHeadCommit] %s: %s", res.Status, rp.URL)
	case http.StatusForbidden: // 403
		// retry
		if retryTimes < 10 {
			retryTimes++
			time.Sleep(100 * time.Duration(retryTimes) * time.Millisecond)
			logrus.Infof("[getHeadCommit] retry: %s, %d", rp.URL, retryTimes)
			rp.getHeadCommit()
		}
	default:
		logrus.Infof("%s: %s", res.Status, rp.URL)
	}
	return commit
}
