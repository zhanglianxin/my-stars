package config

import (
	"github.com/BurntSushi/toml"
)

type UserInfo struct {
	Username string `toml:"username"`
	Token    string `toml:"token"`
}

type Config struct {
	User *UserInfo
}

var Conf *Config

func init() {
	Load("config.toml")
}

func Load(file string) (config *Config) {
	if _, err := toml.DecodeFile(file, &config); nil != err {
		panic(err)
	}
	Conf = config
	return config
}
