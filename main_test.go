package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"sort"
	"github.com/tomnomnom/linkheader"
	"io/ioutil"
)

func TestMapKey(t *testing.T) {
	m := map[string]string{}
	if v, ok := m["name"]; ok {
		fmt.Println("if", v, ok)
	} else {
		fmt.Printf("else %#v %#v", v, ok) // else "" false
	}
}

func TestMakeRequest(t *testing.T) {
	res := makeRequest("https://coolman.site", "get", nil, nil)
	if http.StatusOK != res.StatusCode {
		t.Error("oops!!", res.StatusCode)
	}
}

func TestParseQuery(t *testing.T) {
	q := "https://api.github.com/user/9329713/starred?page=28&name=zhanglianxin"
	fmt.Println(strings.SplitAfter(q, "?")[1])
	values, _ := url.ParseQuery(strings.SplitAfter(q, "?")[1])
	fmt.Println(values)
}

func TestSort(t *testing.T) {
	repos := []string{"abc", "bcd", "bca"}
	fmt.Println(repos)
	sort.Slice(repos, func(i, j int) bool {
		return repos[i] < repos[j]
	})
	fmt.Println(repos)
}

func TestHasNextPage(t *testing.T) {
	fmt.Println(headers)
	res := makeRequest("https://api.github.com/user/starred?page=27", "get", headers, nil)
	links := linkheader.Parse(res.Header.Get("Link"))
	fmt.Println(links)
	l0 := links.FilterByRel("ok")
	l1 := links.FilterByRel("next")
	fmt.Println(len(l0), len(l1), l1)
	if len(l0) > 0 || 1 != len(l1) {
		t.Error("oops")
	}
}

func TestHeadRequest(t *testing.T) {
	res := makeRequest("https://api.github.com/user/starred?page=27", "head", headers, nil)
	links := res.Header.Get("Link")
	fmt.Println(links)
	b, _ := ioutil.ReadAll(res.Body)
	if 0 != len(b) {
		t.Error("body content length error")
	}
}
