package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Qumomf struct {
		Port string `yaml:"port"`
	} `yaml:"qumomf"`
	Tarantool struct {
		ConnectTimeout time.Duration `yaml:"connect_timeout"`
		RequestTimeout time.Duration `yaml:"request_timeout"`
	} `yaml:"tarantool"`
	Shards  map[string][]ShardConfig `yaml:"shards"`
	Routers []ShardConfig            `yaml:"routers"`
}

type ShardConfig struct {
	Name     string `yaml:"name"`
	Addr     string `yaml:"addr"`
	UUID     string `yaml:"uuid"`
	Priority int    `yaml:"priority"`
	Master   bool   `yaml:"master"`
}

func Setup(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
