package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/zhanglianxin/my-stars/gist"
	"github.com/zhanglianxin/my-stars/rate"
	"github.com/zhanglianxin/my-stars/repo"
	"github.com/zhanglianxin/my-stars/req"
	"math/rand"
	"os"
	"sort"
	"time"
)

const (
	apiHost = "https://api.github.com"
	head    = `# Get All My Starred Repos and Gists

> Inspired by [go-web-framework-stars](https://github.com/mingrammer/go-web-framework-stars).

* [my starred repos](repo/README.md)

* [my starred gists](gist/README.md)
`
	repoHead = `# All My Starred Repos

| Project Name | Stars | Forks | Language | Description | Last Push |
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
	headers = req.Headers
	repos   []repo.Repo
	gists   []gist.Gist
	start   time.Time
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

	filterNoUpdateInHalfYear()
	getStarredGists()
	saveTable()
	fmt.Println("--- DONE ---")
	fmt.Printf("cost: %.3f s\n", time.Now().Sub(start).Seconds())
}

func getStarredRepos() {
	path := apiHost + "/user/starred"
	rp := repo.NewRepo()
	if b, e := rp.GetAll(path); nil == e {
		if err := json.Unmarshal(b, &repos); err != nil {
			panic(err)
		}
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
		diff := now.Sub(repos[i].Pushed)
		if diff > halfYear {
			f.WriteString(fmt.Sprintln(repos[i].URL, repos[i].Pushed.Format(time.RFC3339)))
		}
	}
}

func getStarredGists() {
	path := apiHost + "/gists/starred"
	g := gist.NewGist()
	if b, e := g.GetAll(path); nil == e {
		if err := json.Unmarshal(b, &gists); err != nil {
			panic(err)
		}
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
			repos[i].Pushed.Format(time.RFC3339))
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
