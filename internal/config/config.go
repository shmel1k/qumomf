package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	defaultUser           = "guest"
	defaultPassword       = "guest"
	defaultConnectTimeout = 5 * time.Second
	defaultRequestTimeout = 5 * time.Second
)

type Config struct {
	Qumomf struct {
		Port string `yaml:"port"`
	} `yaml:"qumomf"`

	Tarantool struct {
		ConnectTimeout time.Duration `yaml:"connect_timeout"`
		RequestTimeout time.Duration `yaml:"request_timeout"`

		Topology struct {
			ReadOnly bool   `yaml:"readonly,omitempty"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"topology"`
	} `yaml:"tarantool"`

	Clusters map[string]ClusterConfig `yaml:"clusters"`
}

type ClusterConfig struct {
	ReadOnly *bool                       `yaml:"readonly,omitempty"`
	Shards   map[string][]InstanceConfig `yaml:"shards"`
	Routers  []InstanceConfig            `yaml:"routers"`
}

type InstanceConfig struct {
	Name           string         `yaml:"name"`
	Addr           string         `yaml:"addr"`
	UUID           string         `yaml:"uuid"`
	User           *string        `yaml:"user,omitempty"`
	Password       *string        `yaml:"password,omitempty"`
	ConnectTimeout *time.Duration `yaml:"connect_timeout,omitempty"`
	RequestTimeout *time.Duration `yaml:"request_timeout,omitempty"`
	Master         bool           `yaml:"master"`
}

func (c *InstanceConfig) withDefaults() {
	if c == nil {
		return
	}

	if c.ConnectTimeout == nil {
		v := defaultConnectTimeout
		c.ConnectTimeout = &v
	}
	if c.RequestTimeout == nil {
		v := defaultRequestTimeout
		c.RequestTimeout = &v
	}
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
	cfg.withDefaults()
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.overrideEmptyByGlobalConfigs()

	return &cfg, nil
}

func (c *Config) withDefaults() {
	if c == nil {
		return
	}

	topology := &c.Tarantool.Topology
	topology.ReadOnly = true
	topology.User = defaultUser
	topology.Password = defaultPassword
}

func (c *Config) overrideEmptyByGlobalConfigs() {
	overrideFn := func(instCfg *InstanceConfig) {
		globalCfg := c.Tarantool

		if instCfg.ConnectTimeout == nil {
			v := globalCfg.ConnectTimeout
			instCfg.ConnectTimeout = &v
		}
		if instCfg.RequestTimeout == nil {
			v := globalCfg.RequestTimeout
			instCfg.RequestTimeout = &v
		}
		if instCfg.User == nil {
			v := globalCfg.Topology.User
			instCfg.User = &v
		}
		if instCfg.Password == nil {
			v := globalCfg.Topology.Password
			instCfg.Password = &v
		}

		// The last hope: set hardcoded predefined values.
		instCfg.withDefaults()
	}

	for clusterUUID, clusterCfg := range c.Clusters {
		if clusterCfg.ReadOnly == nil {
			v := c.Tarantool.Topology.ReadOnly
			clusterCfg.ReadOnly = &v
		}

		for shardUUID, shard := range clusterCfg.Shards {
			for i := 0; i < len(shard); i++ {
				shardCfg := &shard[i]
				overrideFn(shardCfg)
			}
			clusterCfg.Shards[shardUUID] = shard
		}

		for i := 0; i < len(clusterCfg.Routers); i++ {
			routerCfg := &clusterCfg.Routers[i]
			overrideFn(routerCfg)
		}

		c.Clusters[clusterUUID] = clusterCfg
	}
}
