package gist

import (
	"time"
)

type Gist struct {
	Id          string           `json:"id"`
	Public      bool             `json:"public"`
	Description string           `json:"description"`
	URL         string           `json:"html_url"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdateAt    time.Time        `json:"updated_at"`
	Files       *map[string]File `json:"files"`
	Owner       struct {
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
