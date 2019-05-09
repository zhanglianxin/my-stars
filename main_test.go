package main

import (
	"testing"
	"fmt"
	"net/http"
	"strings"
	"net/url"
)

func TestMapKey(t *testing.T) {
	m := map[string]string{}
	if v, ok := m["name"]; ok {
		fmt.Println("if", v, ok)
	} else {
		fmt.Printf("else %#v %#v", v, ok) // else "" false
	}
}

func TestGetGists(t *testing.T) {
	getGists()
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
