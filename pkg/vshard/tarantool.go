package vshard

import (
	"time"

	"github.com/viciious/go-tarantool"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultRequestTimeout = 5 * time.Second
	defaultPriority       = 1
)

type InstanceConfig struct {
	Name           string        `yaml:"name"`
	Addr           string        `yaml:"addr"`
	UUID           string        `yaml:"uuid"`
	User           string        `yaml:"user"`
	Password       string        `yaml:"password"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	Priority       int           `yaml:"priority"`
	Master         bool          `yaml:"master"`
}

func (c *InstanceConfig) withDefaults() {
	if c == nil {
		c = &InstanceConfig{}
	}

	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaultConnectTimeout
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaultRequestTimeout
	}
	if c.Priority == 0 {
		c.Priority = defaultPriority
	}
}

func setupConnection(c InstanceConfig) *Connector {
	c.withDefaults()
	cfg := &tarantool.Options{
		ConnectTimeout: c.ConnectTimeout,
		QueryTimeout:   c.RequestTimeout,
		Password:       c.Password,
		User:           c.User,
		UUID:           c.UUID,
	}

	conn := tarantool.New(c.Addr, cfg)
	return &Connector{
		cfg:  c,
		conn: conn,
	}
}
