package gist

import (
	"encoding/json"
	"github.com/zhanglianxin/my-stars/contract"
	"github.com/zhanglianxin/my-stars/req"
	"strconv"
	"sync"
	"time"
)

const apiHost = "https://api.github.com/"

type Gist struct {
	contract.GitHub `json:"-"`
	Id              string           `json:"id"`
	Public          bool             `json:"public"`
	Description     string           `json:"description"`
	URL             string           `json:"html_url"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdateAt        time.Time        `json:"updated_at"`
	Files           *map[string]File `json:"files"`
	Owner           struct {
		Login string `json:"login"`
		URL   string `json:"html_url"`
	} `json:"owner"`
}

type File struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawUrl   string `json:"raw_url"`
	Size     int    `json:"size"`
}

var (
	headers    = req.Headers
	retryTimes int
)

func NewGist() (gs *Gist) {
	return &Gist{}
}

func (g *Gist) GetLastPage(path string, pageSize int) int {
	params := map[string]string{
		"per_page": strconv.Itoa(pageSize),
	}
	res := req.MakeRequest(path, "head", headers, params)
	return contract.LastPage(res)
}

func (g *Gist) GetPagedData(path string, pageSize int, page int) ([]byte, error) {
	params := map[string]string{
		"per_page": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}
	res := req.MakeRequest(path, "get", headers, params)
	return json.Marshal(contract.PagedData(res))
}

func (g *Gist) GetAll(path string) ([]byte, error) {
	var gists []Gist
	var err error

	pageSize := 10
	lastPage := g.GetLastPage(path, pageSize)

	var wg sync.WaitGroup
	wg.Add(lastPage)

	for page := 1; page <= lastPage; page++ {
		go func(page int) {
			defer wg.Done()
			if b, e := g.GetPagedData(path, pageSize, page); nil == e {
				var gs []Gist
				json.Unmarshal(b, &gs)
				gists = append(gists, gs...)
			} else {
				err = e
				return
			}
		}(page)
		time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()

	return json.Marshal(gists)
}
