package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

type Config struct {
	Qumomf struct {
		Port string `yaml:"port"`
	} `yaml:"qumomf"`

	Tarantool struct {
		ConnectTimeout time.Duration `yaml:"connect_timeout"`
		RequestTimeout time.Duration `yaml:"request_timeout"`

		Topology struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"topology"`
	} `yaml:"tarantool"`

	Clusters map[string]ClusterConfig `yaml:"clusters"`
}

type ClusterConfig struct {
	Shards  map[vshard.ShardUUID][]vshard.InstanceConfig `yaml:"shards"`
	Routers []vshard.InstanceConfig                      `yaml:"routers"`
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

	cfg.overrideEmptyByGlobalConfigs()

	return &cfg, nil
}

func (c *Config) overrideEmptyByGlobalConfigs() {
	overrideFn := func(instCfg vshard.InstanceConfig) {
		globalCfg := c.Tarantool

		if instCfg.ConnectTimeout == 0 {
			instCfg.ConnectTimeout = globalCfg.ConnectTimeout
		}
		if instCfg.RequestTimeout == 0 {
			instCfg.RequestTimeout = globalCfg.RequestTimeout
		}
		if instCfg.User == "" {
			instCfg.User = globalCfg.Topology.User
		}
		if instCfg.Password == "" {
			instCfg.Password = globalCfg.Topology.Password
		}
	}

	for _, clusterCfg := range c.Clusters {
		for _, set := range clusterCfg.Shards {
			for _, shardCfg := range set {
				overrideFn(shardCfg)
			}
		}

		for _, routerCfg := range clusterCfg.Routers {
			overrideFn(routerCfg)
		}
	}
}
