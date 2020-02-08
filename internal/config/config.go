package config

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/shmel1k/qumomf/internal/tarantool"
	"github.com/shmel1k/qumomf/pkg/vshard"
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
	Shards  map[string][]tarantool.ShardConfig `yaml:"shards"`
	Routers []tarantool.ShardConfig            `yaml:"routers"`
}

func (c *Config) ToShardingConfig() vshard.ShardingConfig {
	var res vshard.ShardingConfig
	for k, v := range c.Shards {
		var r vshard.ReplicasetConfig
		for _, vv := range v {
			r.Replicas[vv.UUID] = vshard.ReplicaConfig{
				Name:   vv.Name,
				Master: vv.Master,
				URI:    vshard.PrepareURI(vv.User, vv.Password, vv.Addr),
			}
		}
		res.Shards[k] = r
	}

	return res
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
