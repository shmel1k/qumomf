package tarantool

import (
	"github.com/viciious/go-tarantool"
)

type ShardConfig struct {
	Name     string `yaml:"name"`
	Addr     string `yaml:"addr"`
	UUID     string `yaml:"uuid"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Priority int    `yaml:"priority"`
	Master   bool   `yaml:"master"`
}

func SetupConnection(c ShardConfig) {
	cfg := tarantool.Options{}
	_ = cfg
}
